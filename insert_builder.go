package sqlb

import (
	"github.com/qjebbs/go-sqlb/internal/clauses"
	"github.com/qjebbs/go-sqlb/internal/dialects"
	"github.com/qjebbs/go-sqlf/v4"
)

var _ sqlf.Builder = (*InsertBuilder)(nil)
var _ Builder = (*InsertBuilder)(nil)

// InsertBuilder is the SQL query builder.
// It's recommended to wrap it with your struct to provide a
// more friendly API and improve fragment reusability.
type InsertBuilder struct {
	dialact dialects.Dialect

	ctes       *clauses.With
	target     string         // target table for insertion
	columns    []string       // select columns and keep values in scanning.
	values     [][]any        // values for insert/update
	selects    sqlf.Builder   // select columns and keep values in scanning.
	conflictOn []string       // conflict target
	conflictDo []sqlf.Builder // conflict do action
	returning  []string       // returning columns

	errors []error // errors during building

	debug     bool // debug mode
	debugName string
}

// NewInsertBuilder returns a new InsertBuilder.
func NewInsertBuilder(dialect ...dialects.Dialect) *InsertBuilder {
	d := dialects.DialectPostgreSQL
	if len(dialect) > 0 {
		d = dialect[0]
	}
	return &InsertBuilder{
		dialact: d,
		ctes:    clauses.NewWith(),
	}
}

// InsertInto sets the target table for insertion.
func (b *InsertBuilder) InsertInto(t string) *InsertBuilder {
	b.target = t
	return b
}

// Columns sets the columns for insertion.
func (b *InsertBuilder) Columns(cols ...string) *InsertBuilder {
	b.columns = cols
	return b
}

// Values adds a row of values for insertion.
func (b *InsertBuilder) Values(vals ...any) *InsertBuilder {
	b.values = append(b.values, vals)
	return b
}

// From sets the SELECT builder for insertion.
//
// Example:
//
//	q := sqlb.NewSelectBuilder().Select(foo.Columns("bar", "baz")).From(foo)
//	b.From(q)
func (b *InsertBuilder) From(s sqlf.Builder) *InsertBuilder {
	b.selects = s
	return b
}

// Returning sets a RETURNING clause to the insert statement.
func (b *InsertBuilder) Returning(columns ...string) *InsertBuilder {
	b.returning = append(b.returning, columns...)
	return b
}

// With adds a CTE to the insert statement.
//
// Example:
//
//	q := sqlb.NewSelectBuilder().Select(foo.Column("*")).From(foo)
//	b.With(table, q)
func (b *InsertBuilder) With(name Table, builder sqlf.Builder) *InsertBuilder {
	b.ctes.With(name, builder)
	return b
}

// OnConflict sets the conflict target for the insert statement.
// The parameter actions are the actions to be taken on conflict,
// which is built after "DO UPDATE SET", e.g., sqlf.F("col = EXCLUDED.col").
// If no actions are provided, it means "DO NOTHING".
//
// Example:
//
//	columns := []string{"a", "b"}
//	b.OnConflict(columns, sqlf.F("c = EXCLUDED.c")) // ON CONFLICT (a, b) DO UPDATE SET c = EXCLUDED.c
//	b.OnConflict(columns)                           // ON CONFLICT (a, b) DO NOTHING
func (b *InsertBuilder) OnConflict(columns []string, actions ...sqlf.Builder) *InsertBuilder {
	b.conflictOn = columns
	b.conflictDo = actions
	return b
}
