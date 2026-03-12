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
	var modelColumns []ModelColumnInfo
	type modelContext struct {
		// current table name, used for inheriting table names in embedded structs
		table       string
		uniqueTable bool
	}
	ctx := &modelContext{
		uniqueTable: true,
	}
	WalkNodeContext(ctx, n, func(ctx *modelContext, n *Node) (*modelContext, bool) {
		if n.Conf == nil {
			return ctx, true
		}
		if n.Conf.Table != "" {
			if ctx.uniqueTable && ctx.table != "" && ctx.table != n.Conf.Table {
				ctx.uniqueTable = false
			}
			ctx.table = n.Conf.Table
		} else {
			// inherit table name
			n.Conf.Table = ctx.table
		}
		if n.IsAnonymous {
			// Anonymous fields are not treated as columns themselves, but their children might be.
			return ctx, true
		}
		if !n.IsExported {
			// Unexported fields cannot be accessed by the generated code, so we skip them.
			return ctx, false
		}
		if n.Conf.Table != "" && n.Conf.Column != "" {
			modelColumns = append(modelColumns, ModelColumnInfo{
				FieldName:  n.Name,
				TableName:  n.Conf.Table,
				ColumnName: n.Conf.Column,
			})
		}
		return ctx, false
	})

	if !ctx.uniqueTable || len(modelColumns) == 0 {
		return nil
	}

	return &StructModelInfo{
		Table:   ctx.table,
		Columns: modelColumns,
	}
}
