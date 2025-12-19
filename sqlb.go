// Package sqlb provides a complex SQL query builder shipped with
// WITH-CTE / JOIN Elimination abilities, while
// 'github.com/qjebbs/go-sqlf' is the underlying foundation.
package sqlb

import (
	"github.com/qjebbs/go-sqlb/internal/clauses"
	"github.com/qjebbs/go-sqlf/v4"
)

// Builder is the interface for sql builders.
type Builder interface {
	// BuildQuery builds and returns the query and args.
	BuildQuery(bindVarStyle sqlf.BindStyle) (query string, args []any, err error)
}

// Table is the table name with optional alias.
type Table = clauses.Table

// NewTable returns a new Table.
//
// Table is a sqlf.Builder, but builds only the applied name,
// since it's more common to use it to build column references, e.g.:
//
//	t := NewTable("table", "t")
//	sqlf.F("?.id", t)  // t.id
//
// If you want to build fragments like `table As t`, use t.TableAs().
//
//	sqlf.F("LEFT JOIN ?", t.TableAs()) // LEFT JOIN table AS t
var NewTable = clauses.NewTable

// Order is the sorting order.
type Order = clauses.Order

// orders
const (
	OrderAsc            Order = clauses.OrderAsc
	OrderAscNullsFirst        = clauses.OrderAscNullsFirst
	OrderAscNullsLast         = clauses.OrderAscNullsLast
	OrderDesc                 = clauses.OrderDesc
	OrderDescNullsFirst       = clauses.OrderDescNullsFirst
	OrderDescNullsLast        = clauses.OrderDescNullsLast
)
