package generate

import "github.com/qjebbs/go-sqlb/tag"

// StructSelectInfo holds information about the fields of a struct that are used in select queries,
// including their select tags and the necessary imports for generating initialization code in the Values() method.
type StructSelectInfo struct {
	Columns    []SelectColumnInfo
	Imports    []string
	InitFields []InitFieldInfo
}

// SelectColumnInfo holds information about a struct field that is used in select queries,
// including the path to access it through embedded structs and whether any part of that path is a pointer.
type SelectColumnInfo struct {
	Selector string
	Diving   bool
	Tags     *tag.Info
}

// InitFieldInfo holds information about a struct field that needs to be initialized in the Values() method.
type InitFieldInfo struct {
	Type     string
	Selector string
}

// SelectorInfo holds information about the path to access a struct field through structs,
// including which parts of the path are pointers and their types.
// This is used for generating initialization code in the Values() method.
type SelectorInfo struct {
	Selectors    []string // The list of selectors to access this field
	PointerTypes []string // The list of types for each selector in the path, used for generating initialization code
	Pointers     []bool   // The list of indices in the selector path that are pointers
}

func parseStructSelects(n *Node) *StructSelectInfo {
	var (
		columns   []SelectColumnInfo
		imports   []string
		selectors []SelectorInfo
	)
	var curTable string
	n.Walk(func(n *Node) bool {
		if n.Conf == nil {
			return true
		}
		if n.Conf.Table != "" {
			curTable = n.Conf.Table
		} else {
			// inherit table name
			n.Conf.Table = curTable
		}
		if !n.IsExported {
			// Unexported fields cannot be accessed by the generated code, so we skip them.
			return false
		}
		if n.IsAnonymous {
			// Anonymous fields are not treated as columns themselves, but their children might be.
			return true
		}
		if n.Conf.Dive {
			// If Dive is true, we want to continue walking into this field's children, but we don't want to treat this field as a column itself.
			return true
		}
		if n.Conf.Table != "" && n.Conf.Column != "" {
			var selectorPath []string
			var pointerIndices []bool
			var types []string
			parent := n.Parent
			for parent != nil && parent.Parent != nil {
				selectorPath = append(selectorPath, parent.Name)
				pointerIndices = append(pointerIndices, parent.IsPtr)
				types = append(types, parent.FieldType)
				if parent.IsPtr && parent.ImportPath != "" {
					// import for generating s.Selector1.Selector2 = new(somepkg.SomeType)
					imports = append(imports, parent.ImportPath)
				}
				parent = parent.Parent
			}
			selectors = append(selectors, SelectorInfo{
				Selectors:    selectorPath,
				Pointers:     pointerIndices,
				PointerTypes: types,
			})
			fieldSelector := ""
			for i := len(selectorPath) - 1; i >= 0; i-- {
				fieldSelector += "." + selectorPath[i]
			}
			fieldSelector += "." + n.Name
			info := SelectColumnInfo{
				Selector: fieldSelector,
				Diving:   isDiving(n),
				Tags:     n.Conf,
			}
			columns = append(columns, info)
		}
		return false
	})
	if len(columns) == 0 {
		return nil
	}
	var seenInitFields = make(map[string]struct{})
	var initFields []InitFieldInfo
	for _, col := range selectors {
		// pointers and selectors are in reverse order,
		// we need to generate initialization code from the
		// outermost pointer to the innermost one.
		for i := len(col.Pointers) - 1; i >= 0; i-- {
			if !col.Pointers[i] {
				continue
			}
			selector := ""
			for j := len(col.Pointers) - 1; j >= i; j-- {
				selector += "." + col.Selectors[j]
			}
			if _, seen := seenInitFields[selector]; seen {
				continue
			}
			initFields = append(initFields, InitFieldInfo{
				Selector: selector,
				Type:     col.PointerTypes[i],
			})
			seenInitFields[selector] = struct{}{}
		}
	}
	return &StructSelectInfo{
		Columns:    columns,
		Imports:    imports,
		InitFields: initFields,
	}
}

func isDiving(n *Node) bool {
	parent := n.Parent
	for parent != nil {
		if parent.Conf != nil && parent.Conf.Dive {
			return true
		}
		parent = parent.Parent
	}
	return false
}
