package sqlb

import (
	"fmt"
	"strings"

	myutil "github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

// BuildQuery builds the query.
func (b *InsertBuilder) BuildQuery(style sqlf.BindStyle) (query string, args []any, err error) {
	ctx := sqlf.NewContext(style)
	query, err = b.buildInternal(ctx)
	if err != nil {
		return "", nil, err
	}
	args = ctx.Args()
	return query, args, nil
}

// Build implements sqlf.Builder
func (b *InsertBuilder) Build(ctx *sqlf.Context) (query string, err error) {
	return b.buildInternal(ctx)
}

// Debug enables debug mode which prints the interpolated query to stdout.
func (b *InsertBuilder) Debug(name ...string) *InsertBuilder {
	b.debug = true
	b.debugName = strings.Replace(strings.Join(name, "_"), " ", "_", -1)
	return b
}

// buildInternal builds the query with the selects.
func (b *InsertBuilder) buildInternal(ctx *sqlf.Context) (string, error) {
	if b == nil {
		return "", nil
	}
	err := b.anyError()
	if err != nil {
		return "", err
	}
	if b.target.Name == "" {
		return "", fmt.Errorf("no target table specified for insert")
	}
	if b.selects == nil && len(b.values) == 0 {
		return "", fmt.Errorf("no values or select specified for insert")
	}
	if b.selects != nil && len(b.values) > 0 {
		return "", fmt.Errorf("cannot specify both select and values for insert")
	}

	clauses := make([]string, 0)
	if b.selects != nil && len(b.ctes.ctes) > 0 {
		myDeps := newDepTables(b.debugName)
		depCtx := contextWithDepTables(sqlf.NewContext(sqlf.BindStyleDollar), myDeps)
		_, err = b.selects.Build(depCtx)
		if err != nil {
			return "", err
		}

		if deps := depTablesFromContext(ctx); deps != nil {
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
		ctes := make(map[string]bool)
		for cte := range myDeps.SourceNames {
			ctes[cte] = true
		}
		with, err := b.ctes.buildRequired(ctx, ctes)
		if err != nil {
			return "", err
		}
		if with != "" {
			clauses = append(clauses, with)
		}
	}
	clauses = append(clauses, fmt.Sprintf("INSERT INTO %s", b.target.Name))
	if len(b.columns) > 0 {
		cols := fmt.Sprintf("(%s)", strings.Join(b.columns, ", "))
		clauses = append(clauses, cols)
	}
	if len(b.values) > 0 {
		valueBuilders := sqlf.Join(", ", myutil.Map(b.values, func(values []any) sqlf.Builder {
			return sqlf.F("(?)", sqlf.JoinArgs(", ", values...))
		})...)
		valuesStr, err := sqlf.Prefix("VALUES", valueBuilders).Build(ctx)
		if err != nil {
			return "", fmt.Errorf("build insert values: %w", err)
		}
		clauses = append(clauses, valuesStr)
	}
	if b.selects != nil {
		sel, err := b.selects.Build(ctx)
		if err != nil {
			return "", fmt.Errorf("build insert from select: %w", err)
		}
		clauses = append(clauses, sel)
	}
	if len(b.conflictOn) > 0 {
		conflictTarget := fmt.Sprintf("ON CONFLICT (%s)", strings.Join(b.conflictOn, ", "))
		clauses = append(clauses, conflictTarget)
		if len(b.conflictDo) == 0 {
			clauses = append(clauses, "DO NOTHING")
		} else {
			conflictActions, err := sqlf.Join(", ", b.conflictDo...).Build(ctx)
			if err != nil {
				return "", fmt.Errorf("build conflict do actions: %w", err)
			}
			clauses = append(clauses, "DO UPDATE SET")
			clauses = append(clauses, conflictActions)
		}
	}
	if len(b.returning) > 0 {
		returning := fmt.Sprintf("RETURNING %s", strings.Join(b.returning, ", "))
		clauses = append(clauses, returning)
	}
	query := strings.TrimSpace(strings.Join(clauses, " "))
	if b.debug {
		printDebugQuery(b.debugName, query, ctx.Args())
	}
	return query, nil
}
