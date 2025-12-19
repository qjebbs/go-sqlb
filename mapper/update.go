package mapper

import (
	"errors"
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

// Update updates a single struct in the database.
func Update[T any](db QueryAble, b UpdateBuilder, value T, options ...Option) error {
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
	updateInfo := buildUpdateInfo(opt.dialect, info)

	if len(updateInfo.matchColumns) == 0 {
		return "", nil, errors.New("no primary key defined for update")
	}
	if len(updateInfo.updateColumns) == 0 {
		return "", nil, errors.New("no updatable columns found for update")
	}

	if updateInfo.table != "" {
		// don't override with empty table in case the table is set manually
		b.SetUpdate(updateInfo.table)
	}
	valueVal := reflect.ValueOf(value)
	if valueVal.Kind() == reflect.Ptr {
		valueVal = valueVal.Elem()
	}
	sets := make([]sqlf.Builder, len(updateInfo.updateColumns))
	for i, fieldIndex := range updateInfo.updateIndices {
		val := valueVal.FieldByIndex(fieldIndex).Interface()
		sets[i] = sqlf.F("? = ?", sqlf.F(updateInfo.updateColumns[i]), val)
	}
	b.SetSets(sets...)

	for i, fieldIndex := range updateInfo.matchIndices {
		val := valueVal.FieldByIndex(fieldIndex).Interface()
		b.AppendWhere(sqlf.F("? = ?", sqlf.F(updateInfo.matchColumns[i]), val))
	}

	query, args, err = b.BuildQuery(opt.style)
	if err != nil {
		return "", nil, err
	}
	return query, args, nil
}

type updateInfo struct {
	table string

	updateColumns []string
	updateIndices [][]int

	matchColumns []string
	matchIndices [][]int
}

func buildUpdateInfo(dialect dialects.Dialect, f *structInfo) updateInfo {
	var r updateInfo
	for _, col := range f.columns {
		if r.table == "" && !col.Diving && col.Table != "" {
			// respect first non-diving column with table specified
			r.table = col.Table
		}
		if col.Column == "" {
			continue
		}
		colIndent := dialect.QuoteIdentifier(col.Column)
		if col.PK || col.Match {
			r.matchColumns = append(r.matchColumns, colIndent)
			r.matchIndices = append(r.matchIndices, col.Index)
			continue
		}
		r.updateColumns = append(r.updateColumns, colIndent)
		r.updateIndices = append(r.updateIndices, col.Index)
	}
	return r
}
