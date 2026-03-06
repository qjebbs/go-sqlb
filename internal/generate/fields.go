package generate

import (
	"go/ast"
	"go/types"
	"log"
	"reflect"
	"strings"

	"github.com/qjebbs/go-sqlb/internal/tag/syntax"
	"golang.org/x/tools/go/packages"
)

// FieldInfo is a helper struct to pass field information around.
// It can represent a field from either AST or go/types.
type FieldInfo struct {
	Name        string
	Tag         string
	IsAnonymous bool
	IsExported  bool
	Type        interface{} `json:"-"` // Can be ast.Expr or types.Type
}

func findFields(pkg *packages.Package, parent *Node, fields []FieldInfo) *Node {
	for i := range fields {
		field := &fields[i]

		var info *syntax.Info
		if field.Tag != "" {
			tagVal := reflect.StructTag(strings.Trim(field.Tag, "`")).Get("sqlb")
			if tagVal == "-" {
				continue
			}
			if tagVal != "" {
				parsed, err := syntax.Parse(tagVal)
				if err != nil {
					log.Fatalf("failed to parse tag: %v", err)
				}
				info = parsed
			}
		}

		isAnonymous := field.IsAnonymous
		isPtr, fieldType, importPath, typeObj := resolveTypeInfo(pkg, field.Type)
		node := &Node{
			Name:        field.Name,
			Conf:        info,
			IsPtr:       isPtr,
			FieldType:   fieldType,
			IsAnonymous: isAnonymous,
			IsExported:  field.IsExported,
			ImportPath:  importPath,
		}

		if !isAnonymous && !field.IsExported {
			continue
		}
		parent.AddChild(node)
		if typeObj != nil {
			underlyingStruct := findUnderlyingStruct(typeObj.Type())
			if underlyingStruct != nil {
				var embeddedFields []FieldInfo
				for i := 0; i < underlyingStruct.NumFields(); i++ {
					f := underlyingStruct.Field(i)
					embeddedFields = append(embeddedFields, FieldInfo{
						Name:        f.Name(),
						Tag:         underlyingStruct.Tag(i),
						IsAnonymous: f.Anonymous(),
						IsExported:  f.Exported(),
						Type:        f.Type(),
					})
				}
				findFields(pkg, node, embeddedFields)
			}
		}
	}
	return parent
}

func resolveTypeInfo(pkg *packages.Package, typeExpr interface{}) (isPtr bool, typeStr, importPath string, typeObj types.Object) {
	qualifier := func(p *types.Package) string {
		if p.Path() == pkg.Types.Path() {
			return ""
		}
		return p.Name()
	}

	var currentType types.Type
	switch t := typeExpr.(type) {
	case ast.Expr:
		currentType = pkg.TypesInfo.TypeOf(t)
		if ptr, ok := t.(*ast.StarExpr); ok {
			isPtr = true
			currentType = pkg.TypesInfo.TypeOf(ptr.X)
		}
	case types.Type:
		currentType = t
		if ptr, ok := t.(*types.Pointer); ok {
			isPtr = true
			currentType = ptr.Elem()
		}
	default:
		return
	}

	if currentType == nil {
		return
	}

	typeStr = types.TypeString(currentType, qualifier)

	if named, ok := currentType.(*types.Named); ok {
		typeObj = named.Obj()
		if typeObj != nil && typeObj.Pkg() != nil {
			typePkg := typeObj.Pkg().Path()
			if typePkg != pkg.Types.Path() {
				importPath = typePkg
			}
		}
	}
	return
}
