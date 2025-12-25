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
func Load[T any](db QueryAble, value T, options ...Option) (T, error) {
	r, err := load(db, value, options...)
	if err != nil {
		var zero T
		return zero, wrapErrWithDebugName("Load", zero, err)
	}
	return r, nil
}

func load[T any](db QueryAble, value T, options ...Option) (T, error) {
	var zero T
	if err := checkPtrStruct(value); err != nil {
		return zero, err
	}
	opt := mergeOptions(options...)
	queryStr, args, dests, err := buildLoadQueryForStruct(value, opt)
	if err != nil {
		return zero, err
	}
	if db == nil {
		return zero, ErrNilDB
	}
	agents := make([]*nullZeroAgent, 0)
	r, err := scan(db, queryStr, args, func() (T, []any) {
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
	for _, agent := range agents {
		agent.Apply()
	}
	return value, nil
}

func buildLoadQueryForStruct[T any](value T, opt *Options) (query string, args []any, dests []fieldInfo, err error) {
	if opt == nil {
		opt = newDefaultOptions()
	}
	info, err := getStructInfo(value)
	if err != nil {
		return "", nil, nil, err
	}
	loadInfo, err := buildLoadInfo(opt.dialect, info, value)
	if err != nil {
		return "", nil, nil, err
	}
	b := sqlb.NewSelectBuilder().
		Select(util.Map(loadInfo.selects, func(c fieldData) sqlf.Builder {
			return sqlf.F(c.Indent)
		})...).
		From(sqlb.NewTable(loadInfo.table))

	loadInfo.EachWhere(func(cond sqlf.Builder) {
		b.Where(cond)
	})

	query, args, err = b.BuildQuery(opt.style)
	if err != nil {
		return "", nil, nil, err
	}
	if opt.debug {
		printDebugQuery("Load", value, query, args)
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

func buildLoadInfo[T any](dialect sqlb.Dialect, f *structInfo, value T) (*loadInfo, error) {
	valueVal := reflect.ValueOf(value)
	if valueVal.Kind() == reflect.Ptr {
		valueVal = valueVal.Elem()
	}
	locatingInfo, err := buildLocatingInfo(dialect, f, valueVal)
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
