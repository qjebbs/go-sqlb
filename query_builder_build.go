package sqlb

import (
	"fmt"
	"strings"

	"github.com/qjebbs/go-sqlf/v4"
	"github.com/qjebbs/go-sqlf/v4/util"
)

var _ Builder = (*QueryBuilder)(nil)
var _ sqlf.Builder = (*QueryBuilder)(nil)

// BuildQuery builds the query.
func (b *QueryBuilder) BuildQuery(style sqlf.BindStyle) (query string, args []any, err error) {
	ctx := sqlf.NewContext(style)
	query, err = b.buildInternal(ctx)
	if err != nil {
		return "", nil, err
	}
	args = ctx.Args()
	return query, args, nil
}

// Build implements sqlf.Builder
func (b *QueryBuilder) Build(ctx *sqlf.Context) (query string, err error) {
	return b.buildInternal(ctx)
}

// Debug enables debug mode which prints the interpolated query to stdout.
func (b *QueryBuilder) Debug(name ...string) *QueryBuilder {
	b.debug = true
	b.debugName = strings.Replace(strings.Join(name, "_"), " ", "_", -1)
	return b
}

// buildInternal builds the query with the selects.
func (b *QueryBuilder) buildInternal(ctx *sqlf.Context) (string, error) {
	if b == nil {
		return "", nil
	}
	err := b.anyError()
	if err != nil {
		return "", err
	}
	clauses := make([]string, 0)

	// SHOULD NOT assume b is self-contained,
	// b can depend on parent CTEs when it's a sub query.
	dep := b.depTablesCache
	if dep == nil {
		dep, err = b.collectDependencies()
		if err != nil {
			return "", err
		}
		b.depTablesCache = dep
	}
	if v := ctx.Value(depTablesKey{}); v != nil {
		if deps, ok := v.(map[string]bool); ok && deps != nil {
			// report dependencies to parent query builder
			for t := range dep {
				deps[t.AppliedName()] = true
			}
			// collecting dependencies only,
			// no need to build anything here
			return "", nil
		}
	}
	sq, err := b.buildCTEs(ctx, dep)
	if err != nil {
		return "", err
	}
	if sq != "" {
		clauses = append(clauses, sq)
	}

	// reserve a position for select
	selectAt := len(clauses)
	clauses = append(clauses, "")
	from, err := b.buildFrom(ctx, dep)
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
	// select must build order, because buildOrders may add columns to touches
	sel, err := b.buildSelects(ctx)
	if err != nil {
		return "", err
	}
	clauses[selectAt] = sel
	query := strings.TrimSpace(strings.Join(clauses, " "))
	if len(b.unions) > 0 {
		union, err := sqlf.Join(" ", b.unions...).Build(ctx)
		if err != nil {
			return "", err
		}
		query = strings.TrimSpace(query + " " + union)
	}
	if b.debug {
		interpolated, err := util.Interpolate(query, ctx.Args())
		if err != nil {
			if b.debugName == "" {
				fmt.Printf("debug: interpolated query: %s\n", err)
			} else {
				fmt.Printf("[%s] debug: interpolated query: %s\n", b.debugName, err)
			}
		}
		if b.debugName == "" {
			fmt.Println(interpolated)
		} else {
			fmt.Printf("[%s] %s\n", b.debugName, interpolated)
		}
	}
	return query, nil
}

func (b *QueryBuilder) buildCTEs(ctx *sqlf.Context, dep map[Table]bool) (string, error) {
	if len(b.ctes) == 0 {
		return "", nil
	}
	clauses := make([]string, 0, len(b.ctes))
	for _, cte := range b.ctes {
		if !dep[cte.table] {
			continue
		}
		query, err := cte.Build(ctx)
		if err != nil {
			return "", fmt.Errorf("build CTE '%s': %w", cte.table, err)
		}
		if query == "" {
			continue
		}
		clauses = append(clauses, fmt.Sprintf(
			"%s AS (%s)",
			cte.table.Name, query,
		))
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return "With " + strings.Join(clauses, ", "), nil
}

func (b *QueryBuilder) buildSelects(ctx *sqlf.Context) (string, error) {
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
	touches, err := sqlf.Join(", ", b.touches...).Build(ctx)
	if err != nil {
		return "", err
	}
	if sel == "" {
		return "", fmt.Errorf("no columns selected")
	}
	if touches == "" {
		return sel, nil
	}
	return sel + ", " + touches, nil
}

func (b *QueryBuilder) buildFrom(ctx *sqlf.Context, dep map[Table]bool) (string, error) {
	tables := make([]string, 0, len(b.tables))
	for _, t := range b.tables {
		if (b.distinct || len(b.groupbys) > 0) && t.optional && !dep[t.table] {
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
