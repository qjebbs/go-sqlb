package mapper

import (
	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlf/v4"
)

// Exists checks whether a record exists in the database.
//
// See Load() for struct tag syntax and locating rules.
func Exists[T any](ctx sqlb.Context, db QueryAble, value T, options ...Option) (bool, error) {
	if err := checkStruct(value); err != nil {
		return false, err
	}
	opt := mergeOptions(options...)

	var debugger *debugger
	if opt.debug {
		debugger = newDebugger("Exists", value, opt)
		defer debugger.print(ctx.BaseDialect())
	}
	queryStr, args, err := buildExistsQueryForStruct(ctx, value, opt)
	if err != nil {
		return false, err
	}
	if debugger != nil {
		debugger.onBuilt(queryStr, args)
	}
	if db == nil {
		return false, ErrNilDB
	}
	var existsInt int
	err = db.QueryRow(queryStr, args...).Scan(&existsInt)
	if debugger != nil {
		debugger.onExec(err)
	}
	if err != nil {
		return false, err
	}
	return existsInt > 0, nil
}

func buildExistsQueryForStruct[T any](ctx sqlb.Context, value T, opt *Options) (query string, args []any, err error) {
	if opt == nil {
		opt = newDefaultOptions()
	}
	info, err := getStructInfo(value)
	if err != nil {
		return "", nil, err
	}
	loadInfo, err := buildLoadInfo(info, value)
	if err != nil {
		return "", nil, err
	}
	b := sqlb.NewSelectBuilder().
		Select(sqlf.F("1")).
		From(sqlb.NewTable(loadInfo.table))

	loadInfo.EachWhere(func(cond sqlf.Builder) {
		b.Where(cond)
	})
	query, args, err = b.Build(ctx)
	if err != nil {
		return "", nil, err
	}
	return query, args, nil
}
