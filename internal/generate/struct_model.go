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
	// Inheritable by latter fields, including children and siblings after.
	// Or, global statistics about the struct
	type inheritable struct {
		// Inheritable
		table string

		// Global statistics
		uniqueTable    bool
		topHasModelTag bool
		columns        []ModelColumnInfo
	}
	type modelContext struct {
		*inheritable
		topLevel bool
	}
	ctx := modelContext{
		topLevel: true,
		inheritable: &inheritable{
			uniqueTable: true,
		},
	}
	WalkNodeContext(ctx, n, func(ctx modelContext, n *Node) (modelContext, bool) {
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
		if ctx.topLevel && n.Conf.Model {
			ctx.topHasModelTag = true
		}
		if n.IsAnonymous {
			// Anonymous fields are not treated as columns themselves, but their children might be.
			ctx.topLevel = false
			return ctx, true
		}
		if !n.IsExported {
			// Unexported fields cannot be accessed by the generated code, so we skip them.
			return ctx, false
		}
		if n.Conf.Table != "" && n.Conf.Column != "" {
			ctx.columns = append(ctx.columns, ModelColumnInfo{
				FieldName:  n.Name,
				TableName:  n.Conf.Table,
				ColumnName: n.Conf.Column,
			})
		}
		return ctx, false
	})

	if !ctx.topHasModelTag || !ctx.uniqueTable || len(ctx.columns) == 0 {
		return nil
	}

	return &StructModelInfo{
		Table:   ctx.table,
		Columns: ctx.columns,
	}
}
