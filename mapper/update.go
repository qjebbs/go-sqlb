package mapper

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/internal/dialects"
	"github.com/qjebbs/go-sqlf/v4"
)

var _ UpdateBuilder = (*sqlb.UpdateBuilder)(nil)

// UpdateBuilder is the interface for builders that support Update method.
type UpdateBuilder interface {
	sqlb.Builder
	SetDialect(d dialects.Dialect)
	SetUpdate(table string)
	SetSets(sets ...sqlf.Builder)
	AppendWhere(conditions sqlf.Builder)
}

// Update updates a single struct into the database.
//
// The struct tag syntax is: `key[:value][;key[:value]]...`, e.g. `sqlb:"pk;col:id;table:users;"`
//
// The supported struct tags are:
//   - table: [Inheritable] Declare base table for the current field and its sub-fields / subsequent sibling fields.
//   - col: The column associated with the field.
//   - noupdate: The field is excluded from UPDATE statement.
//   - pk: The column is primary key, which will be used in WHERE clause to locate the row.
//   - match: The column will be always included in WHERE clause if it is not zero value.
//
// If no `pk` field is defined or set, Update() will return an error to avoid accidental full-table update.
func Update[T any](db QueryAble, b UpdateBuilder, value T, options ...Option) error {
	if err := checkStruct(value); err != nil {
		return err
	}
	opt := mergeOptions(options...)
	queryStr, args, err := buildUpdateQueryForStruct(b, value, opt)
	if err != nil {
		return err
	}
	_, err = db.Exec(queryStr, args...)
	return err
}

func buildUpdateQueryForStruct[T any](b UpdateBuilder, value T, opt *Options) (query string, args []any, err error) {
	if opt == nil {
		opt = newDefaultOptions()
	}
	b.SetDialect(opt.dialect)
	var zero T
	info, err := getStructInfo(zero)
	if err != nil {
		return "", nil, err
	}
	updateInfo, err := buildUpdateInfo(opt.dialect, info, value)
	if err != nil {
		return "", nil, err
	}

	if updateInfo.table != "" {
		// don't override with empty table in case the table is set manually
		b.SetUpdate(updateInfo.table)
	}
	sets := make([]sqlf.Builder, len(updateInfo.updateColumns))
	for i, coldata := range updateInfo.updateColumns {
		sets[i] = sqlf.F("? = ?", sqlf.F(coldata.ColumnIndent), coldata.Value)
	}
	b.SetSets(sets...)

	b.AppendWhere(sqlf.F("? = ?", sqlf.F(updateInfo.pk.ColumnIndent), updateInfo.pk.Value))
	for _, coldata := range updateInfo.matchColumns {
		b.AppendWhere(sqlf.F("? = ?", sqlf.F(coldata.ColumnIndent), coldata.Value))
	}

	query, args, err = b.BuildQuery(opt.style)
	if err != nil {
		return "", nil, err
	}
	return query, args, nil
}

type updateInfo struct {
	table string

	pk            fieldData
	updateColumns []fieldData
	matchColumns  []fieldData
}

type fieldData struct {
	Info         fieldInfo
	ColumnIndent string
	Value        any
}

func buildUpdateInfo[T any](dialect dialects.Dialect, f *structInfo, value T) (*updateInfo, error) {
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
		colValue, ok := getValueAtIndex(col.Index, valueVal)
		if !ok {
			return nil, fmt.Errorf("cannot get value for column %s", col.Column)
		}
		data := fieldData{
			ColumnIndent: colIndent,
			Info:         col,
			Value:        colValue,
		}
		switch {
		case col.PK:
			if r.pk.ColumnIndent != "" {
				return nil, errors.New("multiple primary key columns defined for update")
			}
			r.pk = data
		case col.Match:
			r.matchColumns = append(r.matchColumns, data)
		case col.NoUpdate:
			// skip
		default:
			r.updateColumns = append(r.updateColumns, data)
		}
	}
	if r.pk.ColumnIndent == "" {
		return nil, errors.New("no primary key defined for update")
	}
	if len(r.updateColumns) == 0 {
		return nil, errors.New("no updatable columns found for update")
	}
	return &r, nil
}

func getValueAtIndex(dest []int, v reflect.Value) (any, bool) {
	current, ok := getReflectValueAtIndex(dest, v)
	if !ok {
		return nil, false
	}
	return current.Interface(), true
}

func getReflectValueAtIndex(dest []int, v reflect.Value) (reflect.Value, bool) {
	current := v
	for _, idx := range dest {
		if current.Kind() == reflect.Ptr {
			if current.IsNil() {
				return reflect.Value{}, false
			}
			current = current.Elem()
		}
		current = current.Field(idx)
	}
	return current, true
}
