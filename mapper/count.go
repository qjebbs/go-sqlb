package mapper

import (
	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlf/v4"
)

// Count builds and executes the query to count records into an int64.
func Count(ctx sqlb.Context, db QueryAble, b SelectBuilder, options ...Option) (int64, error) {
	r, err := _count(ctx, db, b, nil, options...)
	if err != nil {
		return 0, wrapErrWithDebugName("Count", b, err)
	}
	return r, nil
}

// CountColumn builds and executes the query to count records based on the specified column into an int64.
func CountColumn(ctx sqlb.Context, db QueryAble, b SelectBuilder, column sqlf.Builder, options ...Option) (int64, error) {
	r, err := _count(ctx, db, b, column, options...)
	if err != nil {
		return 0, wrapErrWithDebugName("CountColumn", b, err)
	}
	return r, nil
}

func _count(ctx sqlb.Context, db QueryAble, b SelectBuilder, column sqlf.Builder, options ...Option) (int64, error) {
	opt := mergeOptions(options...)
	var debugger *debugger
	if opt.debug {
		debugger = newDebugger("Count", b, opt)
		defer debugger.print(ctx.BaseDialect())
	}
	if column == nil {
		b.SetSelect(sqlf.F("COUNT(1)"))
	} else {
		b.SetSelect(sqlf.F("COUNT(?)", column))
	}
	query, args, err := sqlf.Build(ctx, b)
	if err != nil {
		return 0, err
	}
	if debugger != nil {
		debugger.onBuilt(query, args)
	}
	if db == nil {
		return 0, ErrNilDB
	}
	var r int64
	err = db.QueryRow(query, args...).Scan(&r)
	if debugger != nil {
		debugger.onExec(err)
	}
	if err != nil {
		return 0, err
	}
	return r, nil
}
