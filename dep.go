package sqlb

import "github.com/qjebbs/go-sqlf/v4"

type depAnalysisKey struct{}
type dependenciesKey struct{}

// contextWithPruning returns a new context with JOIN / CTE pruning enabled.
func contextWithPruning(ctx *sqlf.Context) *sqlf.Context {
	return sqlf.ContextWithValue(ctx, depAnalysisKey{}, struct{}{})
}

// pruningFromContext extracts JOIN / CTE pruning flag from context.
func pruningFromContext(ctx *sqlf.Context) bool {
	v := ctx.Value(depAnalysisKey{})
	if v != nil {
		return true
	}
	return false
}

// decideContextPruning decides whether to enable pruning based on the builder setting and context.
func decideContextPruning(ctx *sqlf.Context, value bool) (*sqlf.Context, bool) {
	if !value {
		return ctx, pruningFromContext(ctx)
	}
	if !pruningFromContext(ctx) {
		ctx = contextWithPruning(ctx)
	}
	return ctx, true
}

// contextWithDependencies returns a new context with *Dependencies attached.
func contextWithDependencies(ctx *sqlf.Context, deps *dependencies) *sqlf.Context {
	return sqlf.ContextWithValue(ctx, dependenciesKey{}, deps)
}

// dependenciesFromContext extracts *Dependencies from context.
func dependenciesFromContext(ctx *sqlf.Context) *dependencies {
	if v := ctx.Value(dependenciesKey{}); v != nil {
		if deps, ok := v.(*dependencies); ok && deps != nil {
			return deps
		}
	}
	return nil
}

// dependencies tracks the table dependencies during building SQL queries.
type dependencies struct {
	DebugName string
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

// Merge merges another DepTables into this one.
func (d *dependencies) Merge(from *dependencies) {
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

// newDependencies creates a new Dependencies instance.
func newDependencies(debugName ...string) *dependencies {
	var name string
	if len(debugName) > 0 {
		name = debugName[0]
	}
	return &dependencies{
		DebugName:   name,
		Tables:      make(map[Table]bool),
		OuterTables: make(map[Table]bool),
		SourceNames: make(map[string]bool),
	}
}
