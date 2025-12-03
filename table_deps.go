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
	// Tables are the resolved tables referenced by columns.
	// e.g. The table 'f' of 'f.id' in the query below is reported here:
	//   SELECT f.* FROM foo f
	Tables map[Table]bool
	// OuterTables are the unresolved tables referenced by columns from outer scope.
	// e.g. The table 'f' of 'f.id' in the subquery below is reported here:
	//   SELECT f.* FROM foo f
	//   WHERE NOT EXISTS (
	//     SELECT 1 FROM bar b
	//     WHERE b.foo_id = f.id AND b.id = 1
	//   );
	OuterTables map[Table]bool
	// SourceNames are the unresolved table/CTE names from FROM/JOIN clauses.
	// It could be a base table or a CTE from outer scope.
	// e.g. The table 'foo' in the query below is reported here,
	//   SELECT * FROM foo;
	SourceNames map[string]bool
}

func (d *depTables) Merge(from *depTables) {
	for t := range from.Tables {
		d.Tables[t] = true
	}
	for t := range from.OuterTables {
		d.OuterTables[t] = true
	}
	for t := range from.SourceNames {
		d.SourceNames[t] = true
	}
}

func newDepTables(debugName ...string) *depTables {
	var name string
	if len(debugName) > 0 {
		name = debugName[0]
	}
	return &depTables{
		debugName:   name,
		Tables:      make(map[Table]bool),
		OuterTables: make(map[Table]bool),
		SourceNames: make(map[string]bool),
	}
}
