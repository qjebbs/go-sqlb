// Package sqlb sqlb is a powerful, flexible SQL builder for Go.
// It helps you programmatically construct complex SQL queries with
// full transparency and zero hidden behavior.
//   - Chainable, composable SQL builders for SELECT, INSERT, UPDATE, DELETE, and more
//   - Support for advanced SQL features: WITH-CTE, JOIN, subqueries, expressions, etc.
//   - Full control over query structure, no hidden magic, no forced conventions
//   - Works seamlessly with any database/sql driver
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
