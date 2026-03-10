package generate

import (
	"go/ast"

	"golang.org/x/tools/go/packages"
)

func (g *Generator) processFile(pkg *packages.Package, node *ast.File) ([]StructInfo, error) {
	var inspectError error
	var structs []StructInfo
	ast.Inspect(node, func(n ast.Node) bool {
		if inspectError != nil {
			return false
		}
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

		nd, err := parseNodes(pkg, s)
		if err != nil {
			inspectError = err
			return false
		}
		r := parseStruct(ts.Name.Name, nd, g.MapperMethods)
		if r != nil {
			structs = append(structs, *r)
		}
		return false
	})
	return structs, inspectError
}
