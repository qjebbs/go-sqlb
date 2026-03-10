package generate

import (
	"fmt"
	"go/ast"
	"go/types"
	"reflect"
	"strings"

	"github.com/qjebbs/go-sqlb/tag"
	"golang.org/x/tools/go/packages"
)

// FieldInfo is a helper struct to pass field information around.
// It can represent a field from either AST or go/types.
type FieldInfo struct {
	Name        string
	Tag         string
	IsAnonymous bool
	IsExported  bool
	Type        any `json:"-"` // Can be ast.Expr or types.Type
}

var _ objecter = (*types.Named)(nil)
var _ objecter = (*types.Alias)(nil)

// objecter is an interface that abstracts over types that have an Obj() method returning a *types.TypeName.
// This is used to handle both *types.Named and *types.Alias uniformly when resolving type information.
type objecter interface {
	Obj() *types.TypeName
}

func findFields(pkg *packages.Package, parent *Node, fields []FieldInfo) (*Node, error) {
	for i := range fields {
		field := &fields[i]
		isAnonymous := field.IsAnonymous

		var tagVal string
		var info *tag.Info

		if field.Tag != "" {
			tagVal = reflect.StructTag(strings.Trim(field.Tag, "`")).Get("sqlb")
		}
		if tagVal == "-" {
			continue
		}
		if tagVal == "" {
			if !isAnonymous {
				continue
			}
		} else {
			parsed, err := tag.Parse(tagVal)
			if err != nil {
				return nil, fmt.Errorf("failed to parse tag %q: %w", tagVal, err)
			}
			info = parsed
		}

		// don't not ignore unexported fields even if they are not anonymous,
		// as they might be used to define table names, e.g.:
		// type User struct {
		//     _ struct{} `sqlb:"users"`
		//     ID int
		// }

		isPtr, fieldType, importPath, typeObj, err := resolveTypeInfo(pkg, field.Type)
		if err != nil {
			return nil, err
		}
		node := &Node{
			Name:        field.Name,
			Conf:        info,
			IsPtr:       isPtr,
			FieldType:   fieldType,
			IsAnonymous: isAnonymous,
			IsExported:  field.IsExported,
			ImportPath:  importPath,
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
				_, err = findFields(pkg, node, embeddedFields)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return parent, nil
}

func resolveTypeInfo(pkg *packages.Package, typeExpr any) (isPtr bool, typeStr, importPath string, typeObj types.Object, err error) {
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
	if o, ok := currentType.(objecter); ok {
		typeObj = o.Obj()
		if typeObj != nil && typeObj.Pkg() != nil {
			typePkg := typeObj.Pkg().Path()
			if typePkg != pkg.Types.Path() {
				importPath = typePkg
			}
		}
	}
	return
}

// findUnderlyingStruct recursively resolves types to find the base struct.
// It operates purely on go/types information.
func findUnderlyingStruct(t types.Type) *types.Struct {
	if t == nil {
		return nil
	}

	// If it's already a struct, we're done.
	if s, ok := t.Underlying().(*types.Struct); ok {
		return s
	}

	// Follow named types and aliases.
	for {
		switch next := t.(type) {
		case *types.Named:
			t = next.Underlying()
		case *types.Alias:
			t = next.Rhs()
		default:
			// Not a type we can resolve further.
			return nil
		}

		// Check if the new type is a struct.
		if s, ok := t.Underlying().(*types.Struct); ok {
			return s
		}

		// If the underlying type is the same as the current type, we've hit a loop or a base type.
		if t == t.Underlying() {
			return nil
		}
	}
}
