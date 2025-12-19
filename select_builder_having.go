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
	b.resetDepTablesCache()
	b.having.Append(s)
	return b
}

// HavingEquals is a helper func similar to Having(), which adds a simple equality condition. e.g.:
func (b *SelectBuilder) HavingEquals(column sqlf.Builder, value any) *SelectBuilder {
	return b.Having(
		sqlf.F("? = ?", column, value),
	)
}

// HavingNotEquals is a helper func similar to Having(), which adds a simple not-equal condition. e.g.:
func (b *SelectBuilder) HavingNotEquals(column sqlf.Builder, value any) *SelectBuilder {
	return b.Having(
		sqlf.F("? <> ?", column, value),
	)
}

// HavingGreaterThan adds a greater-than condition like `t.id > 1`
func (b *SelectBuilder) HavingGreaterThan(column sqlf.Builder, value any) *SelectBuilder {
	return b.Having(
		sqlf.F("? > ?", column, value),
	)
}

// HavingLessThan adds a less-than condition like `t.id < 1`
func (b *SelectBuilder) HavingLessThan(column sqlf.Builder, value any) *SelectBuilder {
	return b.Having(
		sqlf.F("? < ?", column, value),
	)
}

// HavingGreaterThanOrEqual adds a greater-than-or-equal condition like `t.id >= 1`
func (b *SelectBuilder) HavingGreaterThanOrEqual(column sqlf.Builder, value any) *SelectBuilder {
	return b.Having(
		sqlf.F("? >= ?", column, value),
	)
}

// HavingLessThanOrEqual adds a less-than-or-equal condition like `t.id <= 1`
func (b *SelectBuilder) HavingLessThanOrEqual(column sqlf.Builder, value any) *SelectBuilder {
	return b.Having(
		sqlf.F("? <= ?", column, value),
	)
}

// HavingIsNull adds a IS NULL condition like `t.deleted_at IS NULL`
func (b *SelectBuilder) HavingIsNull(column sqlf.Builder) *SelectBuilder {
	return b.Having(
		sqlf.F("? IS NULL", column),
	)
}

// HavingIsNotNull adds a IS NOT NULL condition like `t.deleted_at IS NOT NULL`
func (b *SelectBuilder) HavingIsNotNull(column sqlf.Builder) *SelectBuilder {
	return b.Having(
		sqlf.F("? IS NOT NULL", column),
	)
}

// HavingBetween adds a BETWEEN condition like `t.created_at BETWEEN ? AND ?`
func (b *SelectBuilder) HavingBetween(column sqlf.Builder, start, end any) *SelectBuilder {
	return b.Having(
		sqlf.F("? BETWEEN ? AND ?", column, start, end),
	)
}

// HavingNotBetween adds a NOT BETWEEN condition like `t.created_at NOT BETWEEN ? AND ?`
func (b *SelectBuilder) HavingNotBetween(column sqlf.Builder, start, end any) *SelectBuilder {
	return b.Having(
		sqlf.F("? NOT BETWEEN ? AND ?", column, start, end),
	)
}

// HavingIn adds a having IN condition like `t.id IN (1,2,3)`
func (b *SelectBuilder) HavingIn(column sqlf.Builder, list any) *SelectBuilder {
	return b.Having(
		sqlf.F(
			"? IN (?)",
			column,
			sqlf.JoinArgs(", ", util.FlattenArgs(list)...),
		),
	)
}

// HavingNotIn adds a having NOT IN condition like `t.id NOT IN (1,2,3)`
func (b *SelectBuilder) HavingNotIn(column sqlf.Builder, list any) *SelectBuilder {
	return b.Having(
		sqlf.F(
			"? NOT IN (?)",
			column,
			sqlf.JoinArgs(", ", util.FlattenArgs(list)...),
		),
	)
}
