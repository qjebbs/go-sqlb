package sqlb

import (
	"github.com/qjebbs/go-sqlf/v4"
	"github.com/qjebbs/go-sqlf/v4/util"
)

// Where add a condition.  e.g.:
//
//	b.Where(sqlf.F(
//		"? = ?",
//		foo.Column("id"), 1,
//	))
func (b *UpdateBuilder) Where(s sqlf.Builder) *UpdateBuilder {
	b.AppendWhere(s)
	return b
}

// AppendWhere appends raw where conditions to the sqlb.
func (b *UpdateBuilder) AppendWhere(s sqlf.Builder) {
	if s == nil {
		return
	}
	b.resetDepTablesCache()
	b.where.Append(s)
}

// WhereEquals is a helper func similar to Where(), which adds a simple equality condition. e.g.:
func (b *UpdateBuilder) WhereEquals(column sqlf.Builder, value any) *UpdateBuilder {
	return b.Where(
		sqlf.F("? = ?", column, value),
	)
}

// WhereNotEquals is a helper func similar to Where(), which adds a simple not-equal condition. e.g.:
func (b *UpdateBuilder) WhereNotEquals(column sqlf.Builder, value any) *UpdateBuilder {
	return b.Where(
		sqlf.F("? <> ?", column, value),
	)
}

// WhereGreaterThan adds a greater-than condition like `t.id > 1`
func (b *UpdateBuilder) WhereGreaterThan(column sqlf.Builder, value any) *UpdateBuilder {
	return b.Where(
		sqlf.F("? > ?", column, value),
	)
}

// WhereLessThan adds a less-than condition like `t.id < 1`
func (b *UpdateBuilder) WhereLessThan(column sqlf.Builder, value any) *UpdateBuilder {
	return b.Where(
		sqlf.F("? < ?", column, value),
	)
}

// WhereGreaterThanOrEqual adds a greater-than-or-equal condition like `t.id >= 1`
func (b *UpdateBuilder) WhereGreaterThanOrEqual(column sqlf.Builder, value any) *UpdateBuilder {
	return b.Where(
		sqlf.F("? >= ?", column, value),
	)
}

// WhereLessThanOrEqual adds a less-than-or-equal condition like `t.id <= 1`
func (b *UpdateBuilder) WhereLessThanOrEqual(column sqlf.Builder, value any) *UpdateBuilder {
	return b.Where(
		sqlf.F("? <= ?", column, value),
	)
}

// WhereIsNull adds a IS NULL condition like `t.deleted_at IS NULL`
func (b *UpdateBuilder) WhereIsNull(column sqlf.Builder) *UpdateBuilder {
	return b.Where(
		sqlf.F("? IS NULL", column),
	)
}

// WhereIsNotNull adds a IS NOT NULL condition like `t.deleted_at IS NOT NULL`
func (b *UpdateBuilder) WhereIsNotNull(column sqlf.Builder) *UpdateBuilder {
	return b.Where(
		sqlf.F("? IS NOT NULL", column),
	)
}

// WhereBetween adds a BETWEEN condition like `t.created_at BETWEEN ? AND ?`
func (b *UpdateBuilder) WhereBetween(column sqlf.Builder, start, end any) *UpdateBuilder {
	return b.Where(
		sqlf.F("? BETWEEN ? AND ?", column, start, end),
	)
}

// WhereNotBetween adds a NOT BETWEEN condition like `t.created_at NOT BETWEEN ? AND ?`
func (b *UpdateBuilder) WhereNotBetween(column sqlf.Builder, start, end any) *UpdateBuilder {
	return b.Where(
		sqlf.F("? NOT BETWEEN ? AND ?", column, start, end),
	)
}

// WhereIn adds a where IN condition like `t.id IN (1,2,3)`
func (b *UpdateBuilder) WhereIn(column sqlf.Builder, list any) *UpdateBuilder {
	return b.Where(
		sqlf.F(
			"? IN (?)",
			column,
			sqlf.JoinArgs(", ", util.FlattenArgs(list)...),
		),
	)
}

// WhereNotIn adds a where NOT IN condition like `t.id NOT IN (1,2,3)`
func (b *UpdateBuilder) WhereNotIn(column sqlf.Builder, list any) *UpdateBuilder {
	return b.Where(
		sqlf.F(
			"? NOT IN (?)",
			column,
			sqlf.JoinArgs(", ", util.FlattenArgs(list)...),
		),
	)
}
