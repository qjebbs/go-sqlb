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

// LeftJoin append / replace a left join table.
// The query builder will remove LEFT JOIN if no columns from the joined table are referenced
// and SELECT DISTINCT or GROUP BY is enabled.
//
// !!! To ensure dependencies are collected as expected, do not hard-code table names.
// Always build tables using Table. For example:
//
//	// GOOD: The dependency of foo will be collected.
//	foo := sqlb.NewTable("foo")
//	b.SELECT(sqlf.F("?.id", foo))
//
//	// BAD: The dependency of foo will NOT be collected.
//	b.SELECT(sqlf.F("foo.id"))
func (b *SelectBuilder) LeftJoin(t Table, on *sqlf.Fragment) *SelectBuilder {
	b.from.Join("LEFT JOIN", t, on, true, false)
	return b
}

// LeftJoinOptional append / replace a left join table, which is forced to be eliminated
// if no columns from the joined table are referenced
// in the query, no matter if SELECT DISTINCT or GROUP BY is enabled.
//
// !!! It's users responsibility to ensure the elemination has no side effects,
// In other words, it's safe only when the left table has one-to-one / one-to-zero
// relationship with the joined right table.
//
// !!! To ensure dependencies are collected as expected, do not hard-code table names.
// Always build tables using Table. For example:
//
//	// GOOD: The dependency of foo will be collected.
//	foo := sqlb.NewTable("foo")
//	b.SELECT(sqlf.F("?.id", foo))
//
//	// BAD: The dependency of foo will NOT be collected.
//	b.SELECT(sqlf.F("foo.id"))
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
