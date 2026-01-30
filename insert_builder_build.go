package sqlb

import (
	"fmt"
	"strings"

	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

// Build builds the query.
func (b *InsertBuilder) Build(ctx Context) (query string, args []any, err error) {
	return sqlf.Build(ctx, b)
}

// BuildTo implements sqlf.Builder
func (b *InsertBuilder) BuildTo(ctx sqlf.Context) (query string, err error) {
	uCtx, err := contextUpgrade(ctx)
	if err != nil {
		return "", err
	}
	return b.buildInternal(uCtx)
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
func (b *InsertBuilder) buildInternal(ctx Context) (string, error) {
	if b == nil {
		return "", nil
	}
	caps := ctx.Dialect().Capabilities()
	if b.target.IsZero() {
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
		with, err := b.ctes.BuildRequired(ctx, ctes)
		if err != nil {
			return "", err
		}
		if with != "" {
			built = append(built, with)
		}
	}
	r, err := sqlf.F("INSERT INTO ?", b.target).BuildTo(ctx)
	if err != nil {
		return "", fmt.Errorf("build insert target: %w", err)
	}
	built = append(built, r)
	if len(b.columns) > 0 {
		cols, err := sqlf.F("(?)", sqlf.Join(b.columns, ", ")).BuildTo(ctx)
		if err != nil {
			return "", fmt.Errorf("build insert columns: %w", err)
		}
		built = append(built, cols)
	}
	// returning clause
	if len(b.returning) > 0 && caps.SupportsOutputInserted {
		returning, err := sqlf.F("OUTPUT ?", sqlf.Join(
			b.returning, ", ",
		)).BuildTo(ctx)
		if err != nil {
			return "", fmt.Errorf("build returning clause: %w", err)
		}
		built = append(built, returning)
	}
	if len(b.values) > 0 {
		valueBuilders := sqlf.Join(util.Map(b.values, func(values []any) sqlf.Builder {
			return sqlf.F("(?)", sqlf.JoinMixed(values, ", "))
		}), ", ")
		valuesStr, err := sqlf.Prefix("VALUES", valueBuilders).BuildTo(ctx)
		if err != nil {
			return "", fmt.Errorf("build insert values: %w", err)
		}
		built = append(built, valuesStr)
	}
	if b.selects != nil {
		sel, err := b.selects.BuildTo(ctx)
		if err != nil {
			return "", fmt.Errorf("build insert from select: %w", err)
		}
		built = append(built, sel)
	}
	// conflict handling
	switch {
	case caps.SupportsOnConflict:
		if len(b.conflictOn) > 0 {
			conflictTarget, err := sqlf.F(
				"ON CONFLICT (?)",
				sqlf.Join(b.conflictOn, ", "),
			).BuildTo(ctx)
			if err != nil {
				return "", fmt.Errorf("build conflict target: %w", err)
			}
			built = append(built, conflictTarget)
			if len(b.conflictDo) == 0 {
				built = append(built, "DO NOTHING")
			} else {
				conflictActions, err := sqlf.Join(b.conflictDo, ", ").BuildTo(ctx)
				if err != nil {
					return "", fmt.Errorf("build conflict do actions: %w", err)
				}
				built = append(built, "DO UPDATE SET")
				built = append(built, conflictActions)
			}
		}
	case caps.SupportsOnDuplicateKeyUpdate:
		if len(b.conflictDo) > 0 {
			built = append(built, "ON DUPLICATE KEY UPDATE")
			conflictActions, err := sqlf.Join(b.conflictDo, ", ").BuildTo(ctx)
			if err != nil {
				return "", fmt.Errorf("build conflict do actions: %w", err)
			}
			built = append(built, conflictActions)
		}
	default:
		if len(b.conflictOn) > 0 || len(b.conflictDo) > 0 {
			return "", fmt.Errorf("ON CONFLICT / DUPLICATE KEY is not supported for dialact %t", ctx.Dialect())
		}
	}

	// returning clause
	if len(b.returning) > 0 {
		switch {
		case caps.SupportsReturning:
			returning, err := sqlf.F("RETURNING ?", sqlf.Join(b.returning, ", ")).BuildTo(ctx)
			if err != nil {
				return "", fmt.Errorf("build returning clause: %w", err)
			}
			built = append(built, returning)
		case caps.SupportsOutputInserted:
			// already built
		default:
			return "", fmt.Errorf("returning is not supported for dialact %T", ctx.BaseDialect())
		}
	}
	query := strings.TrimSpace(strings.Join(built, " "))
	b.debugger.printIfDebug(ctx, query, ctx.Args())
	return query, nil
}

// collectDependencies collects the dependencies of the tables.
func (b *InsertBuilder) collectDependencies(ctx Context) (*dependencies, error) {
	myDeps := newDependencies(b.name)

	// use a separate context to avoid polluting args
	ctx = sqlf.ContextWithNewArgStore(ctx)
	depCtx := contextWithDependencies(ctx, myDeps)
	_, err := b.selects.BuildTo(depCtx)
	if err != nil {
		return nil, err
	}
	return myDeps, nil
}
