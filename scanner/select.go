package scanner

import (
	"database/sql"
	"reflect"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlf/v4"
)

var _ SelectBuilder = (*sqlb.QueryBuilder)(nil)
var _ SelectLimitBuilder = (*sqlb.QueryBuilder)(nil)

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
	queryStr, args, fieldIndices, err := buildQueryForStruct[T](b, opt)
	if err != nil {
		return nil, err
	}
	return scan(db, queryStr, args, func() (T, []any) {
		var dest T
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
	})
}

func buildQueryForStruct[T any](b SelectBuilder, opt *Options) (query string, args []any, fieldIndices [][]int, err error) {
	if opt == nil {
		opt = newDefaultOptions()
	}
	var zero T
	info, err := getStructInfo(zero)
	if err != nil {
		return "", nil, nil, err
	}
	columns, fieldIndices := info.build(opt.dialect, opt.tags)
	b.SetSelect(columns...)
	query, args, err = b.BuildQuery(opt.style)
	if err != nil {
		return "", nil, nil, err
	}
	return query, args, fieldIndices, nil
}
