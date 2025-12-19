package clauses

import "github.com/qjebbs/go-sqlf/v4"

type dependenciesKey struct{}

// ContextWithDependencies returns a new context with *Dependencies attached.
func ContextWithDependencies(ctx *sqlf.Context, deps *Dependencies) *sqlf.Context {
	return sqlf.ContextWith(ctx, dependenciesKey{}, deps)
}

// DependenciesFromContext extracts *Dependencies from context.
func DependenciesFromContext(ctx *sqlf.Context) *Dependencies {
	if v := ctx.Value(dependenciesKey{}); v != nil {
		if deps, ok := v.(*Dependencies); ok && deps != nil {
			return deps
		}
	}
	return nil
}

// Dependencies tracks the table dependencies during building SQL queries.
type Dependencies struct {
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
func (d *Dependencies) Merge(from *Dependencies) {
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

// NewDependencies creates a new Dependencies instance.
func NewDependencies(debugName ...string) *Dependencies {
	var name string
	if len(debugName) > 0 {
		name = debugName[0]
	}
	return &Dependencies{
		DebugName:   name,
		Tables:      make(map[Table]bool),
		OuterTables: make(map[Table]bool),
		SourceNames: make(map[string]bool),
	}
}
