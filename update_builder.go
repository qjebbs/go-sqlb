package sqlb

import (
	"github.com/qjebbs/go-sqlb/internal/clauses"
	"github.com/qjebbs/go-sqlb/internal/dialects"
	"github.com/qjebbs/go-sqlf/v4"
)

var _ sqlf.Builder = (*UpdateBuilder)(nil)
var _ Builder = (*UpdateBuilder)(nil)

// UpdateBuilder is the SQL query builder.
// It's recommended to wrap it with your struct to provide a
// more friendly API and improve fragment reusability.
type UpdateBuilder struct {
	dialact dialects.Dialect
	ctes    *clauses.With
	from    *clauses.From

	target Table
	sets   *clauses.PrefixedList // select columns and keep values in scanning.
	where  *clauses.PrefixedList // where conditions, joined with AND.
	order  *clauses.OrderBy      // order by columns, joined with comma.
	limit  int64                 // limit count

	debug     bool // debug mode
	debugName string

	deps   *selectBuilderDependencies
	errors []error
}

// NewUpdateBuilder returns a new UpdateBuilder.
func NewUpdateBuilder(dialect ...dialects.Dialect) *UpdateBuilder {
	d := dialects.DialectPostgreSQL
	if len(dialect) > 0 {
		d = dialect[0]
	}
	return &UpdateBuilder{
		dialact: d,
		ctes:    clauses.NewWith(),
		from:    clauses.NewFrom(),
		sets:    clauses.NewPrefixedList("SET", ", "),
		where:   clauses.NewPrefixedList("WHERE", " AND "),
		order:   clauses.NewOrderBy(),
	}
}

// Update set the update target table.
func (b *UpdateBuilder) Update(table Table) *UpdateBuilder {
	b.SetUpdate(table)
	if b.dialact == dialects.DialectMySQL {
		b.from.ImplicitedFrom(table)
	}
	return b
}

// SetUpdate set the update target table,
// which implements the UpdateBuilder interface.
func (b *UpdateBuilder) SetUpdate(table Table) {
	b.resetDepTablesCache()
	b.target = table
}

// Set set the update sets.
// Multiple calls will append the sets, for example:
//
//	b.Set("name", "Alice").Set("age", 30)
//
// The value can also be a sqlf.Builder, for example:
//
//	Users := NewTable("users", u)
//	b.Set("age", Users.Column("age")).From(Users).Where(...)
func (b *UpdateBuilder) Set(column string, value any) *UpdateBuilder {
	b.resetDepTablesCache()
	b.sets.Append(sqlf.F("? = ?", sqlf.F(column), value))
	return b
}

// SetSets set and replace the update sets,
// which implements the UpdateBuilder interface.
func (b *UpdateBuilder) SetSets(sets ...sqlf.Builder) {
	b.resetDepTablesCache()
	b.sets.Replace(sets)
}

// Limit set the limit.
func (b *UpdateBuilder) Limit(limit int64) *UpdateBuilder {
	if limit > 0 {
		b.limit = limit
	}
	return b
}

func (b *UpdateBuilder) resetDepTablesCache() {
	b.deps = nil
}
