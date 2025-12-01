package sqlb

import "github.com/qjebbs/go-sqlf/v4"

type depTablesKey struct{}

func contextWithDepTables(ctx *sqlf.Context, deps *depTables) *sqlf.Context {
	return sqlf.ContextWith(ctx, depTablesKey{}, deps)
}

func depTablesFromContext(ctx *sqlf.Context) *depTables {
	if v := ctx.Value(depTablesKey{}); v != nil {
		if deps, ok := v.(*depTables); ok && deps != nil {
			return deps
		}
	}
	return nil
}

type depTables struct {
	debugName string
	// tables are the unresolved tables reported by subqueries
	tables map[Table]bool
}

func (d *depTables) Merge(from *depTables) {
	for t := range from.tables {
		d.tables[t] = true
	}
}

func newDepTables(debugName ...string) *depTables {
	var name string
	if len(debugName) > 0 {
		name = debugName[0]
	}
	return &depTables{
		debugName: name,
		tables:    make(map[Table]bool),
	}
}
