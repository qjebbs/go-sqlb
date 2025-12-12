package scanner

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/internal/tag/syntax"
	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

type structInfo struct {
	columns []fieldInfo
	err     error
}

type fieldInfo struct {
	column sqlf.Builder
	index  []int
	tags   map[string]struct{}
}

func (f structInfo) filterByTag(tags []string) (columns []sqlf.Builder, fieldIndices [][]int) {
	for _, col := range f.columns {
		if col.tags == nil {
			columns = append(columns, col.column)
			fieldIndices = append(fieldIndices, col.index)
			continue
		}
		for _, tag := range tags {
			if _, ok := col.tags[tag]; ok {
				columns = append(columns, col.column)
				fieldIndices = append(fieldIndices, col.index)
				break
			}
		}
	}
	return
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
	var columns []fieldInfo
	var findFields func(t reflect.Type, basePath []int, declaredTables []string) error
	findFields = func(t reflect.Type, basePath []int, declaredTables []string) error {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			currentPath := append(basePath, i)
			fieldType := field.Type
			if !field.IsExported() {
				continue
			}

			curDeclaredTables := declaredTables
			tag := field.Tag.Get("sqlb")
			if field.Anonymous {
				if tag != "" {
					info, err := syntax.Parse(tag)
					if err != nil {
						return fmt.Errorf("sqlb tag: on %T.%s: %q: %w", zero, field.Name, tag, err)
					}
					if info.Column != "" || len(info.On) > 0 {
						return fmt.Errorf("sqlb tag: cannot declare column or on filters on anonymous field: %T.%s", zero, field.Name)
					}
					curDeclaredTables = info.Tables
				}
				if fieldType.Kind() == reflect.Ptr {
					fieldType = fieldType.Elem()
				}
				if fieldType.Kind() == reflect.Struct {
					err := findFields(fieldType, currentPath, curDeclaredTables)
					if err != nil {
						return err
					}
					continue
				}
			}

			if tag != "" {
				info, err := syntax.Parse(tag)
				if err != nil {
					return fmt.Errorf("sqlb tag: column definition on %T.%s: %q : %w", zero, field.Name, tag, err)
				}
				useDeclaredTables := false
				if len(info.Tables) == 0 {
					useDeclaredTables = true
					info.Tables = declaredTables
				}
				tables := util.Map(info.Tables, func(t string) any {
					return sqlb.NewTable("", t)
				})
				column := sqlf.F(info.Column, tables...)
				if useDeclaredTables {
					column.NoUsageCheck()
				}
				var tags map[string]struct{}
				if len(info.On) > 0 {
					tags = make(map[string]struct{})
					for _, tag := range info.On {
						tags[tag] = struct{}{}
					}
				}
				columns = append(columns, fieldInfo{
					column: column,
					index:  currentPath,
					tags:   tags,
				})
			}
		}
		return nil
	}

	err := findFields(typ, nil, nil)
	if err != nil {
		return &structInfo{err: err}
	}

	if len(columns) == 0 {
		return &structInfo{
			err: fmt.Errorf("no fields with 'sqlb' tag found in struct %T", zero),
		}
	}

	return &structInfo{
		columns: columns,
		err:     nil,
	}
}
