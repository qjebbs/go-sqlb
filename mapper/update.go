package mapper

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/internal/dialects"
	"github.com/qjebbs/go-sqlf/v4"
)

// Update updates a single struct into the database.
//
// Update omits zero-value fields by default. To force do a full update including zero-value fields,
// use WithUpdateAll() option.
//
// The struct tag syntax is: `key[:value][;key[:value]]...`, e.g. `sqlb:"pk;col:id;table:users;"`
//
// The supported struct tags are:
//   - table: [Inheritable] Declare base table for the current field and its sub-fields / subsequent sibling fields.
//   - col: The column associated with the field.
//   - pk: The column is primary key, which will be used in WHERE clause to locate the row.
//   - match: The column will be always included in WHERE clause even if it is zero value.
//   - readonly: The field is excluded from UPDATE statement.
//
// If no `pk` field is defined, Update() will return an error to avoid accidental full-table update.
func Update[T any](db QueryAble, value T, options ...Option) error {
	return wrapErrWithDebugName("Update", value, update(db, value, true, options...))
}

// Patch is similar to Update(), but it only updates non-zero fields of the struct.
//
// See Update() for more details.
func Patch[T any](db QueryAble, value T, options ...Option) error {
	return wrapErrWithDebugName("Update", value, update(db, value, false, options...))
}

func update[T any](db QueryAble, value T, updateAll bool, options ...Option) error {
	if err := checkStruct(value); err != nil {
		return err
	}
	opt := mergeOptions(options...)
	queryStr, args, err := buildUpdateQueryForStruct(value, updateAll, opt)
	if err != nil {
		return err
	}
	if db == nil {
		return ErrNilDB
	}
	r, err := db.Exec(queryStr, args...)
	if err != nil {
		return err
	}
	rowsAffected, err := r.RowsAffected()
	if err != nil {
		return nil
	}
	if rowsAffected == 0 {
		return errors.New("no rows updated")
	}
	if rowsAffected > 1 {
		return fmt.Errorf("unexpectedly updated %d rows, wrong 'pk'?", rowsAffected)
	}
	return err
}

func buildUpdateQueryForStruct[T any](value T, updateAll bool, opt *Options) (query string, args []any, err error) {
	if opt == nil {
		opt = newDefaultOptions()
	}
	b := sqlb.NewUpdateBuilder(opt.dialect)
	if opt.debug {
		b.Debug(debugName("Update", value))
	}

	var zero T
	info, err := getStructInfo(zero)
	if err != nil {
		return "", nil, err
	}
	updateInfo, err := buildUpdateInfo(opt.dialect, info, updateAll, opt, value)
	if err != nil {
		return "", nil, err
	}

	if updateInfo.table != "" {
		// don't override with empty table in case the table is set manually
		b.Update(updateInfo.table)
	}
	for _, coldata := range updateInfo.updateColumns {
		b.Set(coldata.Indent, coldata.Value)
	}

	b.AppendWhere(eqOrIsNull(updateInfo.pk.Indent, updateInfo.pk.Value))
	for _, coldata := range updateInfo.matchColumns {
		b.AppendWhere(eqOrIsNull(coldata.Indent, coldata.Value))
	}

	query, args, err = b.BuildQuery(opt.style)
	if err != nil {
		return "", nil, err
	}
	return query, args, nil
}

func eqOrIsNull(column string, value any) sqlf.Builder {
	if value == nil {
		return sqlf.F("? IS NULL", sqlf.F(column))
	}
	return sqlf.F("? = ?", sqlf.F(column), value)
}

type updateInfo struct {
	table string

	pk            fieldData
	updateColumns []fieldData
	matchColumns  []fieldData
}

type fieldData struct {
	Info   fieldInfo
	Indent string
	Value  any
	IsZero bool
}

func buildUpdateInfo[T any](dialect dialects.Dialect, f *structInfo, updateAll bool, opt *Options, value T) (*updateInfo, error) {
	valueVal := reflect.ValueOf(value)
	if valueVal.Kind() == reflect.Ptr {
		valueVal = valueVal.Elem()
	}
	var r updateInfo
	for _, col := range f.columns {
		if col.Diving {
			continue
		}
		if r.table == "" && col.Table != "" {
			// respect first column with table specified
			r.table = col.Table
		}
		if col.Column == "" {
			continue
		}
		colIndent := dialect.QuoteIdentifier(col.Column)
		colValue, iszero, ok := getValueAtIndex(col.Index, valueVal)
		if !ok {
			return nil, fmt.Errorf("cannot get value for column %s", col.Column)
		}
		data := fieldData{
			Info:   col,
			Indent: colIndent,
			Value:  colValue,
			IsZero: iszero,
		}
		switch {
		case col.PK:
			if r.pk.Indent != "" {
				return nil, errors.New("multiple primary key columns defined for update")
			}
			r.pk = data
		case col.Match:
			r.matchColumns = append(r.matchColumns, data)
		case col.ReadOnly || (iszero && !updateAll):
			// skip
		default:
			r.updateColumns = append(r.updateColumns, data)
		}
	}
	if r.pk.Indent == "" {
		return nil, errors.New("no primary key defined for update")
	}
	if len(r.updateColumns) == 0 {
		return nil, errors.New("no updatable columns found for update")
	}
	return &r, nil
}
