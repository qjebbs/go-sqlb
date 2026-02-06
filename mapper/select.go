package mapper

import (
	"database/sql"
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
// The struct tag syntax is: `key[:value][;key[:value]]...`, e.g. `sqlb:"col:id;from:u;"`
//
// The supported struct tags are:
//   - sel<:expr>: Specify expression to select for this field. It's used together with `from` key to declare tables used in the expression, e.g. `sel:COALESCE(?.name,”);from:u;`, which is required by dependency analysis.
//   - from<:name[,names]...>: [Inheritable] Declare from tables for this field or its sub-fields / subsequent sibling fields. It accepts multiple Applied-Table-Name, comma-separated, e.g. `from:f,b`.
//   - sel_on<:tag,[,tags]...>: Scan the field only on any one of tags specified, comma-separated. e.g. `sel_on:full;`
//   - col<:name>: If `sel` key is not specified, specify the column to select for this field. It's recommended to use `col` key for simple column selection, which can be shared usage in INSERT/UPDATE/DELETE operations. e.g. `col:name;from:u;`
//   - dive: For struct fields, dive into scan its field. e.g. `dive;`
//   - table<:name>: [Inheritable] Declare base table for the current field and its sub-fields / subsequent sibling fields. It usually works with `WithNullZeroTables()` Option.
//
// Applied-Table-Name: The name of the table that is effective in the current query. For example, `f` in `sqlb.NewTable("foo", "f")`, and `foo` in `sqlb.NewTable("foo")`.
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
	agents := make([]*nullZeroAgent, 0)
	r, err := scan(ctx, db, queryStr, args, debugger, func() (T, []any) {
		var dest T
		dest, fields, ag := prepareScanDestinations(dest, dests, opt)
		agents = append(agents, ag...)
		return dest, fields
	})
	if err != nil {
		return nil, err
	}
	if len(agents) > 0 {
		for _, agent := range agents {
			agent.Apply()
		}
		if debugger != nil {
			debugger.onPostScan(nil)
		}
	}
	return r, nil
}

// prepareScanDestinations prepares the destinations for scanning the query results into the struct fields.
// !!! MUST return dest since the param 'dest' and the caller 'dest' is different variable.
// prepareScanDestinations will create new instances for nil pointer fields as needed which
// affects only the param 'dest'.
func prepareScanDestinations[T any](dest T, dests []fieldInfo, opt *Options) (T, []any, []*nullZeroAgent) {
	destValue := reflect.ValueOf(&dest).Elem()
	if destValue.Kind() == reflect.Ptr {
		if destValue.IsNil() {
			destValue.Set(reflect.New(destValue.Type().Elem()))
		}
		destValue = destValue.Elem()
	}
	fields := make([]any, len(dests))
	agents := make([]*nullZeroAgent, 0, len(dests))
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
		target := field.Addr().Interface()
		if field.Kind() == reflect.Ptr || !opt.enableNullZero(dest.Table) {
			fields[i] = target
			continue
		}
		agent := newNullZeroAgent(field)
		fields[i] = agent.Agent()
		agents = append(agents, agent)
	}
	return dest, fields, agents
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
	columns, dests := buildSelectInfo(ctx.Dialect(), opt, info)
	b.SetSelect(columns...)
	query, args, err = sqlf.Build(ctx, b)
	if err != nil {
		return "", nil, nil, err
	}
	return query, args, dests, nil
}

func buildSelectInfo(dialect dialect.Dialect, opt *Options, f *structInfo) (columns []sqlf.Builder, dests []fieldInfo) {
	for _, col := range f.columns {
		included := opt.matchTag(col.SelectOn)
		if !included {
			continue
		}
		// sel tag takes precedence over col tag
		checkUsage := !col.InheritedFroms
		expr := col.Select
		if expr == "" && col.Column != "" {
			checkUsage = false
			expr = "?." + dialect.QuoteIdentifier(col.Column)
		}
		column := sqlf.F(expr, util.Map(col.From, func(t string) any {
			return sqlb.NewTable("", t)
		})...)
		if !checkUsage {
			column.NoUsageCheck()
		}
		columns = append(columns, column)
		dests = append(dests, col)
	}
	return
}
