package sqlb

import (
	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

var _ sqlf.Builder = (*InsertBuilder)(nil)
var _ Builder = (*InsertBuilder)(nil)

// InsertBuilder is the SQL query sqlb.
// It's recommended to wrap it with your struct to provide a
// more friendly API and improve fragment reusability.
type InsertBuilder struct {
	ctes       *clauseWith
	target     Table          // target table for insertion
	columns    []sqlf.Builder // select columns and keep values in scanning.
	values     [][]any        // values for insert/update
	selects    sqlf.Builder   // select columns and keep values in scanning.
	conflictOn []sqlf.Builder // conflict target
	conflictDo []sqlf.Builder // conflict do action
	returning  []sqlf.Builder // returning columns

	errors []error // errors during building

	pruning bool
	debugger
}

// NewInsertBuilder returns a new InsertBuilder.
func NewInsertBuilder() *InsertBuilder {
	return &InsertBuilder{
		ctes: newWith(),
	}
}

// InsertInto sets the target table for insertion.
func (b *InsertBuilder) InsertInto(t string) *InsertBuilder {
	b.target = NewTable(t)
	return b
}

// Columns sets the columns for insertion.
func (b *InsertBuilder) Columns(cols ...string) *InsertBuilder {
	b.columns = util.Map(cols, func(c string) sqlf.Builder {
		return sqlf.Identifier(c)
	})
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
	cols := util.Map(columns, func(c string) sqlf.Builder {
		return sqlf.Identifier(c)
	})
	b.returning = append(b.returning, cols...)
	return b
}

// With adds a builder as common table expression.
//
// The CTE will be automatically eliminated if all the conditions below are met:
//   - Pruning is enabled by `b.EnableElimination()` or parent builders
//   - The table is not referenced anywhere in the query
func (b *InsertBuilder) With(name Table, builder sqlf.Builder) *InsertBuilder {
	b.ctes.With(name, builder)
	return b
}

// WithValues adds a VALUES common table expression.
// Supported dialects: Postgres, SQLite.
func (b *InsertBuilder) WithValues(name Table, columns, types []string, values [][]any) *InsertBuilder {
	b.ctes.WithValues(name, columns, types, values)
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
	b.conflictOn = util.Map(columns, func(c string) sqlf.Builder {
		return sqlf.Identifier(c)
	})
	b.conflictDo = actions
	return b
}
