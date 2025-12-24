package sqlb

import (
	"fmt"

	"github.com/qjebbs/go-sqlf/v4"
)

var _ sqlf.Builder = (*clauseOrderBy)(nil)

// clauseOrderBy represents a SQL ORDER BY clause.
type clauseOrderBy struct {
	orders []*orderItem
}

// orderItem represents a single order by item.
type orderItem struct {
	column sqlf.Builder
	order  Order
}

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

// newOrderBy creates a new OrderBy instance.
func newOrderBy() *clauseOrderBy {
	return &clauseOrderBy{}
}

// Add adds an order item.
func (o *clauseOrderBy) Add(column sqlf.Builder, order Order) *clauseOrderBy {
	o.orders = append(o.orders, &orderItem{
		column: column,
		order:  order,
	})
	return o
}

// Build implements sqlf.Builder
func (o *clauseOrderBy) Build(ctx *sqlf.Context) (string, error) {
	if o == nil || len(o.orders) == 0 {
		return "", nil
	}
	builders := make([]sqlf.Builder, 0, len(o.orders))
	for _, item := range o.orders {
		if item.order > OrderDescNullsLast {
			return "", fmt.Errorf("invalid order: %d", item.order)
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
