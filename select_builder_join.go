package sqlb

import (
	"github.com/qjebbs/go-sqlf/v4"
)

// From set the from table.
func (b *SelectBuilder) From(t Table) *SelectBuilder {
	b.resetDepTablesCache()
	b.from.From(t)
	return b
}

// InnerJoin append a inner join table.
func (b *SelectBuilder) InnerJoin(t Table, on *sqlf.Fragment) *SelectBuilder {
	b.from.Join("INNER JOIN", t, on, false, false)
	return b
}

// LeftJoin append / replace a left join table,
// which will be automatically eliminated if all the conditions below are met:
//   - Pruning is enabled by `b.EnableElimination()` or parent builders
//   - SELECT DISTINCT or GROUP BY is enabled.
//   - The table is not referenced anywhere in the query
func (b *SelectBuilder) LeftJoin(t Table, on *sqlf.Fragment) *SelectBuilder {
	b.from.Join("LEFT JOIN", t, on, true, false)
	return b
}

// LeftJoinOptional append / replace a left join table,
// which will be automatically eliminated if all the conditions below are met:
//   - Pruning is enabled by `b.EnableElimination()` or parent builders
//   - The table is not referenced anywhere in the query
//
// !!! Unlike LeftJoin, a LeftJoinOptional table can be eliminated even when
// SELECT DISTINCT or GROUP BY is not used.
// It's users responsibility to ensure the elemination has no side effects,
// In other words, it's safe only when the left table has one-to-one / one-to-zero
// relationship with the joined right table.
func (b *SelectBuilder) LeftJoinOptional(t Table, on *sqlf.Fragment) *SelectBuilder {
	b.from.Join("LEFT JOIN", t, on, true, true)
	return b
}

// RightJoin append / replace a right join table.
func (b *SelectBuilder) RightJoin(t Table, on *sqlf.Fragment) *SelectBuilder {
	b.from.Join("RIGHT JOIN", t, on, false, false)
	return b
}

// FullJoin append / replace a full join table.
func (b *SelectBuilder) FullJoin(t Table, on *sqlf.Fragment) *SelectBuilder {
	b.from.Join("FULL JOIN", t, on, false, false)
	return b
}

// CrossJoin append / replace a cross join table.
func (b *SelectBuilder) CrossJoin(t Table) *SelectBuilder {
	b.from.Join("CROSS JOIN", t, nil, false, false)
	return b
}
