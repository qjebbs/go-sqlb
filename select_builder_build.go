package sqlb

import (
	"fmt"
	"strings"

	"github.com/qjebbs/go-sqlf/v4"
	"github.com/qjebbs/go-sqlf/v4/util"
)

// BuildQuery builds the query.
func (b *SelectBuilder) BuildQuery(style sqlf.BindStyle) (query string, args []any, err error) {
	ctx := sqlf.NewContext(style)
	query, err = b.buildInternal(ctx)
	if err != nil {
		return "", nil, err
	}
	args = ctx.Args()
	return query, args, nil
}

// Build implements sqlf.Builder
func (b *SelectBuilder) Build(ctx *sqlf.Context) (query string, err error) {
	return b.buildInternal(ctx)
}

// Debug enables debug mode which prints the interpolated query to stdout.
func (b *SelectBuilder) Debug(name ...string) *SelectBuilder {
	b.debug = true
	b.debugName = strings.Replace(strings.Join(name, "_"), " ", "_", -1)
	return b
}

// buildInternal builds the query with the selects.
func (b *SelectBuilder) buildInternal(ctx *sqlf.Context) (string, error) {
	if b == nil {
		return "", nil
	}
	err := b.anyError()
	if err != nil {
		return "", err
	}
	clauses := make([]string, 0)

	myDeps, err := b.collectDependencies()
	if err != nil {
		return "", err
	}
	if deps := depTablesFromContext(ctx); deps != nil {
		for t := range myDeps.unresolved.OuterTables {
			deps.OuterTables[t] = true
		}
		for t := range myDeps.unresolved.SourceNames {
			deps.SourceNames[t] = true
		}
		// collecting dependencies only,
		// no need to build anything here
		return "", nil
	}
	with, err := b.ctes.buildRequired(ctx, myDeps.cteDeps)
	if err != nil {
		return "", err
	}
	if with != "" {
		clauses = append(clauses, with)
	}
	// reserve a position for select
	sel, err := b.buildSelects(ctx)
	if err != nil {
		return "", err
	}
	clauses = append(clauses, sel)
	from, err := b.buildFrom(ctx, myDeps.queryDeps)
	if err != nil {
		return "", err
	}
	if from != "" {
		clauses = append(clauses, from)
	}
	where, err := sqlf.Prefix(
		"WHERE",
		sqlf.Join(" AND ", b.conditions...),
	).Build(ctx)
	if err != nil {
		return "", err
	}
	if where != "" {
		clauses = append(clauses, where)
	}
	groupby, err := sqlf.Prefix(
		"GROUP BY",
		sqlf.Join(", ", b.groupbys...),
	).Build(ctx)
	if err != nil {
		return "", err
	}
	if groupby != "" {
		clauses = append(clauses, groupby)
		having, err := sqlf.Prefix(
			"HAVING",
			sqlf.Join(" AND ", b.havings...),
		).Build(ctx)
		if err != nil {
			return "", err
		}
		if having != "" {
			clauses = append(clauses, having)
		}
	}
	order, err := b.buildOrders(ctx)
	if err != nil {
		return "", err
	}
	if order != "" {
		clauses = append(clauses, order)
	}
	if b.limit > 0 {
		clauses = append(clauses, fmt.Sprintf(`LIMIT %d`, b.limit))
	}
	if b.offset > 0 {
		clauses = append(clauses, fmt.Sprintf(`OFFSET %d`, b.offset))
	}
	query := strings.TrimSpace(strings.Join(clauses, " "))
	if len(b.unions) > 0 {
		union, err := sqlf.Join(" ", b.unions...).Build(ctx)
		if err != nil {
			return "", err
		}
		query = strings.TrimSpace(query + " " + union)
	}
	if b.debug {
		prefix := b.debugName
		if prefix == "" {
			prefix = "sqlb"
		}
		interpolated, err := util.Interpolate(query, ctx.Args())
		if err != nil {
			fmt.Printf("[%s] interpolating: %s\n", prefix, err)
		}
		fmt.Printf("[%s] %s\n", prefix, interpolated)
	}
	return query, nil
}

func (b *SelectBuilder) buildSelects(ctx *sqlf.Context) (string, error) {
	prefix := "SELECT"
	if b.distinct {
		prefix = "SELECT DISTINCT"
	}
	sel, err := sqlf.Prefix(
		prefix,
		sqlf.Join(", ", b.selects...),
	).Build(ctx)
	if err != nil {
		return "", err
	}
	if sel == "" {
		return "", fmt.Errorf("no columns selected")
	}
	return sel, nil
}

func (b *SelectBuilder) buildFrom(ctx *sqlf.Context, dep *depTables) (string, error) {
	tables := make([]string, 0, len(b.tables))
	for _, t := range b.tables {
		if b.shouldEliminateTable(t, dep) {
			continue
		}
		c, err := t.Builder.Build(ctx)
		if err != nil {
			return "", fmt.Errorf("build FROM '%s': %w", t.table, err)
		}
		tables = append(tables, c)
	}
	return "FROM " + strings.Join(tables, " "), nil
}

func (b *SelectBuilder) shouldEliminateTable(t *fromTable, dep *depTables) bool {
	if !t.optional || dep.Tables[t.table] {
		return false
	}
	// automatic elimination for LEFT JOIN tables
	if b.distinct || len(b.groupbys) > 0 {
		return true
	}
	return t.forceEliminate
}
