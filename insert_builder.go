package sqlb

import (
	"github.com/qjebbs/go-sqlf/v4"
)

var _ sqlf.Builder = (*InsertBuilder)(nil)
var _ Builder = (*InsertBuilder)(nil)

// InsertBuilder is the SQL query builder.
// It's recommended to wrap it with your struct to provide a
// more friendly API and improve fragment reusability.
type InsertBuilder struct {
	ctes       *_CTEs
	target     Table
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
func NewInsertBuilder() *InsertBuilder {
	return &InsertBuilder{
		ctes: newCTEs(),
	}
}

// InsertInto sets the target table for insertion.
func (b *InsertBuilder) InsertInto(t Table) *InsertBuilder {
	b.target = t
	return b
}

// Columns sets the columns for insertion.
func (b *InsertBuilder) Columns(cols ...string) *InsertBuilder {
	b.SetColumns(cols)
	return b
}

// SetColumns sets the columns for insertion,
// which implements the InsertBuilder interface.
func (b *InsertBuilder) SetColumns(cols []string) {
	b.columns = cols
}

// Values adds a row of values for insertion.
func (b *InsertBuilder) Values(vals ...any) *InsertBuilder {
	b.values = append(b.values, vals)
	return b
}

// SetValues sets multiple rows of values for insertion,
// which implements the InsertBuilder interface.
func (b *InsertBuilder) SetValues(rows [][]any) {
	b.values = rows
}

// From sets the SELECT builder for insertion.
func (b *InsertBuilder) From(s sqlf.Builder) *InsertBuilder {
	b.selects = s
	return b
}

// Returning sets a RETURNING clause to the insert statement.
func (b *InsertBuilder) Returning(columns ...string) *InsertBuilder {
	b.returning = append(b.returning, columns...)
	return b
}

// SetReturning sets returning columns
func (b *InsertBuilder) SetReturning(columns []string) {
	b.returning = columns
}

// With adds a CTE to the insert statement.
func (b *InsertBuilder) With(name Table, builder sqlf.Builder) *InsertBuilder {
	b.ctes.With(name, builder)
	return b
}

// OnConflict sets the conflict target for the insert statement.
func (b *InsertBuilder) OnConflict(columns ...string) *InsertBuilder {
	b.conflictOn = columns
	return b
}

// DoUpdateSet sets the conflict action for the insert statement.
func (b *InsertBuilder) DoUpdateSet(actions ...sqlf.Builder) *InsertBuilder {
	b.conflictDo = append(b.conflictDo, actions...)
	return b
}

// SetConflictDo sets the conflict action for the insert statement.
func (b *InsertBuilder) SetConflictDo(columns []string, actions []sqlf.Builder) {
	b.conflictOn = columns
	b.conflictDo = actions
}
