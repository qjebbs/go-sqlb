package sqlb

import (
	"github.com/qjebbs/go-sqlf/v4"
	"github.com/qjebbs/go-sqlf/v4/util"
)

// Where add a condition.  e.g.:
//
//	b.Where(
//		sqlf.F("? = ?", a.Column("id"), 1),
//	)
func (b *QueryBuilder) Where(s sqlf.Builder) *QueryBuilder {
	if s == nil {
		return b
	}
	b.resetDepTablesCache()
	b.conditions = append(b.conditions, s)
	return b
}

// Where2 is a helper func similar to Where(), which adds a simple where condition. e.g.:
//
//	b.Where2(column, "=", 1)
//
// it's equivalent to:
//
//	b.Where(
//		sqlf.F("? = ?", column, 1),
//	)
func (b *QueryBuilder) Where2(column sqlf.Builder, op string, arg any) *QueryBuilder {
	b.resetDepTablesCache()
	b.conditions = append(
		b.conditions,
		sqlf.F("?"+op+"?", column, arg),
	)
	return b
}

// WhereIn adds a where IN condition like `t.id IN (1,2,3)`
func (b *QueryBuilder) WhereIn(column sqlf.Builder, list any) *QueryBuilder {
	return b.Where(
		sqlf.F(
			"? IN (?)",
			column,
			sqlf.JoinArgs(", ", util.FlattenArgs(list)...),
		),
	)
}

// WhereNotIn adds a where NOT IN condition like `t.id NOT IN (1,2,3)`
func (b *QueryBuilder) WhereNotIn(column sqlf.Builder, list any) *QueryBuilder {
	return b.Where(
		sqlf.F(
			"? NOT IN (?)",
			column,
			sqlf.JoinArgs(", ", util.FlattenArgs(list)...),
		),
	)
}
