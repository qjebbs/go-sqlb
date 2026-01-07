package sqlb

import (
	"context"
	"strings"

	"github.com/qjebbs/go-sqlf/v4"
)

// BuildQuery builds the query.
func (b *DeleteBuilder) BuildQuery(style sqlf.BindStyle) (query string, args []any, err error) {
	return b.BuildQueryContext(context.Background(), style)
}

// BuildQueryContext builds the query with the given context.
func (b *DeleteBuilder) BuildQueryContext(ctx context.Context, style sqlf.BindStyle) (query string, args []any, err error) {
	buildCtx := sqlf.NewContext(ctx, style)
	query, err = b.buildInternal(buildCtx)
	if err != nil {
		return "", nil, err
	}
	args = buildCtx.Args()
	return query, args, nil
}

// Build implements sqlf.Builder
func (b *DeleteBuilder) Build(ctx *sqlf.Context) (query string, err error) {
	return b.buildInternal(ctx)
}

// Debug enables debug mode which prints the interpolated query to stdout.
func (b *DeleteBuilder) Debug(name ...string) *DeleteBuilder {
	b.debugger.Debug(name...)
	return b
}

// buildInternal builds the query with the selects.
func (b *DeleteBuilder) buildInternal(ctx *sqlf.Context) (string, error) {
	if b == nil {
		return "", nil
	}
	built := make([]string, 0)
	// Delete target
	built = append(built, "DELETE FROM")
	built = append(built, b.target)
	where, err := b.where.Build(ctx)
	if err != nil {
		return "", err
	}
	if where != "" {
		built = append(built, where)
	}
	query := strings.TrimSpace(strings.Join(built, " "))
	b.debugger.printIfDebug(query, ctx.Args())
	return query, nil
}
