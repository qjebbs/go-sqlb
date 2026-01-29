package sqlb

import (
	"context"
	"errors"

	"github.com/qjebbs/go-sqlb/dialect"
	"github.com/qjebbs/go-sqlf/v4"
)

var defaultDialect = dialect.PostgreSQL{}

var _ Context = (*defaultCtx)(nil)

type defaultCtx struct {
	sqlf.Context

	// cached values
	d dialect.Dialect
}

// Dialect returns the dialect.
func (c *defaultCtx) Dialect() dialect.Dialect {
	if c.d == nil {
		c.d = c.BaseDialect().(dialect.Dialect)
	}
	return c.d
}

// NewContext returns a new Context with an argument store for the given dialect.
// If no store is provided, a new one is created using the dialect's NewArgStore method.
func NewContext(parent context.Context, dialect dialect.Dialect) Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	if dialect == nil {
		dialect = defaultDialect
	}
	return &defaultCtx{
		Context: sqlf.NewContext(parent, dialect),
	}
}

// ContextWithValue returns a new Context with the given value added to the context's value store.
func ContextWithValue(parent Context, key, value any) Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return &defaultCtx{
		Context: sqlf.ContextWithValue(unwrapContext(parent), key, value),
	}
}

// ContextWithNewArgStore returns a new context with a new ArgStore created from the dialect in the parent context.
//
// It's useful for creating sub-contexts that need their own ArgStore, like what sqlf.Build() does.
func ContextWithNewArgStore(parent Context) Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return &defaultCtx{
		Context: sqlf.ContextWithNewArgStore(unwrapContext(parent)),
	}
}

// unwrapContext extracts *defaultCtx.Context to avoid double wrapping.
func unwrapContext(ctx Context) sqlf.Context {
	if ctx, ok := ctx.(*defaultCtx); ok {
		// optimize for the common case
		return ctx.Context
	}
	return ctx
}

// ContextUpgrade upgrades a sqlf.Context to sqlb.Context.
func ContextUpgrade(ctx sqlf.Context) (Context, error) {
	if uc, ok := ctx.(Context); ok {
		return uc, nil
	}
	return nil, errors.New("the context does not imlpements sqlb.Context, consider create the context with sqlb.NewContext")
}
