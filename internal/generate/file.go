package generate

import (
	"go/ast"

	"golang.org/x/tools/go/packages"
)

func (g *Generator) processFile(pkg *packages.Package, node *ast.File) []StructInfo {
	var structs []StructInfo
	ast.Inspect(node, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			// ignore function literals to avoid processing struct types defined inside them,
			// which are not relevant for SQL builder generation.
			return false
		}
		// fmt.Printf("node %T\n", n)

		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		// fmt.Printf("type %s\n", ts.Name)

		s, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}

		nd := parseNodes(pkg, s)
		r := parseStruct(ts.Name.Name, nd)
		if r != nil {
			structs = append(structs, *r)
		}
		return false
	})
	return structs
}
