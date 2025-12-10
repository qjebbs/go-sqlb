package sqlb

import (
	"fmt"

	"github.com/qjebbs/go-sqlf/v4"
)

// Order is the sorting order.
type Order uint

// orders
const (
	OrderAsc Order = iota
	OrderAscNullsFirst
	OrderAscNullsLast
	OrderDesc
	OrderDescNullsFirst
	OrderDescNullsLast
)

var orders = []string{
	"ASC",
	"ASC NULLS FIRST",
	"ASC NULLS LAST",
	"DESC",
	"DESC NULLS FIRST",
	"DESC NULLS LAST",
}

type orderItem struct {
	column sqlf.Builder
	order  Order
}

// OrderBy set the sorting order. the order can be "ASC", "DESC", "ASC NULLS FIRST" or "DESC NULLS LAST"
//
// !!! Make sure the columns are built from sqlb.Table to have their dependencies tracked.
//
//	foo := sqlb.NewTable("foo")
//	b.OrderBy(foo.Column("bar"), sqlb.OrderAsc)
func (b *QueryBuilder) OrderBy(column sqlf.Builder, order Order) *QueryBuilder {
	b.resetDepTablesCache()
	b.orders = append(b.orders, &orderItem{column: column, order: order})
	return b
}

func (b *QueryBuilder) buildOrders(ctx *sqlf.Context) (string, error) {
	builders := make([]sqlf.Builder, 0, len(b.orders))
	for _, item := range b.orders {
		if item.order > OrderDescNullsLast {
			b.pushError(fmt.Errorf("invalid order: %d", item.order))
			continue
		}
		builders = append(builders, sqlf.F(
			"? "+orders[item.order],
			item.column,
		))
	}
	f := sqlf.Prefix(
		"ORDER BY",
		sqlf.Join(", ", builders...),
	)
	return f.Build(ctx)
}
