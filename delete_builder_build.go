package sqlb

import (
	"strings"

	"github.com/qjebbs/go-sqlf/v4"
)

// BuildQuery builds the query.
func (b *DeleteBuilder) BuildQuery(style sqlf.BindStyle) (query string, args []any, err error) {
	ctx := sqlf.NewContext(style)
	query, err = b.buildInternal(ctx)
	if err != nil {
		return "", nil, err
	}
	args = ctx.Args()
	return query, args, nil
}

// Build implements sqlf.Builder
func (b *DeleteBuilder) Build(ctx *sqlf.Context) (query string, err error) {
	return b.buildInternal(ctx)
}

// Debug enables debug mode which prints the interpolated query to stdout.
func (b *DeleteBuilder) Debug(name ...string) *DeleteBuilder {
	b.debug = true
	b.debugName = strings.Replace(strings.Join(name, "_"), " ", "_", -1)
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
	if b.debug {
		printDebugQuery(b.debugName, query, ctx.Args())
	}
	return query, nil
}
