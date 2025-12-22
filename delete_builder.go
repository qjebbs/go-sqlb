package sqlb

import (
	"github.com/qjebbs/go-sqlb/internal/clauses"
	"github.com/qjebbs/go-sqlf/v4"
	"github.com/qjebbs/go-sqlf/v4/util"
)

var _ sqlf.Builder = (*DeleteBuilder)(nil)
var _ Builder = (*DeleteBuilder)(nil)

// DeleteBuilder is the SQL query builder.
// It's recommended to wrap it with your struct to provide a
// more friendly API and improve fragment reusability.
type DeleteBuilder struct {
	target string
	where  *clauses.PrefixedList // where conditions, joined with AND.

	debug     bool // debug mode
	debugName string
}

// NewDeleteBuilder returns a new DeleteBuilder.
func NewDeleteBuilder() *DeleteBuilder {
	return &DeleteBuilder{
		where: clauses.NewPrefixedList("WHERE", " AND "),
	}
}

// DeleteFrom set the Delete target table.
func (b *DeleteBuilder) DeleteFrom(table string) *DeleteBuilder {
	b.target = table
	return b
}

// Where add a condition.  e.g.:
//
//	b.Where(sqlf.F(
//		"? = ?",
//		foo.Column("id"), 1,
//	))
func (b *DeleteBuilder) Where(s sqlf.Builder) *DeleteBuilder {
	if s == nil {
		return b
	}
	b.where.Append(s)
	return b
}

// WhereEquals is a helper func similar to Where(), which adds a simple equality condition. e.g.:
func (b *DeleteBuilder) WhereEquals(column sqlf.Builder, value any) *DeleteBuilder {
	return b.Where(
		sqlf.F("? = ?", column, value),
	)
}

// WhereNotEquals is a helper func similar to Where(), which adds a simple not-equal condition. e.g.:
func (b *DeleteBuilder) WhereNotEquals(column sqlf.Builder, value any) *DeleteBuilder {
	return b.Where(
		sqlf.F("? <> ?", column, value),
	)
}

// WhereGreaterThan adds a greater-than condition like `t.id > 1`
func (b *DeleteBuilder) WhereGreaterThan(column sqlf.Builder, value any) *DeleteBuilder {
	return b.Where(
		sqlf.F("? > ?", column, value),
	)
}

// WhereLessThan adds a less-than condition like `t.id < 1`
func (b *DeleteBuilder) WhereLessThan(column sqlf.Builder, value any) *DeleteBuilder {
	return b.Where(
		sqlf.F("? < ?", column, value),
	)
}

// WhereGreaterThanOrEqual adds a greater-than-or-equal condition like `t.id >= 1`
func (b *DeleteBuilder) WhereGreaterThanOrEqual(column sqlf.Builder, value any) *DeleteBuilder {
	return b.Where(
		sqlf.F("? >= ?", column, value),
	)
}

// WhereLessThanOrEqual adds a less-than-or-equal condition like `t.id <= 1`
func (b *DeleteBuilder) WhereLessThanOrEqual(column sqlf.Builder, value any) *DeleteBuilder {
	return b.Where(
		sqlf.F("? <= ?", column, value),
	)
}

// WhereIsNull adds a IS NULL condition like `t.deleted_at IS NULL`
func (b *DeleteBuilder) WhereIsNull(column sqlf.Builder) *DeleteBuilder {
	return b.Where(
		sqlf.F("? IS NULL", column),
	)
}

// WhereIsNotNull adds a IS NOT NULL condition like `t.deleted_at IS NOT NULL`
func (b *DeleteBuilder) WhereIsNotNull(column sqlf.Builder) *DeleteBuilder {
	return b.Where(
		sqlf.F("? IS NOT NULL", column),
	)
}

// WhereBetween adds a BETWEEN condition like `t.created_at BETWEEN ? AND ?`
func (b *DeleteBuilder) WhereBetween(column sqlf.Builder, start, end any) *DeleteBuilder {
	return b.Where(
		sqlf.F("? BETWEEN ? AND ?", column, start, end),
	)
}

// WhereNotBetween adds a NOT BETWEEN condition like `t.created_at NOT BETWEEN ? AND ?`
func (b *DeleteBuilder) WhereNotBetween(column sqlf.Builder, start, end any) *DeleteBuilder {
	return b.Where(
		sqlf.F("? NOT BETWEEN ? AND ?", column, start, end),
	)
}

// WhereIn adds a where IN condition like `t.id IN (1,2,3)`
func (b *DeleteBuilder) WhereIn(column sqlf.Builder, list any) *DeleteBuilder {
	return b.Where(
		sqlf.F(
			"? IN (?)",
			column,
			sqlf.JoinArgs(", ", util.FlattenArgs(list)...),
		),
	)
}

// WhereNotIn adds a where NOT IN condition like `t.id NOT IN (1,2,3)`
func (b *DeleteBuilder) WhereNotIn(column sqlf.Builder, list any) *DeleteBuilder {
	return b.Where(
		sqlf.F(
			"? NOT IN (?)",
			column,
			sqlf.JoinArgs(", ", util.FlattenArgs(list)...),
		),
	)
}
