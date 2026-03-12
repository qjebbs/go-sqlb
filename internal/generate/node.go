package generate

import (
	"fmt"
	"go/ast"
	"go/types"

	"github.com/qjebbs/go-sqlb/tag"
	"golang.org/x/tools/go/packages"
)

// Node represents a node in the struct field hierarchy, which can be used to generate SQL builder code.
type Node struct {
	Name        string
	FieldType   string // The string representation of the field's type, qualified with package name if necessary
	IsAnonymous bool   // Indicates if the field is an anonymous (embedded) field
	IsExported  bool   // Indicates if the field is exported (public)
	IsPtr       bool   `json:",omitempty"` // Indicates if the field is a pointer type
	ImportPath  string `json:",omitempty"` // The package path of the field's type, if it's from an imported package

	Conf *tag.Info `json:",omitempty"` // The parsed tag information, if available

	Parent   *Node   `json:"-"`
	Children []*Node `json:",omitempty"`
}

// AddChild adds a child node to the current node and sets the parent reference of the child.
func (n *Node) AddChild(child *Node) {
	child.Parent = n
	n.Children = append(n.Children, child)
}

// WalkNodeContext is a variant of Walk that allows passing a context value through the traversal.
func WalkNodeContext[T any](ctx T, n *Node, fn func(T, *Node) (T, bool)) {
	newCtx, ok := fn(ctx, n)
	if !ok {
		return
	}
	for _, child := range n.Children {
		WalkNodeContext(newCtx, child, fn)
	}
}

func parseNodes(pkg *packages.Package, typ *ast.StructType) (*Node, error) {
	var initialFields []FieldInfo
	for _, f := range typ.Fields.List {
		var name string
		if len(f.Names) > 0 {
			name = f.Names[0].Name
		} else {
			// Handle anonymous fields using type information for robustness
			if typ := pkg.TypesInfo.TypeOf(f.Type); typ != nil {
				if ptr, ok := typ.(*types.Pointer); ok {
					typ = ptr.Elem()
				}
				if o, ok := typ.(objecter); ok {
					name = o.Obj().Name()
				} else {
					return nil, fmt.Errorf("not able to determine name for anonymous field with %T of %v", typ, typ)
				}
			}
		}
		var tag string
		if f.Tag != nil {
			tag = f.Tag.Value
		}
		initialFields = append(initialFields, FieldInfo{
			Name:        name,
			Tag:         tag,
			IsAnonymous: f.Names == nil,
			IsExported:  f.Names == nil || f.Names[0].IsExported(),
			Type:        f.Type,
		})
	}
	return findFields(pkg, &Node{}, initialFields)
}
