// Package sqlb provides a complex SQL query builder shipped with
// WITH-CTE / JOIN Elimination capabilities, while
// 'github.com/qjebbs/go-sqlf' is the underlying foundation.
package sqlb

import "github.com/qjebbs/go-sqlf/v4"

// Builder is the interface for sql builders.
type Builder interface {
	// BuildQuery builds and returns the query and args.
	BuildQuery(bindVarStyle sqlf.BindStyle) (query string, args []any, err error)
}
