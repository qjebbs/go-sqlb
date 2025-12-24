package sqlb

import (
	"errors"

	"github.com/qjebbs/go-sqlf/v4"
)

// From set the from table.
func (b *UpdateBuilder) From(t Table) *UpdateBuilder {
	if b.dialact == DialectMySQL {
		b.pushError(errors.New("MySQL does not support FROM clause in UPDATE statements"))
		return b
	}
	b.resetDepTablesCache()
	b.from.From(t)
	return b
}

// InnerJoin append a inner join table.
func (b *UpdateBuilder) InnerJoin(t Table, on *sqlf.Fragment) *UpdateBuilder {
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
func (b *UpdateBuilder) LeftJoin(t Table, on *sqlf.Fragment) *UpdateBuilder {
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
func (b *UpdateBuilder) LeftJoinOptional(t Table, on *sqlf.Fragment) *UpdateBuilder {
	b.from.Join("LEFT JOIN", t, on, true, true)
	return b
}

// RightJoin append / replace a right join table.
func (b *UpdateBuilder) RightJoin(t Table, on *sqlf.Fragment) *UpdateBuilder {
	b.from.Join("RIGHT JOIN", t, on, false, false)
	return b
}

// FullJoin append / replace a full join table.
func (b *UpdateBuilder) FullJoin(t Table, on *sqlf.Fragment) *UpdateBuilder {
	b.from.Join("FULL JOIN", t, on, false, false)
	return b
}

// CrossJoin append / replace a cross join table.
func (b *UpdateBuilder) CrossJoin(t Table) *UpdateBuilder {
	b.from.Join("CROSS JOIN", t, nil, false, false)
	return b
}

// With adds a fragment as common table expression,
// the built query of s should be a subquery.
//
// !!! UpdateBuilder tracks dependencies of CTEs with the help of sqlb.Table.
//
// If the CTE builder depends on other CTEs,
// make sure all the table references are built from sqlb.Table,
// for example:
//
//	foo := sqlb.NewTable("foo")
//	bar := sqlb.NewTable("bar")
//	builderFoo := sqlf.F("SELECT * FROM users WHERE active")
//	// the dependency is tracked only if the foo (of sqlb.Table) is used
//	builderBar := sqlf.F("SELECT * FROM ?", foo)
//	builder := sqlb.NewUpdateBuilder().
//		With(foo, builderFoo).With(bar, builderBar).
//		Update(bar.Column("*")).From(bar)
func (b *UpdateBuilder) With(name Table, builder sqlf.Builder) *UpdateBuilder {
	b.resetDepTablesCache()
	b.ctes.With(name, builder)
	return b
}
