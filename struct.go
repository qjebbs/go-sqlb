package sqlb

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/qjebbs/go-sqlb/internal/util"
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
			tag := getFieldTag(field)
			if tag != "" {
				// tag formats:
				// 1. <table>.<column>
				// 2. <expression>;[table1, table2...]
				// e.g. "u.id", "COALESCE(?.id,?.user_id,0);u,j"
				if tableName, columnName, ok := parseSimpleTag(tag); ok {
					table := NewTable("", tableName)
					column := table.Column(columnName)
					columns = append(columns, column)
				} else {
					seg := strings.SplitN(tag, ";", 2)
					var tables []any
					if len(seg) == 2 {
						tableNames := strings.Split(seg[1], ",")
						tables = util.Map(tableNames, func(t string) any {
							return NewTable("", strings.TrimSpace(t))
						})
					}
					column := sqlf.F(seg[0], tables...)
					// try build column to catch errors early for better error messages
					ctx := sqlf.NewContext(sqlf.BindStyleDollar)
					_, err := column.Build(ctx)
					if err != nil {
						return fmt.Errorf("invalid sqlb tag %q from %T.%s: %w", tag, zero, field.Name, err)
					}
					columns = append(columns, column)
				}
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

// parseSimpleTag parses a simple sqlb tag in the format of "<table>.<column>".
func parseSimpleTag(tag string) (tableName, columnName string, ok bool) {
	if strings.Contains(tag, ";") {
		return "", "", false
	}
	seg := strings.SplitN(tag, ".", 2)
	if len(seg) != 2 {
		return "", "", false
	}
	tableName = seg[0]
	columnName = seg[1]
	if tableName == "" || columnName == "" {
		return "", "", false
	}
	if !isAllowedName(tableName) || !isAllowedName(columnName) {
		return "", "", false
	}
	return tableName, columnName, true
}

// isAllowedName checks if characters of name are [a-zA-Z0-9_@#$] and not starting with a digit.
func isAllowedName(name string) bool {
	for i, ch := range name {
		if !(ch >= 'a' && ch <= 'z' ||
			ch >= 'A' && ch <= 'Z' ||
			ch >= '0' && ch <= '9' ||
			ch == '_' || ch == '@' || ch == '#') {
			return false
		}
		if i == 0 && ch >= '0' && ch <= '9' {
			return false
		}
	}
	return true
}

// getFieldTag gets the sqlb tag from a struct field.
func getFieldTag(field reflect.StructField) string {
	return field.Tag.Get("sqlb")
}
