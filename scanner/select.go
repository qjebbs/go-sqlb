package scanner

import (
	"reflect"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlf/v4"
)

var _ SelectBuilder = (*sqlb.QueryBuilder)(nil)

// SelectBuilder is the interface for builders that support Select method.
type SelectBuilder interface {
	sqlb.Builder
	SetSelect(columns ...sqlf.Builder)
}

// Select executes the query and scans the results into a slice of struct T.
func Select[T any](db QueryAble, query SelectBuilder, options ...Option) ([]T, error) {
	opt := mergeOptions(options...)
	queryStr, args, fieldIndices, err := buildQueryForStruct[T](query, opt.style, opt.tags)
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

func buildQueryForStruct[T any](b SelectBuilder, style sqlf.BindStyle, tags []string) (query string, args []any, fieldIndices [][]int, err error) {
	var zero T
	info, err := getStructInfo(zero)
	if err != nil {
		return "", nil, nil, err
	}
	columns, fieldIndices := info.filterByTag(tags)
	b.SetSelect(columns...)
	query, args, err = b.BuildQuery(style)
	if err != nil {
		return "", nil, nil, err
	}
	return query, args, fieldIndices, nil
}
