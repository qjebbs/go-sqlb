package sqlb

import (
	"context"
	"errors"
	"fmt"

	"github.com/qjebbs/go-sqlb/dialect"
	"github.com/qjebbs/go-sqlf/v4"
)

// errors
var (
	ErrNoDialectInContext = errors.New("no sqlb/dialect.Dialect in the context, please use sqlb.NewContext or sqlb.ContextWithDialect to create a context with dialect")
	ErrInvalidDialect     = errors.New("dialect in the context does not implement sqlb/dialect.Dialect")
)

// NewContext returns a new Context with an argument store for the given dialect.
// If no store is provided, a new one is created using the dialect's NewArgStore method.
func NewContext(parent context.Context, dialect dialect.Dialect) *sqlf.Context {
	return sqlf.NewContext(parent, dialect)
}

// ContextWithDialect returns a new context with the given dialect.
func ContextWithDialect(ctx context.Context, dialect dialect.Dialect) *sqlf.Context {
	return sqlf.ContextWithDialect(ctx, dialect)
}

// DialectFromContext retrieves the dialect from the context.
func DialectFromContext(ctx context.Context) (dialect.Dialect, error) {
	d, ok := sqlf.DialectFromContext(ctx)
	if !ok {
		return nil, ErrNoDialectInContext
	}
	if dialect, ok := dialect.Upgrade(d); ok {
		return dialect, nil
	}
	return nil, fmt.Errorf("%T: %w", d, ErrInvalidDialect)
}
