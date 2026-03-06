package generate

// StructModelInfo holds information about a struct that maps to a database table,
// including the unique table name and the columns that map to struct fields.
type StructModelInfo struct {
	Table   string
	Columns []ModelColumnInfo
}

// ModelColumnInfo holds information about a struct field that maps to a database column.
type ModelColumnInfo struct {
	FieldName  string
	TableName  string
	ColumnName string
}

func parseStructModel(n *Node) *StructModelInfo {
	var curTable string
	var modelColumns []ModelColumnInfo
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
			modelColumns = append(modelColumns, ModelColumnInfo{
				FieldName:  n.Name,
				TableName:  n.Conf.Table,
				ColumnName: n.Conf.Column,
			})
		}
		return false
	})

	if !uniqueTable || len(modelColumns) == 0 {
		return nil
	}

	return &StructModelInfo{
		Table:   curTable,
		Columns: modelColumns,
	}
}
