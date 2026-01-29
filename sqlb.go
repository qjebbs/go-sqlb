// Package sqlb sqlb is a powerful SQL builder and struct mapper. It provides,
//   - SQL builders to craft complex queries.
//   - Effortlessly map query results to Go structs.
//   - Declarative automation of common CRUD operations.
//
// With sqlb, All queries are explicitly coded or declared, there is no hidden behavior,
// preserving both flexibility and transparency in your database interactions.
package sqlb

import (
	"github.com/qjebbs/go-sqlb/dialect"
	"github.com/qjebbs/go-sqlf/v4"
)

// Builder is the interface for sql builders.
type Builder interface {
	// Build builds and returns the query and args with the given context.
	Build(ctx Context) (query string, args []any, err error)
}

// Context is the context for building SQL queries in sqlb.
type Context interface {
	sqlf.Context
	Dialect() dialect.Dialect
}

// Build builds a Builder into a query string and its corresponding arguments.
// It creates a new context to ensure that the original context is not modified.
func Build(ctx Context, b sqlf.Builder) (query string, args []any, err error) {
	if b == nil {
		return "", nil, nil
	}
	// make sure not committing args to the original context
	ctx = ContextWithNewArgStore(ctx)
	query, err = b.BuildTo(ctx)
	if err != nil {
		return "", nil, err
	}
	return query, ctx.Args(), nil
}
