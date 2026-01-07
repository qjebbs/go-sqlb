// Package sqlb sqlb is a powerful SQL builder and struct mapper. It provides,
//   - SQL builders to craft complex queries.
//   - Effortlessly map query results to Go structs.
//   - Declarative automation of common CRUD operations.
//
// With sqlb, All queries are explicitly coded or declared, there is no hidden behavior,
// preserving both flexibility and transparency in your database interactions.
package sqlb

import (
	"context"

	"github.com/qjebbs/go-sqlf/v4"
)

// Builder is the interface for sql builders.
type Builder interface {
	// BuildQueryContext builds and returns the query and args with the given context.
	BuildQueryContext(ctx context.Context, style sqlf.BindStyle) (query string, args []any, err error)
}
