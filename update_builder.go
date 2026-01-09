package sqlb

import (
	"github.com/qjebbs/go-sqlf/v4"
)

var _ sqlf.Builder = (*UpdateBuilder)(nil)
var _ Builder = (*UpdateBuilder)(nil)

// UpdateBuilder is the SQL query sqlb.
// It's recommended to wrap it with your struct to provide a
// more friendly API and improve fragment reusability.
type UpdateBuilder struct {
	dialact Dialect
	ctes    *clauseWith
	from    *clauseFrom

	target string
	sets   *clauseList // select columns and keep values in scanning.
	where  *clauseList // where conditions, joined with AND.
	order  *clauseList // order by columns, joined with comma.
	limit  int64       // limit count

	debugger

	pruning bool
	deps    *selectBuilderDependencies
	errors  []error
}

// NewUpdateBuilder returns a new UpdateBuilder.
func NewUpdateBuilder(dialect ...Dialect) *UpdateBuilder {
	d := DialectPostgres
	if len(dialect) > 0 {
		d = dialect[0]
	}
	return &UpdateBuilder{
		dialact: d,
		ctes:    newWith(),
		from:    newFrom(),
		sets:    newPrefixedList("SET", ", "),
		where:   newPrefixedList("WHERE", " AND "),
		order:   newPrefixedList("ORDER BY", ", "),
	}
}

// Update set the update target table.
func (b *UpdateBuilder) Update(table string) *UpdateBuilder {
	if b.dialact == DialectMySQL {
		b.from.ImplicitedFrom(NewTable(table))
	}
	b.resetDepTablesCache()
	b.target = table
	return b
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

// OrderBy set the sorting order.
//
//	foo := sqlb.NewTable("foo")
//	b.OrderBy(sqlf.F("? DESC", foo.Column("bar")))
func (b *UpdateBuilder) OrderBy(order ...sqlf.Builder) *UpdateBuilder {
	b.resetDepTablesCache()
	b.order.Append(order...)
	return b
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
