package mapper

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/qjebbs/go-sqlb/internal/tag/syntax"
	"github.com/qjebbs/go-sqlf/v4"
)

type structInfo struct {
	columns []fieldInfo
	err     error
}

type fieldInfo struct {
	syntax.Info

	Name           string // field name
	Diving         bool   // whether this field is from a 'dive' operation
	InheritedFroms bool   // whether the 'from' value is inherited
	Index          []int  // field index in the struct
}

func (f fieldInfo) NewColumnBuilder() sqlf.Builder {
	return sqlf.Identifier(f.Info.Column)
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
	type context struct {
		table  string
		froms  []string
		diving bool
	}
	var findFields func(t reflect.Type, basePath []int, ctx context) error
	findFields = func(t reflect.Type, basePath []int, ctx context) error {
		curTable := ctx.table
		curFroms := ctx.froms
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			currentPath := append(basePath, i)
			fieldType := field.Type

			var info *syntax.Info
			var inheritedFroms = false
			tag := field.Tag.Get("sqlb")
			if tag == "-" {
				continue
			}
			if tag != "" {
				parsed, err := syntax.Parse(tag)
				if err != nil {
					return fmt.Errorf("sqlb tag: on %T.%s: %q: %w", zero, field.Name, tag, err)
				}
				if len(parsed.From) > 0 {
					curFroms = parsed.From
				} else {
					inheritedFroms = true
					parsed.From = curFroms
				}
				if parsed.Table != "" {
					curTable = parsed.Table
				} else {
					parsed.Table = curTable
				}
				info = parsed
			}
			if field.Anonymous {
				if info != nil {
					if info.Column != "" || len(info.SelectOn) > 0 {
						return fmt.Errorf("sqlb tag: %T.%s: anonymous field supports only the 'tables' key", zero, field.Name)
					}
				}
				if fieldType.Kind() == reflect.Ptr {
					fieldType = fieldType.Elem()
				}
				if fieldType.Kind() == reflect.Struct {
					ctx.table = curTable
					ctx.froms = curFroms
					err := findFields(fieldType, currentPath, ctx)
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
					ctx.table = curTable
					ctx.froms = curFroms
					ctx.diving = ctx.diving || info.Dive
					err := findFields(fieldType, currentPath, ctx)
					if err != nil {
						return err
					}
					continue
				}
				if info.Column == "" && info.Select == "" {
					continue
				}
				columns = append(columns, fieldInfo{
					Info:           *info,
					Name:           field.Name,
					InheritedFroms: inheritedFroms,
					Index:          currentPath,
				})
			}
		}
		return nil
	}

	err := findFields(typ, nil, context{})
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
