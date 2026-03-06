package generate

// StructInfo holds information about a struct to be processed by the template.
type StructInfo struct {
	Name        string
	Columns     []ColumnInfo
	UniqueTable string
}

// ColumnInfo holds information about a struct field that maps to a database column.
type ColumnInfo struct {
	FieldName  string
	TableName  string
	ColumnName string
}

func parseStruct(name string, n *Node) *StructInfo {
	var curTable string
	var columns []ColumnInfo

	uniqueTable := true
	n.Walk(func(n *Node) bool {
		if n.Conf == nil {
			return true
		}
		if n.Conf.Table != "" {
			if uniqueTable && curTable != "" && curTable != n.Conf.Table {
				uniqueTable = false
			}
			curTable = n.Conf.Table
		} else {
			// inherit table name
			n.Conf.Table = curTable
		}
		if n.IsAnonymous {
			// Anonymous fields are not treated as columns themselves, but their children might be.
			return true
		}
		if !n.IsExported {
			// Unexported fields cannot be accessed by the generated code, so we skip them.
			return false
		}
		if n.Conf.Table != "" && n.Conf.Column != "" {
			columns = append(columns, ColumnInfo{
				FieldName:  n.Name,
				TableName:  n.Conf.Table,
				ColumnName: n.Conf.Column,
			})
		}
		return false
	})
	if len(columns) == 0 {
		return nil
	}
	if !uniqueTable {
		curTable = ""
	}
	return &StructInfo{
		Name:        name,
		Columns:     columns,
		UniqueTable: curTable,
	}
}
