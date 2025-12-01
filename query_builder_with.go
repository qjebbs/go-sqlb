package sqlb

import "github.com/qjebbs/go-sqlf/v4"

// With adds a fragment as common table expression,
// the built query of s should be a subquery.
//
// !!! QueryBuilder tracks dependencies of CTEs with the help of sqlb.Table.
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
//	builder := sqlb.NewQueryBuilder().
//		With(foo, builderFoo).With(bar, builderBar).
//		Select(bar.Column("*")).From(bar)
func (b *QueryBuilder) With(name Table, builder sqlf.Builder) *QueryBuilder {
	b.resetDepTablesCache()
	b.ctes.With(name, builder)
	return b
}
