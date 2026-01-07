package sqlb

import (
	"context"
	"fmt"
	"strings"

	"github.com/qjebbs/go-sqlf/v4"
)

// BuildQuery builds the query.
func (b *UpdateBuilder) BuildQuery(style sqlf.BindStyle) (query string, args []any, err error) {
	return b.BuildQueryContext(context.Background(), style)
}

// BuildQueryContext builds the query with the given context.
func (b *UpdateBuilder) BuildQueryContext(ctx context.Context, style sqlf.BindStyle) (query string, args []any, err error) {
	buildCtx := sqlf.NewContext(ctx, style)
	query, err = b.buildInternal(buildCtx)
	if err != nil {
		return "", nil, err
	}
	args = buildCtx.Args()
	return query, args, nil
}

// Build implements sqlf.Builder
func (b *UpdateBuilder) Build(ctx *sqlf.Context) (query string, err error) {
	return b.buildInternal(ctx)
}

// Debug enables debug mode which prints the interpolated query to stdout.
func (b *UpdateBuilder) Debug(name ...string) *UpdateBuilder {
	b.debugger.Debug(name...)
	return b
}

// EnableElimination enables JOIN / CTE elimination based on dependency analysis.
// To use elimination, make sure all table references are done via Table objects.
//
// For example,
//
//	t := sqlb.NewTable("foo", "f")
//	b.Where(sqlf.F("? = ?", t.Column("id"), 1))
//	// instead of
//	b.Where(sqlf.F("f.id = ?", 1))
func (b *UpdateBuilder) EnableElimination() *UpdateBuilder {
	b.pruning = true
	return b
}

// buildInternal builds the query with the selects.
func (b *UpdateBuilder) buildInternal(ctx *sqlf.Context) (string, error) {
	if err := b.anyError(); err != nil {
		return "", b.anyError()
	}
	if b == nil {
		return "", nil
	}
	built := make([]string, 0)

	ctx, pruning := decideContextPruning(ctx, b.pruning)

	var err error
	var myDeps = &selectBuilderDependencies{}
	if pruning {
		myDeps, err = b.collectDependencies(ctx)
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
	}
	with, err := b.ctes.BuildRequired(ctx, myDeps.cteDeps)
	if err != nil {
		return "", err
	}
	if with != "" {
		built = append(built, with)
	}
	// UPDATE target
	built = append(built, "UPDATE")
	built = append(built, b.target)
	if b.dialact == DialectMySQL {
		// MySQL join goes first
		joins, err := b.from.BuildRequired(ctx, b.joinBuilderMeta(), myDeps.queryDeps)
		if err != nil {
			return "", err
		}
		if joins != "" {
			built = append(built, joins)
		}
	}
	// SET sets
	sets, err := b.sets.Build(ctx)
	if err != nil {
		return "", err
	}
	if sets == "" {
		return "", fmt.Errorf("no columns set for update")
	}
	built = append(built, sets)

	if b.dialact != DialectMySQL {
		// FROM / JOINS
		joins, err := b.from.BuildRequired(ctx, b.joinBuilderMeta(), myDeps.queryDeps)
		if err != nil {
			return "", err
		}
		if joins != "" {
			built = append(built, joins)
		}
	}
	where, err := b.where.Build(ctx)
	if err != nil {
		return "", err
	}
	if where != "" {
		built = append(built, where)
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
	query := strings.TrimSpace(strings.Join(built, " "))
	b.debugger.printIfDebug(query, ctx.Args())
	return query, nil
}

func (b *UpdateBuilder) joinBuilderMeta() *fromBuilderMeta {
	return &fromBuilderMeta{
		DebugName: b.name,
		DependOnMe: []sqlf.Builder{
			b.sets,
			b.where,
			b.order,
		},
		Distinct:   false,
		HasGroupBy: false,
	}
}

// collectDependencies collects the dependencies of the tables.
func (b *UpdateBuilder) collectDependencies(ctx *sqlf.Context) (*selectBuilderDependencies, error) {
	if b.deps != nil {
		return b.deps, nil
	}
	// use a separate context to avoid polluting args
	ctx = sqlf.NewContext(ctx, sqlf.BindStyleQuestion)
	queryDeps, err := b.from.CollectDependencies(ctx, b.joinBuilderMeta())
	if err != nil {
		return nil, err
	}
	cteRequired, cteUnresolved, err := b.ctes.CollectDependenciesForDeps(ctx, queryDeps)
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
