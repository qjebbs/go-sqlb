package sqlb

import (
	"strings"

	"github.com/qjebbs/go-sqlf/v4"
)

// Build builds the query.
func (b *DeleteBuilder) Build(ctx Context) (query string, args []any, err error) {
	return Build(ctx, b)
}

// BuildTo implements sqlf.Builder
func (b *DeleteBuilder) BuildTo(ctx sqlf.Context) (query string, err error) {
	uCtx, err := ContextUpgrade(ctx)
	if err != nil {
		return "", err
	}
	return b.buildInternal(uCtx)
}

// Debug enables debug mode which prints the interpolated query to stdout.
func (b *DeleteBuilder) Debug(name ...string) *DeleteBuilder {
	b.debugger.Debug(name...)
	return b
}

// buildInternal builds the query with the selects.
func (b *DeleteBuilder) buildInternal(ctx Context) (string, error) {
	if b == nil {
		return "", nil
	}
	built := make([]string, 0)
	// Delete target
	r, err := sqlf.F("DELETE FROM ? ", b.target).BuildTo(ctx)
	if err != nil {
		return "", err
	}
	built = append(built, r)
	where, err := b.where.BuildTo(ctx)
	if err != nil {
		return "", err
	}
	if where != "" {
		built = append(built, where)
	}
	query := strings.TrimSpace(strings.Join(built, " "))
	b.debugger.printIfDebug(ctx, query, ctx.Args())
	return query, nil
}
