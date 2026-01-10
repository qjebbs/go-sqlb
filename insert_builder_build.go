package sqlb

import (
	"context"
	"fmt"
	"strings"

	"github.com/qjebbs/go-sqlb/internal/util"
	myutil "github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

// BuildQuery builds the query.
func (b *InsertBuilder) BuildQuery(style sqlf.BindStyle) (query string, args []any, err error) {
	return b.BuildQueryContext(context.Background(), style)
}

// BuildQueryContext builds the query with the given context.
func (b *InsertBuilder) BuildQueryContext(ctx context.Context, style sqlf.BindStyle) (query string, args []any, err error) {
	buildCtx := sqlf.NewContext(ctx, style)
	query, err = b.buildInternal(buildCtx)
	if err != nil {
		return "", nil, err
	}
	args = buildCtx.Args()
	return query, args, nil
}

// Build implements sqlf.Builder
func (b *InsertBuilder) Build(ctx *sqlf.Context) (query string, err error) {
	return b.buildInternal(ctx)
}

// Debug enables debug mode which prints the interpolated query to stdout.
func (b *InsertBuilder) Debug(name ...string) *InsertBuilder {
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
func (b *InsertBuilder) EnableElimination() *InsertBuilder {
	b.pruning = true
	return b
}

// buildInternal builds the query with the selects.
func (b *InsertBuilder) buildInternal(ctx *sqlf.Context) (string, error) {
	if b == nil {
		return "", nil
	}
	if b.target == "" {
		return "", fmt.Errorf("no target table specified for insert")
	}
	if b.selects == nil && len(b.values) == 0 {
		return "", fmt.Errorf("no values or select specified for insert")
	}
	if b.selects != nil && len(b.values) > 0 {
		return "", fmt.Errorf("cannot specify both select and values for insert")
	}

	ctx, pruning := decideContextPruning(ctx, b.pruning)
	built := make([]string, 0)
	if b.selects != nil && b.ctes.HasCTE() {
		var err error
		myDeps := newDependencies(b.name)
		if pruning {
			myDeps, err = b.collectDependencies(ctx)
			if err != nil {
				return "", err
			}
			if deps := dependenciesFromContext(ctx); deps != nil {
				for t := range myDeps.OuterTables {
					deps.OuterTables[t] = true
				}
				for t := range myDeps.SourceNames {
					deps.SourceNames[t] = true
				}
				// collecting dependencies only,
				// no need to build anything here
				return "", nil
			}
		}
		ctes := make(map[string]bool)
		for cte := range myDeps.SourceNames {
			ctes[cte] = true
		}
		with, err := b.ctes.BuildRequired(ctx, ctes, b.dialact)
		if err != nil {
			return "", err
		}
		if with != "" {
			built = append(built, with)
		}
	}
	built = append(built, fmt.Sprintf("INSERT INTO %s", b.target))
	if len(b.columns) > 0 {
		cols := fmt.Sprintf("(%s)", strings.Join(b.columns, ", "))
		built = append(built, cols)
	}
	// returning clause
	if len(b.returning) > 0 && b.dialact == DialectSQLServer {
		returning, err := sqlf.F("OUTPUT ?", sqlf.Join(
			", ", util.Map(b.returning, func(r string) sqlf.Builder {
				return sqlf.F("INSERTED." + r)
			})...,
		)).Build(ctx)
		if err != nil {
			return "", fmt.Errorf("build returning clause: %w", err)
		}
		built = append(built, returning)
	}
	if len(b.values) > 0 {
		valueBuilders := sqlf.Join(", ", myutil.Map(b.values, func(values []any) sqlf.Builder {
			return sqlf.F("(?)", sqlf.JoinMixed(", ", values...))
		})...)
		valuesStr, err := sqlf.Prefix("VALUES", valueBuilders).Build(ctx)
		if err != nil {
			return "", fmt.Errorf("build insert values: %w", err)
		}
		built = append(built, valuesStr)
	}
	if b.selects != nil {
		sel, err := b.selects.Build(ctx)
		if err != nil {
			return "", fmt.Errorf("build insert from select: %w", err)
		}
		built = append(built, sel)
	}
	// conflict handling
	switch b.dialact {
	case DialectPostgres, DialectSQLite:
		if len(b.conflictOn) > 0 {
			conflictTarget := fmt.Sprintf("ON CONFLICT (%s)", strings.Join(b.conflictOn, ", "))
			built = append(built, conflictTarget)
			if len(b.conflictDo) == 0 {
				built = append(built, "DO NOTHING")
			} else {
				conflictActions, err := sqlf.Join(", ", b.conflictDo...).Build(ctx)
				if err != nil {
					return "", fmt.Errorf("build conflict do actions: %w", err)
				}
				built = append(built, "DO UPDATE SET")
				built = append(built, conflictActions)
			}
		}
	case DialectMySQL:
		if len(b.conflictDo) > 0 {
			built = append(built, "ON DUPLICATE KEY UPDATE")
			conflictActions, err := sqlf.Join(", ", b.conflictDo...).Build(ctx)
			if err != nil {
				return "", fmt.Errorf("build conflict do actions: %w", err)
			}
			built = append(built, conflictActions)
		}
	default:
		if len(b.conflictOn) > 0 || len(b.conflictDo) > 0 {
			return "", fmt.Errorf("ON CONFLICT is not supported for %s", b.dialact.String())
		}
	}

	// returning clause
	if len(b.returning) > 0 {
		switch b.dialact {
		case DialectPostgres, DialectSQLite:
			returning := fmt.Sprintf("RETURNING %s", strings.Join(b.returning, ", "))
			built = append(built, returning)
		case DialectSQLServer:
			// already built
		default:
			return "", fmt.Errorf("returning is not supported for %s", b.dialact.String())
		}
	}
	query := strings.TrimSpace(strings.Join(built, " "))
	b.debugger.printIfDebug(query, ctx.Args())
	return query, nil
}

// collectDependencies collects the dependencies of the tables.
func (b *InsertBuilder) collectDependencies(ctx *sqlf.Context) (*dependencies, error) {
	myDeps := newDependencies(b.name)

	// use a separate context to avoid polluting args
	ctx = sqlf.NewContext(ctx, sqlf.BindStyleQuestion)
	depCtx := contextWithDependencies(ctx, myDeps)
	_, err := b.selects.Build(depCtx)
	if err != nil {
		return nil, err
	}
	return myDeps, nil
}
