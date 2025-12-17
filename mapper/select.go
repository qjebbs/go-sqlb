package mapper

import (
	"database/sql"
	"reflect"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

var _ SelectBuilder = (*sqlb.SelectBuilder)(nil)
var _ SelectLimitBuilder = (*sqlb.SelectBuilder)(nil)

// SelectBuilder is the interface for builders that support Select method.
type SelectBuilder interface {
	sqlb.Builder
	SetSelect(columns ...sqlf.Builder)
}

// SelectLimitBuilder is the interface for builders that support Limit method.
type SelectLimitBuilder interface {
	SelectBuilder
	SetLimit(n int64)
}

// SelectOne executes the query and scans the result into a struct T.
func SelectOne[T any](db QueryAble, b SelectLimitBuilder, options ...Option) (T, error) {
	b.SetLimit(1)
	r, err := Select[T](db, b, options...)
	if err != nil {
		var zero T
		return zero, err
	}
	if len(r) == 0 {
		var zero T
		return zero, sql.ErrNoRows
	}
	return r[0], nil
}

// Select executes the query and scans the results into a slice of struct T.
func Select[T any](db QueryAble, b SelectBuilder, options ...Option) ([]T, error) {
	opt := mergeOptions(options...)
	queryStr, args, fieldIndices, err := buildSelectQueryForStruct[T](b, opt)
	if err != nil {
		return nil, err
	}
	return scan(db, queryStr, args, func() (T, []any) {
		var dest T
		return prepareScanDestinations(dest, fieldIndices)
	})
}

// prepareScanDestinations prepares the destinations for scanning the query results into the struct fields.
// !!! MUST return dest since the param 'dest' and the caller 'dest' is different variable.
// prepareScanDestinations will create new instances for nil pointer fields as needed which
// affects only the param 'dest'.
func prepareScanDestinations[T any](dest T, fieldIndices [][]int) (T, []any) {
	destValue := reflect.ValueOf(&dest).Elem()
	if destValue.Kind() == reflect.Ptr {
		if destValue.IsNil() {
			destValue.Set(reflect.New(destValue.Type().Elem()))
		}
		destValue = destValue.Elem()
	}
	fields := make([]any, len(fieldIndices))
	for i, indexPath := range fieldIndices {
		field := destValue.FieldByIndex(indexPath)
		if i < len(indexPath)-1 && field.Kind() == reflect.Ptr && field.IsNil() {
			// if any ancestor field is a pointer and nil, create a new instance.
			field.Set(reflect.New(field.Type().Elem()))
			field = field.Elem()
		}
		fields[i] = field.Addr().Interface()
	}
	return dest, fields
}

func buildSelectQueryForStruct[T any](b SelectBuilder, opt *Options) (query string, args []any, fieldIndices [][]int, err error) {
	if opt == nil {
		opt = newDefaultOptions()
	}
	var zero T
	info, err := getStructInfo(zero)
	if err != nil {
		return "", nil, nil, err
	}
	columns, fieldIndices := buildSelectInfo(opt.dialect, opt.tags, info)
	b.SetSelect(columns...)
	query, args, err = b.BuildQuery(opt.style)
	if err != nil {
		return "", nil, nil, err
	}
	return query, args, fieldIndices, nil
}

func buildSelectInfo(dialect Dialect, tags []string, f *structInfo) (columns []sqlf.Builder, fieldIndices [][]int) {
	for _, col := range f.columns {
		included := len(col.On) == 0
		if !included && len(tags) > 0 {
			for _, tag := range tags {
				if util.Index(col.On, tag) >= 0 {
					included = true
					break
				}
			}
		}
		if !included {
			continue
		}
		// sel tag takes precedence over col tag
		checkUsage := col.CheckUsage
		expr := col.Select
		if expr == "" && col.Column != "" && len(col.Tables) > 0 {
			checkUsage = false
			expr = "?." + dialect.QuoteIdentifier(col.Column)
		}
		column := sqlf.F(expr, util.Map(col.Tables, func(t string) any {
			return sqlb.NewTable("", t)
		})...)
		if !checkUsage {
			column.NoUsageCheck()
		}
		columns = append(columns, column)
		fieldIndices = append(fieldIndices, col.Index)
	}
	return
}
