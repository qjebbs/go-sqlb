package mapper

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/qjebbs/go-sqlb/tag"
	"github.com/qjebbs/go-sqlf/v4"
)

type structInfo struct {
	columns        []fieldInfo
	uniqueTable    bool
	topHasModelTag bool

	err error
}

type fieldInfo struct {
	tag.Info

	Name   string       // field name
	Type   reflect.Type // field type
	Diving bool         // whether this field is from a 'dive' operation
	Index  []int        // field index in the struct
}

func (f fieldInfo) NewColumnBuilder() sqlf.Builder {
	return sqlf.Identifier(f.Info.Column)
}

var structCache sync.Map

func getModelStructInfo(zero any) (*structInfo, error) {
	info, err := getStructInfo(zero)
	if err != nil {
		return nil, err
	}
	if !info.topHasModelTag {
		return nil, fmt.Errorf("refuse to operate a struct without 'model' tag on top-level / first level embedded fields, to avoid unexpected behavior")
	}
	if !info.uniqueTable {
		return nil, fmt.Errorf("multiple tables found in a model struct")
	}
	return info, nil
}

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
	// Inheritable by latter fields, including children and siblings after.
	// Or, global statistics about the struct
	type inheritable struct {
		// Inheritable
		table string

		// Global statistics
		structInfo
	}
	type context struct {
		*inheritable
		topLevel             bool
		directEmbedding      bool
		directEmbeddingLevel int
		diving               bool
	}
	var findFields func(t reflect.Type, basePath []int, ctx context) error
	findFields = func(t reflect.Type, basePath []int, ctx context) error {
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			currentPath := append(basePath, i)

			var info *tag.Info
			tagVal := field.Tag.Get("sqlb")
			if tagVal == "-" {
				continue
			}
			if tagVal != "" {
				parsed, err := tag.Parse(tagVal)
				if err != nil {
					return fmt.Errorf("sqlb tag: on %T.%s: %q: %w", zero, field.Name, tagVal, err)
				}
				if parsed.Table != "" {
					if ctx.table != "" && ctx.table != parsed.Table {
						ctx.uniqueTable = false
					}
					ctx.table = parsed.Table
				} else {
					parsed.Table = ctx.table
				}
				if parsed.Model && (ctx.topLevel || ctx.directEmbeddingLevel == 1) {
					ctx.topHasModelTag = true
				}
				info = parsed
			}
			if field.Anonymous {
				if field.Type.Kind() == reflect.Ptr {
					field.Type = field.Type.Elem()
				}
				if field.Type.Kind() == reflect.Struct {
					newCtx := ctx
					if ctx.topLevel || ctx.directEmbedding {
						newCtx.directEmbedding = true
						newCtx.directEmbeddingLevel++
					}
					newCtx.topLevel = false
					err := findFields(field.Type, currentPath, newCtx)
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
					if field.Type.Kind() == reflect.Ptr {
						field.Type = field.Type.Elem()
					}
					if field.Type.Kind() != reflect.Struct {
						return fmt.Errorf("sqlb tag: column definition on %T.%s: 'dive' can be used only with struct fields", zero, field.Name)
					}
					newCtx := ctx
					newCtx.diving = true
					newCtx.topLevel = false
					err := findFields(field.Type, currentPath, newCtx)
					if err != nil {
						return err
					}
					continue
				}
				if info.Column == "" && info.Select == "" {
					continue
				}
				ctx.columns = append(ctx.columns, fieldInfo{
					Info:  *info,
					Name:  field.Name,
					Type:  field.Type,
					Index: currentPath,
				})
			}
		}
		return nil
	}

	ctx := context{
		inheritable: &inheritable{
			structInfo: structInfo{
				uniqueTable: true,
			},
		},
		topLevel: true,
	}
	err := findFields(typ, nil, ctx)
	if err != nil {
		return &structInfo{err: err}
	}

	if len(ctx.columns) == 0 {
		ctx.structInfo.err = fmt.Errorf("no fields with 'sqlb' tag found in struct %T", zero)
	}

	return &ctx.structInfo
}
