package mapper

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/dialect"
	"github.com/qjebbs/go-sqlb/internal/util"
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
	if err := checkPtrStruct(zero); err != nil {
		return nil, err
	}
	opt := mergeOptions(options...)
	var debugger *debugger
	if opt.debug {
		debugger = newDebugger("Select", zero, opt)
		defer debugger.print(ctx.BaseDialect())
	}
	queryStr, args, dests, err := buildSelectQueryForStruct[T](ctx, b, opt)
	if err != nil {
		return nil, err
	}
	if debugger != nil {
		debugger.onBuilt(queryStr, args)
	}
	if db == nil {
		return nil, ErrNilDB
	}
	return scan(ctx, db, queryStr, args, debugger, func() (T, []any) {
		var dest T
		dest, fields := prepareScanDestinations(dest, dests)
		return dest, fields
	})
}

// prepareScanDestinations prepares the destinations for scanning the query results into the struct fields.
// !!! MUST return dest since the param 'dest' and the caller 'dest' is different variable.
// prepareScanDestinations will create new instances for nil pointer fields as needed which
// affects only the param 'dest'.
func prepareScanDestinations[T any](dest T, dests []fieldInfo) (T, []any) {
	destValue := reflect.ValueOf(&dest).Elem()
	if destValue.Kind() == reflect.Ptr {
		if destValue.IsNil() {
			destValue.Set(reflect.New(destValue.Type().Elem()))
		}
		destValue = destValue.Elem()
	}
	fields := make([]any, len(dests))
	for i, dest := range dests {
		current := destValue
		// Traverse the field path and initialize nil pointers.
		for _, fieldIndex := range dest.Index[:len(dest.Index)-1] {
			current = current.Field(fieldIndex)
			if current.Kind() == reflect.Ptr {
				if current.IsNil() {
					current.Set(reflect.New(current.Type().Elem()))
				}
				current = current.Elem()
			}
		}
		field := current.Field(dest.Index[len(dest.Index)-1])
		fields[i] = field.Addr().Interface()
	}
	return dest, fields
}

func buildSelectQueryForStruct[T any](ctx sqlb.Context, b SelectBuilder, opt *Options) (query string, args []any, dests []fieldInfo, err error) {
	if opt == nil {
		opt = newDefaultOptions()
	}
	var zero T
	info, err := getStructInfo(zero)
	if err != nil {
		return "", nil, nil, err
	}
	columns, dests, err := buildSelectInfo(ctx.Dialect(), opt, info)
	if err != nil {
		return "", nil, nil, err
	}
	b.SetSelect(columns...)
	query, args, err = sqlf.Build(ctx, b)
	if err != nil {
		return "", nil, nil, err
	}
	return query, args, dests, nil
}

func buildSelectInfo(d dialect.Dialect, opt *Options, f *structInfo) (columns []sqlf.Builder, dests []fieldInfo, err error) {
	for _, col := range f.columns {
		if col.Column == "" && col.Select == "" {
			continue
		}
		if !opt.matchTag(col.SelectOn) {
			continue
		}

		var column sqlf.Builder
		// sel tag takes precedence over col tag
		if col.Select != "" {
			var frag *sqlf.Fragment
			if len(col.From) > 0 {
				frag = sqlf.F(col.Select, util.Map(col.From, func(t string) any {
					return sqlb.NewTable("", t)
				})...)
			} else {
				frag = sqlf.F(col.Select, sqlb.NewTable(col.Table))
			}
			frag.NoUsageCheck()
			column = frag
		} else {
			column = sqlf.F("?.?", sqlb.NewTable(col.Table), sqlf.Identifier(col.Column))
		}
		if opt.enableNullZero(col.Table) &&
			dialect.CheckNullCoalesceable(col.Type) {
			if c, err := d.NullCoalesce(column, col.Type); err == nil {
				if c != nil {
					column = c
				}
			} else {
				return nil, nil, fmt.Errorf("field %s: %w", col.Name, err)
			}
		}
		columns = append(columns, column)
		dests = append(dests, col)
	}
	return columns, dests, nil
}
