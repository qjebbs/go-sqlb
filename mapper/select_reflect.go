package mapper

import (
	"fmt"
	"reflect"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/dialect"
	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlb/tag"
	"github.com/qjebbs/go-sqlf/v4"
)

func _select[T any](ctx sqlb.Context, db QueryAble, b SelectBuilder, options ...Option) ([]T, error) {
	var zero T
	if m, ok := any(zero).(selectableModel[T]); ok {
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
		return prepareScanDestinations(dest, dests)
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
		if !opt.selectTags.Match(col.SelectOn) {
			continue
		}

		column, err := buildSelectColumn(d, &col.Info, col.Type, opt)
		if err != nil {
			return nil, nil, fmt.Errorf("field %s: %w", col.Name, err)
		}
		columns = append(columns, column)
		dests = append(dests, col)
	}
	return columns, dests, nil
}

func buildSelectColumn(d dialect.Dialect, tags *tag.Info, rtype reflect.Type, opt *Options) (sqlf.Builder, error) {
	var column sqlf.Builder
	// sel tag takes precedence over col tag
	if tags.Select != "" {
		var frag *sqlf.Fragment
		if len(tags.From) > 0 {
			frag = sqlf.F(tags.Select, util.Map(tags.From, func(t string) any {
				return sqlb.NewTable("", t)
			})...)
		} else {
			frag = sqlf.F(tags.Select, sqlb.NewTable(tags.Table))
		}
		frag.NoUsageCheck()
		column = frag
	} else {
		column = sqlf.F("?.?", sqlb.NewTable(tags.Table), sqlf.Identifier(tags.Column))
	}
	if opt.enableNullZero(tags.Table) &&
		dialect.CheckNullCoalesceable(rtype) {
		if c, err := d.NullCoalesce(column, rtype); err == nil {
			if c != nil {
				column = c
			}
		} else {
			return nil, fmt.Errorf("build column %s: %w", tags.Column, err)
		}
	}
	return column, nil
}
