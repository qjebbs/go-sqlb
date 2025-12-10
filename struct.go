package sqlb

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/qjebbs/go-sqlf/v4"
)

// SelectBuilder is the interface for builders that support Select method.
type SelectBuilder interface {
	Builder
	SetSelect(columns ...sqlf.Builder)
}

// QueryStruct queries the built query and scans rows into a slice of structs.
func QueryStruct[T any](db QueryAble, query SelectBuilder, style sqlf.BindStyle) ([]T, error) {
	queryStr, args, fieldIndices, err := buildQueryForStruct[T](query, style)
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

// BuildQueryForStruct builds the query for struct T and returns the query string and args.
func BuildQueryForStruct[T any](b SelectBuilder, style sqlf.BindStyle) (query string, args []any, err error) {
	query, args, _, err = buildQueryForStruct[T](b, style)
	return query, args, err
}

func buildQueryForStruct[T any](b SelectBuilder, style sqlf.BindStyle) (query string, args []any, fieldIndices [][]int, err error) {
	var zero T
	info, err := getStructInfo(zero)
	if err != nil {
		return "", nil, nil, err
	}
	b.SetSelect(info.columns...)
	query, args, err = b.BuildQuery(style)
	if err != nil {
		return "", nil, nil, err
	}
	return query, args, info.fieldIndices, nil
}

type structInfo struct {
	columns      []sqlf.Builder
	fieldIndices [][]int
	err          error
}

var structCache sync.Map

func getStructInfo(zero any) (*structInfo, error) {
	typ := reflect.TypeOf(zero)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct got %T", zero)
	}

	cached, found := structCache.Load(typ)
	if found {
		info := cached.(*structInfo)
		return info, info.err
	}
	info := parseStructInfo(typ, zero)
	structCache.Store(typ, info)
	return info, info.err
}

func parseStructInfo(typ reflect.Type, zero any) *structInfo {
	var fieldIndices [][]int
	var columns []sqlf.Builder
	var findFields func(t reflect.Type, basePath []int) error
	findFields = func(t reflect.Type, basePath []int) error {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			currentPath := append(basePath, i)
			fieldType := field.Type
			if !field.IsExported() {
				continue
			}
			if field.Anonymous {
				if fieldType.Kind() == reflect.Ptr {
					fieldType = fieldType.Elem()
				}
				if fieldType.Kind() == reflect.Struct {
					err := findFields(fieldType, currentPath)
					if err != nil {
						return err
					}
					continue
				}
			}

			tag := field.Tag.Get("sqlb")
			if tag != "" {
				column, err := parseTag(tag)
				if err != nil {
					return err
				}
				columns = append(columns, column)
				fieldIndices = append(fieldIndices, currentPath)
			}
		}
		return nil
	}

	err := findFields(typ, nil)
	if err != nil {
		return &structInfo{err: err}
	}

	if len(columns) == 0 {
		return &structInfo{
			err: fmt.Errorf("no fields with 'sqlb' tag found in struct %T", zero),
		}
	}

	return &structInfo{
		columns:      columns,
		fieldIndices: fieldIndices,
		err:          nil,
	}
}
