package sqlb

import (
	"github.com/qjebbs/go-sqlf/v4"
	"github.com/qjebbs/go-sqlf/v4/util"
)

// Having add a having condition.
//
// !!! Make sure the columns are built from sqlb.Table to have their dependencies tracked.
//
//	foo := sqlb.NewTable("foo")
//	b.Having(sqlf.F(
//		"? = ?", foo.Column("id"), 1,
//	))
func (b *SelectBuilder) Having(s sqlf.Builder) *SelectBuilder {
	if s == nil {
		return b
	}
	b.resetDepTablesCache()
	b.havings = append(b.havings, s)
	return b
}

// Having2 is a helper func similar to Having(), which adds a simple where condition.
//
// !!! Make sure the columns are built from sqlb.Table to have their dependencies tracked.
//
//	foo := sqlb.NewTable("foo")
//	b.Having2(foo.Column("id"), "=", 1)
//
// equivalent to:
//
//	b.Having(sqlf.F(
//		"? = ?", foo.Column("id"), 1,
//	))
func (b *SelectBuilder) Having2(column sqlf.Builder, op string, arg any) *SelectBuilder {
	b.resetDepTablesCache()
	b.havings = append(
		b.havings,
		sqlf.F("?"+op+"?", column, arg),
	)
	return b
}

// HavingIn adds a where IN condition like `t.id IN (1,2,3)`
//
// !!! Make sure the columns are built from sqlb.Table to have their dependencies tracked.
func (b *SelectBuilder) HavingIn(column sqlf.Builder, list any) *SelectBuilder {
	return b.Having(
		sqlf.F(
			"? IN (?)",
			column,
			sqlf.JoinArgs(", ", util.FlattenArgs(list)...),
		),
	)
}

// HavingNotIn adds a where NOT IN condition like `t.id NOT IN (1,2,3)`
//
// !!! Make sure the columns are built from sqlb.Table to have their dependencies tracked.
func (b *SelectBuilder) HavingNotIn(column sqlf.Builder, list any) *SelectBuilder {
	return b.Having(
		sqlf.F(
			"? NOT IN (?)",
			column,
			sqlf.JoinArgs(", ", util.FlattenArgs(list)...),
		),
	)
}
