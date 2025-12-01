package sqlb

import (
	"fmt"
	"strings"

	"github.com/qjebbs/go-sqlf/v4"
)

var _ Builder = (*_CTEs)(nil)
var _ sqlf.Builder = (*_CTEs)(nil)

// _With creates a new CTEs instance with a builder as common table expression.
//
// !!! *CTEs tracks dependencies the help of sqlb.Table,
// make sure all the table references are built from sqlb.Table,
//
// For example:
//
//	foo := sqlb.NewTable("foo")
//	sqlb._With(foo, sqlf.F("SELECT 1")).
//		For(sqlf.F(
//			"SELECT * FROM ?", foo,
//		))
func _With(table Table, builder sqlf.Builder) *_CTEs {
	return newCTEs().With(table, builder)
}

// _CTEs represents a SQL WITH clause.
type _CTEs struct {
	builder sqlf.Builder

	debugName string

	// All Table values HAVE NO ALIAS inside CTEs.

	ctes     []*cte
	ctesDict map[Table]*cte // the actual cte names, not aliases

	deps           map[Table]bool
	unresolvedDeps map[Table]bool
}

type cte struct {
	sqlf.Builder
	table Table
}

// newCTEs creates a new CTEs instance.
func newCTEs() *_CTEs {
	return &_CTEs{
		ctesDict: make(map[Table]*cte),
	}
}

// With adds a fragment as common table expression,
func (w *_CTEs) With(table Table, builder sqlf.Builder) *_CTEs {
	w.resetDepTablesCache()
	t := table.WithAlias("")
	cte := &cte{
		table:   t,
		Builder: builder,
	}
	w.ctes = append(w.ctes, cte)
	w.ctesDict[t] = cte
	return w
}

// For sets the main query builder the CTEs is for.
func (w *_CTEs) For(builder sqlf.Builder) *_CTEs {
	w.resetDepTablesCache()
	w.builder = builder
	return w
}

// BuildQuery implements Builder
func (w *_CTEs) BuildQuery(style sqlf.BindStyle) (string, []any, error) {
	if w == nil {
		return "", nil, nil
	}
	ctx := sqlf.NewContext(style)
	query, err := w.Build(ctx)
	if err != nil {
		return "", nil, err
	}
	args := ctx.Args()
	return query, args, nil
}

// Build implements sqlf.Builder
func (w *_CTEs) Build(ctx *sqlf.Context) (string, error) {
	if w == nil {
		return "", nil
	}
	required, unresolved, err := w.collectDependencies(ctx)
	if err != nil {
		return "", err
	}

	if parentDeps := depTablesFromContext(ctx); parentDeps != nil {
		for table := range unresolved {
			// report undefined dependencies to parent query builder
			// if b.debug && b.debugName != "" {
			// 	fmt.Printf("[%s] reporting dependency table: %s\n", b.debugName, name)
			// }
			parentDeps.tables[table] = true
		}
		// collecting dependencies only,
		// no need to build anything here
		return "", nil
	}
	return w.buildRequired(ctx, required)
}

func (w *_CTEs) buildRequired(ctx *sqlf.Context, required map[Table]bool) (query string, err error) {
	if w.builder != nil {
		query, err = w.builder.Build(ctx)
	}
	if len(w.ctes) == 0 {
		return query, nil
	}
	cteClauses := make([]string, 0, len(w.ctes))
	for _, cte := range w.ctes {
		if !required[cte.table] {
			continue
		}
		sq, err := cte.Build(ctx)
		if err != nil {
			return "", err
		}
		if sq != "" {
			cteClauses = append(cteClauses, fmt.Sprintf(
				"%s AS (%s)",
				cte.table.Name, sq,
			))
		}
	}
	if len(cteClauses) == 0 {
		return query, nil
	}
	withClauses := "WITH " + strings.Join(cteClauses, ", ")
	if query == "" {
		return withClauses, nil
	}
	return withClauses + " " + query, nil
}

func (w *_CTEs) collectDependencies(ctx *sqlf.Context) (required, unresolved map[Table]bool, err error) {
	if w.builder == nil {
		return nil, nil, nil
	}
	if w.deps != nil {
		return w.deps, w.unresolvedDeps, nil
	}
	deps := newDepTables(w.debugName)
	ctx = contextWithDepTables(ctx, deps)
	// collect dependencies from query builder
	_, err = w.builder.Build(ctx)
	if err != nil {
		return nil, nil, err
	}
	required, unresolved, err = w.collectDependenciesForTables(deps)
	if err != nil {
		return nil, nil, err
	}
	w.deps = required
	w.unresolvedDeps = unresolved
	return required, unresolved, nil
}

func (w *_CTEs) collectDependenciesForTables(deps *depTables) (required, unresolved map[Table]bool, err error) {
	required = make(map[Table]bool)
	unresolved = make(map[Table]bool)
	for t := range deps.tables {
		table := t.WithAlias("")
		required[table] = true
	}
	w.collectDepsBetweenCTEs(required)
	for t := range required {
		if _, ok := w.ctesDict[t]; !ok {
			unresolved[t] = true
		}
	}
	return required, unresolved, nil
}

func (w *_CTEs) collectDepsBetweenCTEs(required map[Table]bool) error {
	cetDeps := make(map[Table]bool)
	for _, cte := range w.ctes {
		if !required[cte.table] {
			continue
		}
		err := w.collectDepsFromCTE(cetDeps, cte)
		if err != nil {
			return err
		}
	}
	// merge collected deps
	for t := range cetDeps {
		required[t] = true
	}
	return nil
}

func (w *_CTEs) collectDepsFromCTE(deps map[Table]bool, cte *cte) error {
	key := cte.table
	if deps[key] {
		return nil
	}
	deps[key] = true

	tables := newDepTables(w.debugName)
	ctx := contextWithDepTables(sqlf.NewContext(sqlf.BindStyleDollar), tables)
	_, err := cte.Builder.Build(ctx)
	if err != nil {
		return fmt.Errorf("collect dependencies of CTE %q: %w", cte.table, err)
	}
	for t := range tables.tables {
		table := t.WithAlias("")
		if cte, ok := w.ctesDict[table]; ok {
			err := w.collectDepsFromCTE(deps, cte)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *_CTEs) resetDepTablesCache() {
	w.deps = nil
	w.unresolvedDeps = nil
}

// Debug enables debug mode which prints the interpolated query to stdout.
func (w *_CTEs) Debug(name ...string) *_CTEs {
	w.debugName = strings.Replace(strings.Join(name, "_"), " ", "_", -1)
	return w
}
