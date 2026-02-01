package sqlb

import (
	"context"
	"errors"

	"github.com/qjebbs/go-sqlb/dialect"
	"github.com/qjebbs/go-sqlf/v4"
)

// ContextWithValue is a sqlb version of sqlf.ContextWithValue.
// It returns a new Context derived from parent with the key and value set.
var ContextWithValue = sqlf.ContextWithValue[Context]

// ContextWithNewArgStore is a sqlb version of sqlf.ContextWithNewArgStore.
// It returns a new context with a new ArgStore created from the dialect in the parent context.
var ContextWithNewArgStore = sqlf.ContextWithNewArgStore[Context]

var defaultDialect = dialect.PostgreSQL{}

// NewContext returns a new Context with an argument store for the given dialect.
// If no store is provided, a new one is created using the dialect's NewArgStore method.
func NewContext(parent context.Context, dialect dialect.Dialect) Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	if dialect == nil {
		dialect = defaultDialect
	}
	return newDeafultCtx(parent, dialect)
}

// contextUpgrade upgrades a sqlf.Context to sqlb.Context.
// It's a helper for `BuildTo(ctx sqlf.Context)` functions who implement sqlf.Builder but recieve and accept sqlb.Context only.
func contextUpgrade(ctx sqlf.Context) (Context, error) {
	if uc, ok := ctx.(Context); ok {
		return uc, nil
	}
	return nil, errors.New("the context does not implement sqlb.Context, consider creating the context with sqlb.NewContext")
}

var _ Context = (*defaultCtx)(nil)

type defaultCtx struct {
	sqlf.Context

	// cached values
	d dialect.Dialect
}

func newDeafultCtx(parent context.Context, dialect dialect.Dialect) *defaultCtx {
	return &defaultCtx{
		Context: sqlf.NewContext(parent, dialect),
	}
}

func (c *defaultCtx) WithContextFunc(fn sqlf.ContextDeriver) sqlf.Context {
	return &defaultCtx{
		Context: c.Context.WithContextFunc(fn),
	}
}

// Dialect returns the dialect.
func (c *defaultCtx) Dialect() dialect.Dialect {
	// no need to check nil c, since user cannot create defaultCtx directly.
	if c.d == nil {
		// no need to check type assertion error, since the creation of c ensures the dialect is of the correct type.
		c.d = c.BaseDialect().(dialect.Dialect)
	}
	return c.d
}
