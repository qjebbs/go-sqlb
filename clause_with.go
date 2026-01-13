package sqlb

import (
	"fmt"
	"strings"

	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

// clauseWith represents a SQL WITH clause.
type clauseWith struct {
	debugger
	ctes     []*cte
	ctesDict map[string]*cte // the actual cte names, not aliases

	pruning        bool
	deps           map[string]bool
	unresolvedDeps *dependencies
}

type cte struct {
	sqlf.Builder
	table   Table
	columns []sqlf.Builder
	types   []string
	values  [][]any
}

// newWith creates a new With instance.
func newWith() *clauseWith {
	return &clauseWith{
		ctesDict: make(map[string]*cte),
	}
}

// With adds a builder as common table expression,
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

// WithValues adds a VALUES common table expression.
func (w *clauseWith) WithValues(table Table, columns []string, types []string, values [][]any) *clauseWith {
	w.resetDepTablesCache()
	t := table.WithAlias("")
	cte := &cte{
		table: t,
		columns: util.Map(columns, func(c string) sqlf.Builder {
			return sqlf.Identifier(c)
		}),
		types:  types,
		values: values,
	}
	w.ctes = append(w.ctes, cte)
	w.ctesDict[t.Name] = cte
	return w
}

// BuildRequired builds the WITH clause including only the required CTEs.
func (w *clauseWith) BuildRequired(ctx *sqlf.Context, required map[string]bool) (query string, err error) {
	pruning := pruningFromContext(ctx)
	cteClauses := make([]sqlf.Builder, 0, len(w.ctes))
	dialect, err := DialectFromContext(ctx)
	if err != nil {
		return "", err
	}
	for _, cte := range w.ctes {
		if pruning && (required == nil || !required[cte.table.Name]) {
			continue
		}
		builder := cte.Builder
		if builder == nil && len(cte.columns) > 0 {
			if len(cte.values) == 0 {
				return "", fmt.Errorf("WithValues(%s): values cannot be empty", cte.table.Name)
			}
			if len(cte.columns) != len(cte.values[0]) {
				return "", fmt.Errorf("WithValues(%s): number of columns and values do not match", cte.table.Name)
			}
			sb := new(strings.Builder)
			sb.WriteRune('(')
			for i := range cte.columns {
				if i > 0 {
					sb.WriteString(", ")
				}
				colType := ""
				if i < len(cte.types) {
					colType = cte.types[i]
					expr := dialect.CastType(colType)
					if expr == "" {
						return "", fmt.Errorf("WithValues: unsupported dialect %T for type casting", dialect)
					}
					sb.WriteString(expr)
				} else {
					sb.WriteString("?")
				}
			}
			sb.WriteRune(')')
			rowTmpl := sb.String()
			builder = sqlf.Join(", ", util.Map(cte.values, func(i []any) sqlf.Builder {
				return sqlf.F(rowTmpl, i...)
			})...)
			builder = sqlf.Prefix("VALUES", builder)
		}
		if len(cte.columns) == 0 {
			cteClauses = append(cteClauses, sqlf.F(
				"? AS (?)",
				sqlf.Identifier(cte.table.Name), builder,
			))
		} else {
			cteClauses = append(cteClauses, sqlf.F(
				"? (?) AS (?)",
				sqlf.Identifier(cte.table.Name),
				sqlf.Join(", ", cte.columns...),
				builder,
			))
		}
	}
	if len(cteClauses) == 0 {
		return "", nil
	}
	return sqlf.Prefix("WITH", sqlf.Join(", ", cteClauses...)).BuildTo(ctx)
}

// CollectDependenciesForDeps collects the table dependencies for specific deps
func (w *clauseWith) CollectDependenciesForDeps(ctx *sqlf.Context, deps *dependencies) (required map[string]bool, unresolved *dependencies, err error) {
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
	w.collectDepsBetweenCTEs(ctx, required)
	for t := range required {
		if _, ok := w.ctesDict[t]; !ok {
			unresolved.SourceNames[t] = true
		}
	}
	return required, unresolved, nil
}

func (w *clauseWith) collectDepsBetweenCTEs(ctx *sqlf.Context, required map[string]bool) error {
	cetDeps := make(map[string]bool)
	for _, cte := range w.ctes {
		if !required[cte.table.Name] {
			continue
		}
		err := w.collectDepsFromCTE(ctx, cetDeps, cte)
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

func (w *clauseWith) collectDepsFromCTE(ctx *sqlf.Context, deps map[string]bool, cte *cte) error {
	key := cte.table.Name
	if deps[key] {
		return nil
	}
	deps[key] = true

	if cte.Builder == nil {
		// WithValues has no dependencies
		return nil
	}
	tables := newDependencies(w.name)
	depCtx := contextWithDependencies(ctx, tables)
	_, err := cte.Builder.BuildTo(depCtx)
	if err != nil {
		return fmt.Errorf("collect dependencies of CTE %q: %w", cte.table, err)
	}
	// CTE can depend on other CTEs
	for t := range tables.Tables {
		if cte, ok := w.ctesDict[t.Name]; ok {
			err := w.collectDepsFromCTE(ctx, deps, cte)
			if err != nil {
				return err
			}
		}
	}
	// subquery of a CTE can depend on other CTEs
	for t := range tables.SourceNames {
		if cte, ok := w.ctesDict[t]; ok {
			err := w.collectDepsFromCTE(ctx, deps, cte)
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
	w.debugger.Debug(name...)
	return w
}

// EnableElimination enables JOIN / CTE pruning based on dependency analysis.
// To use pruning, make sure all table references are done via Table objects.
//
// For example,
//
//	t := sqlb.NewTable("foo", "f")
//	b.Where(sqlf.F("? = ?", t.Column("id"), 1))
//	// instead of
//	b.Where(sqlf.F("f.id = ?", 1))
func (w *clauseWith) EnableElimination() *clauseWith {
	w.pruning = true
	return w
}

// HasCTE returns true if there is at least one CTE defined.
func (w *clauseWith) HasCTE() bool {
	return len(w.ctes) > 0
}
