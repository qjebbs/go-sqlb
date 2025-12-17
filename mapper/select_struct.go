package mapper

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
	selected   string              // select expression
	column     string              // column definition
	tables     []string            // tables to use for this column
	checkUsage bool                // whether to do usage check for tables inherited from anonymous fields
	index      []int               // field index in the struct
	tags       map[string]struct{} // tags for including this field
}

func (f structInfo) build(dialect Dialect, tags []string) (columns []sqlf.Builder, fieldIndices [][]int) {
	for _, col := range f.columns {
		included := col.tags == nil
		if !included && len(tags) > 0 {
			for _, tag := range tags {
				if _, ok := col.tags[tag]; ok {
					included = true
					break
				}
			}
		}
		if !included {
			continue
		}
		// sel tag takes precedence over col tag
		checkUsage := col.checkUsage
		expr := col.selected
		if expr == "" && col.column != "" && len(col.tables) > 0 {
			checkUsage = false
			if isReservedWord(dialect, col.column) {
				col.column = `"` + col.column + `"`
			}
			expr = "?." + col.column
		}
		column := sqlf.F(expr, util.Map(col.tables, func(t string) any {
			return sqlb.NewTable("", t)
		})...)
		if !checkUsage {
			column.NoUsageCheck()
		}
		columns = append(columns, column)
		fieldIndices = append(fieldIndices, col.index)
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
		curTables := declaredTables
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			currentPath := append(basePath, i)
			fieldType := field.Type

			var info *syntax.Info
			var checkUsage = true
			var tables []string
			if tag := field.Tag.Get("sqlb"); tag != "" {
				parsed, err := syntax.Parse(tag)
				if err != nil {
					return fmt.Errorf("sqlb tag: on %T.%s: %q: %w", zero, field.Name, tag, err)
				}
				if len(parsed.Tables) > 0 {
					checkUsage = true
					tables = parsed.Tables
					curTables = parsed.Tables
				} else {
					checkUsage = false
					tables = curTables
				}
				info = parsed
			}
			if field.Anonymous {
				if info != nil {
					if info.Column != "" || len(info.On) > 0 {
						return fmt.Errorf("sqlb tag: %T.%s: anonymous field supports only the 'tables' key", zero, field.Name)
					}
				}
				if fieldType.Kind() == reflect.Ptr {
					fieldType = fieldType.Elem()
				}
				if fieldType.Kind() == reflect.Struct {
					err := findFields(fieldType, currentPath, curTables)
					if err != nil {
						return err
					}
					continue
				}
			}

			if !field.IsExported() {
				continue
			}

			if info != nil {
				if info.Dive {
					if fieldType.Kind() == reflect.Ptr {
						fieldType = fieldType.Elem()
					}
					if fieldType.Kind() != reflect.Struct {
						return fmt.Errorf("sqlb tag: column definition on %T.%s: 'dive' can be used only with struct fields", zero, field.Name)
					}
					err := findFields(fieldType, currentPath, curTables)
					if err != nil {
						return err
					}
					continue
				}
				if info.Column == "" && info.Select == "" {
					continue
				}

				var tags map[string]struct{}
				if len(info.On) > 0 {
					tags = make(map[string]struct{})
					for _, tag := range info.On {
						tags[tag] = struct{}{}
					}
				}
				columns = append(columns, fieldInfo{
					selected:   info.Select,
					column:     info.Column,
					tables:     tables,
					checkUsage: checkUsage,
					index:      currentPath,
					tags:       tags,
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
