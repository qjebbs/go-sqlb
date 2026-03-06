package generate

import (
	"go/ast"
	"go/types"

	"github.com/qjebbs/go-sqlb/internal/tag/syntax"
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

	Conf *syntax.Info `json:",omitempty"` // The parsed tag information, if available

	Parent   *Node   `json:"-"`
	Children []*Node `json:",omitempty"`
}

// AddChild adds a child node to the current node and sets the parent reference of the child.
func (n *Node) AddChild(child *Node) {
	child.Parent = n
	n.Children = append(n.Children, child)
}

// Walk traverses the node hierarchy and applies the given function to each node.
func (n *Node) Walk(fn func(*Node) bool) {
	if !fn(n) {
		return
	}
	for _, child := range n.Children {
		child.Walk(fn)
	}
}

func parseNodes(pkg *packages.Package, typ *ast.StructType) *Node {
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
