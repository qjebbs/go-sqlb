package sqlb

import (
	"fmt"
	"strings"

	"github.com/qjebbs/go-sqlf/v4"
)

var _ sqlf.Builder = (*clauseWith)(nil)

// clauseWith represents a SQL WITH clause.
type clauseWith struct {
	builder sqlf.Builder

	debugName string

	ctes     []*cte
	ctesDict map[string]*cte // the actual cte names, not aliases

	deps           map[string]bool
	unresolvedDeps *dependencies
}

type cte struct {
	sqlf.Builder
	table Table
}

// newWith creates a new With instance.
func newWith() *clauseWith {
	return &clauseWith{
		ctesDict: make(map[string]*cte),
	}
}

// With adds a fragment as common table expression,
func (w *clauseWith) With(table Table, builder sqlf.Builder) *clauseWith {
	w.resetDepTablesCache()
	t := table.WithAlias("")
	cte := &cte{
		table:   t,
		Builder: builder,
	}
	w.ctes = append(w.ctes, cte)
	w.ctesDict[t.Name] = cte
	return w
}

// For sets the main query builder the CTEs is for.
func (w *clauseWith) For(builder sqlf.Builder) *clauseWith {
	w.resetDepTablesCache()
	w.builder = builder
	return w
}

// BuildQuery implements Builder
func (w *clauseWith) BuildQuery(style sqlf.BindStyle) (string, []any, error) {
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
func (w *clauseWith) Build(ctx *sqlf.Context) (string, error) {
	if w == nil {
		return "", nil
	}
	required, unresolved, err := w.collectDependencies(ctx)
	if err != nil {
		return "", err
	}

	if deps := dependenciesFromContext(ctx); deps != nil {
		for table := range unresolved.SourceNames {
			// report undefined dependencies to parent query builder
			// if b.debug && b.debugName != "" {
			// 	fmt.Printf("[%s] reporting dependency table: %s\n", b.debugName, name)
			// }
			deps.SourceNames[table] = true
		}
		for table := range unresolved.OuterTables {
			// report undefined dependencies to parent query builder
			// if b.debug && b.debugName != "" {
			// 	fmt.Printf("[%s] reporting dependency table: %s\n", b.debugName, name)
			// }
			deps.OuterTables[table] = true
		}
		// collecting dependencies only,
		// no need to build anything here
		return "", nil
	}
	return w.BuildRequired(ctx, required)
}

// BuildRequired builds the WITH clause including only the required CTEs.
func (w *clauseWith) BuildRequired(ctx *sqlf.Context, required map[string]bool) (query string, err error) {
	if w.builder != nil {
		query, err = w.builder.Build(ctx)
	}
	if len(w.ctes) == 0 {
		return query, nil
	}
	cteClauses := make([]string, 0, len(w.ctes))
	for _, cte := range w.ctes {
		if !required[cte.table.Name] {
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

func (w *clauseWith) collectDependencies(ctx *sqlf.Context) (required map[string]bool, unresolved *dependencies, err error) {
	if w.builder == nil {
		return nil, nil, nil
	}
	if w.deps != nil {
		return w.deps, w.unresolvedDeps, nil
	}
	deps := newDependencies(w.debugName)
	ctx = contextWithDependencies(ctx, deps)
	// collect dependencies from query builder
	_, err = w.builder.Build(ctx)
	if err != nil {
		return nil, nil, err
	}
	required, unresolved, err = w.CollectDependenciesForDeps(deps)
	if err != nil {
		return nil, nil, err
	}
	w.deps = required
	w.unresolvedDeps = unresolved
	return required, unresolved, nil
}

// CollectDependenciesForDeps collects the table dependencies for specific deps
func (w *clauseWith) CollectDependenciesForDeps(deps *dependencies) (required map[string]bool, unresolved *dependencies, err error) {
	required = make(map[string]bool)
	unresolved = newDependencies()
	for t := range deps.SourceNames {
		required[t] = true
	}
	// CTE subqueries can be sqlf.Builder that do not analyze tables,
	// so we need to collect dependencies from .Tables
	// BUT this can cause problems which reporting source names
	// that are not needed to.
	for t := range deps.Tables {
		required[t.Name] = true
	}
	for t := range deps.OuterTables {
		// w has no knowledge of outer tables, report as unresolved
		unresolved.OuterTables[t] = true
	}
	w.collectDepsBetweenCTEs(required)
	for t := range required {
		if _, ok := w.ctesDict[t]; !ok {
			unresolved.SourceNames[t] = true
		}
	}
	return required, unresolved, nil
}

func (w *clauseWith) collectDepsBetweenCTEs(required map[string]bool) error {
	cetDeps := make(map[string]bool)
	for _, cte := range w.ctes {
		if !required[cte.table.Name] {
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

func (w *clauseWith) collectDepsFromCTE(deps map[string]bool, cte *cte) error {
	key := cte.table.Name
	if deps[key] {
		return nil
	}
	deps[key] = true

	tables := newDependencies(w.debugName)
	ctx := contextWithDependencies(sqlf.NewContext(sqlf.BindStyleDollar), tables)
	_, err := cte.Builder.Build(ctx)
	if err != nil {
		return fmt.Errorf("collect dependencies of CTE %q: %w", cte.table, err)
	}
	// CTE can depend on other CTEs
	for t := range tables.Tables {
		if cte, ok := w.ctesDict[t.Name]; ok {
			err := w.collectDepsFromCTE(deps, cte)
			if err != nil {
				return err
			}
		}
	}
	// subquery of a CTE can depend on other CTEs
	for t := range tables.SourceNames {
		if cte, ok := w.ctesDict[t]; ok {
			err := w.collectDepsFromCTE(deps, cte)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *clauseWith) resetDepTablesCache() {
	w.deps = nil
	w.unresolvedDeps = nil
}

// Debug enables debug mode which prints the interpolated query to stdout.
func (w *clauseWith) Debug(name ...string) *clauseWith {
	w.debugName = strings.Replace(strings.Join(name, "_"), " ", "_", -1)
	return w
}

// HasCTE returns true if there is at least one CTE defined.
func (w *clauseWith) HasCTE() bool {
	return len(w.ctes) > 0
}
