package generate

// StructInfo holds information about a struct to be processed by the template.
type StructInfo struct {
	Name   string
	Model  *StructModelInfo
	Select *StructSelectInfo
}

func parseStruct(name string, n *Node) *StructInfo {
	model := parseStructModel(n)
	selects := parseStructSelects(n)
	if model == nil && selects == nil {
		return nil
	}
	return &StructInfo{
		Name:   name,
		Model:  model,
		Select: selects,
	}
}
