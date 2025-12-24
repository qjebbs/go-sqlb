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
	built := make([]string, 0)

	myDeps, err := b.collectDependencies()
	if err != nil {
		return "", err
	}
	if deps := dependenciesFromContext(ctx); deps != nil {
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
	with, err := b.ctes.BuildRequired(ctx, myDeps.cteDeps)
	if err != nil {
		return "", err
	}
	if with != "" {
		built = append(built, with)
	}
	sel, err := b.buildSelects(ctx)
	if err != nil {
		return "", err
	}
	built = append(built, sel)
	from, err := b.from.BuildRequired(ctx, &fromBuilderMeta{
		DebugName:  b.debugName,
		Distinct:   b.distinct,
		HasGroupBy: !b.groupbys.Empty(),
	}, myDeps.queryDeps)
	if err != nil {
		return "", err
	}
	if from != "" {
		built = append(built, from)
	}
	where, err := b.where.Build(ctx)
	if err != nil {
		return "", err
	}
	if where != "" {
		built = append(built, where)
	}
	groupby, err := b.groupbys.Build(ctx)
	if err != nil {
		return "", err
	}
	if groupby != "" {
		built = append(built, groupby)
		having, err := b.having.Build(ctx)
		if err != nil {
			return "", err
		}
		if having != "" {
			built = append(built, having)
		}
	}
	order, err := b.order.Build(ctx)
	if err != nil {
		return "", err
	}
	if order != "" {
		built = append(built, order)
	}
	if b.limit > 0 {
		built = append(built, fmt.Sprintf(`LIMIT %d`, b.limit))
	}
	if b.offset > 0 {
		built = append(built, fmt.Sprintf(`OFFSET %d`, b.offset))
	}
	query := strings.TrimSpace(strings.Join(built, " "))
	if !b.unions.Empty() {
		union, err := b.unions.Build(ctx)
		if err != nil {
			return "", err
		}
		query = strings.TrimSpace(query + " " + union)
	}
	if b.debug {
		printDebugQuery(b.debugName, query, ctx.Args())
	}
	return query, nil
}

func printDebugQuery(name, query string, args []any) {
	prefix := name
	if prefix == "" {
		prefix = "sqlb"
	}
	interpolated, err := util.Interpolate(query, args)
	if err != nil {
		fmt.Printf("[%s] interpolating: %s\n", prefix, err)
	}
	fmt.Printf("[%s] %s\n", prefix, interpolated)
}

func (b *SelectBuilder) buildSelects(ctx *sqlf.Context) (string, error) {
	if b.distinct {
		b.selects.SetPrefix("SELECT DISTINCT")
	} else {
		b.selects.SetPrefix("SELECT")
	}
	sel, err := b.selects.Build(ctx)
	if err != nil {
		return "", err
	}
	if sel == "" {
		return "", fmt.Errorf("no columns selected")
	}
	return sel, nil
}

type selectBuilderDependencies struct {
	queryDeps  *dependencies
	cteDeps    map[string]bool
	unresolved *dependencies
}

// collectDependencies collects the dependencies of the tables.
func (b *SelectBuilder) collectDependencies() (*selectBuilderDependencies, error) {
	if b.deps != nil {
		return b.deps, nil
	}
	queryDeps, err := b.from.CollectDependencies(&fromBuilderMeta{
		DebugName: b.debugName,
		DependOnMe: []sqlf.Builder{
			b.selects,
			b.where,
			b.order,
			b.groupbys,
			b.having,
			b.unions,
		},
		Distinct:   b.distinct,
		HasGroupBy: !b.groupbys.Empty(),
	})
	if err != nil {
		return nil, err
	}
	cteRequired, cteUnresolved, err := b.ctes.CollectDependenciesForDeps(queryDeps)
	if err != nil {
		return nil, err
	}
	r := &selectBuilderDependencies{
		queryDeps:  queryDeps,
		cteDeps:    cteRequired,
		unresolved: cteUnresolved,
	}
	b.deps = r
	// if b.debug && b.debugName != "" {
	// 	fmt.Printf("[%s] unresolved: %s\n", b.debugName, util.Map(
	// 		util.MapKeys(r.unresolved),
	// 		func(t Table) string { return t.Name },
	// 	))
	// }
	return r, nil
}
