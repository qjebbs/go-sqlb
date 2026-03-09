package mapper

import (
	"database/sql"
	"fmt"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlf/v4"
)

var _ SelectBuilder = (*sqlb.SelectBuilder)(nil)
var _ SelectLimitBuilder = (*sqlb.SelectBuilder)(nil)

// SelectBuilder is the interface for builders that support Select method.
type SelectBuilder interface {
	sqlf.Builder
	SetSelect(columns ...sqlf.Builder)
}

// SelectLimitBuilder is the interface for builders that support Limit method.
type SelectLimitBuilder interface {
	SelectBuilder
	SetLimit(n int64)
}

// SelectOne executes the query and scans the result into a struct T.
//
// See Select() for supported struct tags.
func SelectOne[T any](ctx sqlb.Context, db QueryAble, b SelectLimitBuilder, options ...Option) (T, error) {
	b.SetLimit(1)
	r, err := Select[T](ctx, db, b, options...)
	if err != nil {
		var zero T
		return zero, err
	}
	if len(r) == 0 {
		var zero T
		return zero, sql.ErrNoRows
	}
	return r[0], nil
}

// Select builds and executes the query and scans the results into a slice of struct T.
//
// The struct tag syntax is: `key[:value][;key[:value]]...`, e.g. `sqlb:"table:users;col:id"`
//
// The supported struct tags are:
//   - table<:name>: [Inheritable] Declare the database table for the current field and its sub-fields / subsequent sibling fields, e.g. `table:foo;`
//   - col<:name>: the column to select for this field, e.g. `col:id;`
//   - sel_on<:tag,[,tags]...>: Scan the field only on any one of tags specified, comma-separated. e.g. `sel_on:full;`
//   - sel<:expr>: Specify expression to select for this field. It's used together with `from` key to declare tables used in the expression, e.g. “sel:COALESCE(?.bar,?.baz);from:f,b;`, which is required by dependency analysis.
//   - from<:name[,names]...>: Works with 'sel', it accepts multiple Applied-Table-Name, comma-separated.
//   - dive: For struct fields, dive into and scan its fields. e.g. `dive;`
func Select[T any](ctx sqlb.Context, db QueryAble, b SelectBuilder, options ...Option) ([]T, error) {
	r, err := _select[T](ctx, db, b, options...)
	if err != nil {
		var zero T
		return nil, wrapErrWithDebugName("Select", zero, err)
	}
	return r, nil
}

func _select[T any](ctx sqlb.Context, db QueryAble, b SelectBuilder, options ...Option) ([]T, error) {
	var zero T
	m, ok := any(zero).(selectableModel[T])
	if !ok {
		return _selectReflect(ctx, db, zero, b, options...)
	}
	// m can be nil if T is an pointer type, so we need to create a new instance to call the methods.
	// It's safe to call New() on a nil pointer receiver, as long as the method doesn't access any
	// fields of the struct.
	model := any(m.New()).(selectableModel[T])
	r, err := _selectModel(ctx, db, b, model, options...)
	if err != nil {
		return nil, fmt.Errorf("select model: %w", err)
	}
	return r, nil

}
