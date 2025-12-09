package sqlb

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/qjebbs/go-sqlf/v4"
)

// QueryStruct queries the built query and scans rows into a slice of structs.
func QueryStruct[T any](db QueryAble, query *QueryBuilder, style sqlf.BindStyle) ([]T, error) {
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
			if field.Kind() == reflect.Ptr && field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
				field = field.Elem()
			}
			fields[i] = field.Addr().Interface()
		}
		return dest, fields
	})
}

// BuildQueryForStruct builds the query for struct T and returns the query string and args.
func BuildQueryForStruct[T any](b *QueryBuilder, style sqlf.BindStyle) (query string, args []any, err error) {
	query, args, _, err = buildQueryForStruct[T](b, style)
	return query, args, err
}

func buildQueryForStruct[T any](b *QueryBuilder, style sqlf.BindStyle) (query string, args []any, fieldIndices [][]int, err error) {
	var zero T
	typ := reflect.TypeOf(zero)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return "", nil, nil, fmt.Errorf("expected struct got %T", zero)
	}
	columns := make([]sqlf.Builder, 0)
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
					err = findFields(fieldType, currentPath)
					if err != nil {
						return err
					}
					continue
				}
			}
			tag := getFieldTag(field)
			if tag != "" {
				seg := strings.SplitN(tag, ".", 2)
				if len(seg) != 2 {
					return fmt.Errorf("invalid sqlb tag %q from %T.%s", tag, zero, field.Name)
				}
				// construct a table that reports dependencies correctly,
				// Table reports only applied table name, so the actual name is not important here.
				table := NewTable("", seg[0])
				columns = append(columns, table.Column(seg[1]))
				fieldIndices = append(fieldIndices, currentPath)
			}
		}
		return nil
	}

	err = findFields(typ, nil)
	if err != nil {
		return "", nil, nil, err
	}

	if len(columns) == 0 {
		return "", nil, nil, fmt.Errorf("no fields with 'sqlb' tag found in struct %T", zero)
	}
	b.SelectReplace(columns...)
	query, args, err = b.BuildQuery(style)
	if err != nil {
		return "", nil, nil, err
	}
	return query, args, fieldIndices, err
}

// getFieldTag gets the sqlb tag from a struct field.
func getFieldTag(field reflect.StructField) string {
	return field.Tag.Get("sqlb")
}
