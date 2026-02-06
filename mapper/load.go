package mapper

import (
	"database/sql"
	"errors"
	"reflect"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

// Load loads a struct T from the database.
//
// The struct tag syntax is: `key[:value][;key[:value]]...`, e.g. `sqlb:"pk;col:id;table:users;"`
//
// The supported struct tags are:
//   - table<:name>: [Inheritable] Declare base table for the current field and its sub-fields / subsequent sibling fields.
//   - col<:name>: The column in database associated with the field.
//   - pk: The column is primary key.
//   - unique: The column is unique.
//   - unique_group[:name[,name]...]: The column is one of the "Composite Unique" groups. If there is only one unique_group in the struct, the group name can be omitted.
//   - match: The column will be always included in WHERE clause even if it is zero value.
//
// To locate the loading row, it will use non-zero `pk`, `unique`, or `unique_group` fields in priority order.
func Load[T any](ctx sqlb.Context, db QueryAble, value T, options ...Option) (T, error) {
	r, err := load(ctx, db, value, options...)
	if err != nil {
		var zero T
		return zero, wrapErrWithDebugName("Load", zero, err)
	}
	return r, nil
}

func load[T any](ctx sqlb.Context, db QueryAble, value T, options ...Option) (T, error) {
	var zero T
	err := checkPtrStruct(value)
	if err != nil {
		return zero, err
	}
	opt := mergeOptions(options...)

	var debugger *debugger
	if opt.debug {
		debugger = newDebugger("Load", value, opt)
		defer debugger.print(ctx.BaseDialect())
	}
	query, args, dests, err := buildLoadQueryForStruct(ctx, value, opt)
	if err != nil {
		return zero, err
	}
	if debugger != nil {
		debugger.onBuilt(query, args)
	}
	if db == nil {
		return zero, ErrNilDB
	}
	agents := make([]*nullZeroAgent, 0)
	r, err := scan(ctx, db, query, args, debugger, func() (T, []any) {
		dest, fields, ag := prepareScanDestinations(value, dests, opt)
		agents = append(agents, ag...)
		return dest, fields
	})
	if err != nil {
		return zero, err
	}
	if len(r) == 0 {
		return zero, sql.ErrNoRows
	}
	if len(agents) > 0 {
		for _, agent := range agents {
			agent.Apply()
		}
		debugger.onPostScan(nil)
	}
	return value, nil
}

func buildLoadQueryForStruct[T any](ctx sqlb.Context, value T, opt *Options) (query string, args []any, dests []fieldInfo, err error) {
	if opt == nil {
		opt = newDefaultOptions()
	}
	info, err := getStructInfo(value)
	if err != nil {
		return "", nil, nil, err
	}
	loadInfo, err := buildLoadInfo(info, value)
	if err != nil {
		return "", nil, nil, err
	}
	b := sqlb.NewSelectBuilder().
		Select(util.Map(loadInfo.selects, func(c fieldData) sqlf.Builder {
			return c.Info.NewColumnBuilder()
		})...).
		From(sqlb.NewTable(loadInfo.table))

	loadInfo.EachWhere(func(cond sqlf.Builder) {
		b.Where(cond)
	})

	query, args, err = b.Build(ctx)
	if err != nil {
		return "", nil, nil, err
	}
	dests = util.Map(loadInfo.selects, func(i fieldData) fieldInfo {
		return i.Info
	})
	return query, args, dests, nil
}

type loadInfo struct {
	locatingInfo
	selects []fieldData
}

func buildLoadInfo[T any](f *structInfo, value T) (*loadInfo, error) {
	valueVal := reflect.ValueOf(value)
	if valueVal.Kind() == reflect.Ptr {
		valueVal = valueVal.Elem()
	}
	locatingInfo, err := buildLocatingInfo(f, valueVal)
	if err != nil {
		return nil, err
	}
	if len(locatingInfo.others) == 0 {
		return nil, errors.New("no columns to load")
	}
	selects := make([]fieldData, 0, len(locatingInfo.others))
	for _, fd := range locatingInfo.others {
		if fd.Info.SoftDelete {
			continue
		}
		selects = append(selects, fd)
	}
	return &loadInfo{
		locatingInfo: *locatingInfo,
		selects:      selects,
	}, nil
}
