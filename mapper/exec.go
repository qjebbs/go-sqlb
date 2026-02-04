package mapper

import (
	"database/sql"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlf/v4"
)

// Exec Executes a sqlf.Builder query against the database.
func Exec(ctx sqlb.Context, db QueryAble, b sqlf.Builder, options ...Option) (sql.Result, error) {
	opt := mergeOptions(options...)
	var debugger *debugger
	if opt.debug {
		debugger = newDebugger("Exec", b, opt)
		defer debugger.print(ctx.BaseDialect())
	}
	query, args, err := sqlf.Build(ctx, b)
	if err != nil {
		return nil, err
	}
	if debugger != nil {
		debugger.onBuilt(query, args)
	}
	r, err := db.Exec(query, args...)
	if debugger != nil {
		debugger.onExec(err)
	}
	return r, err
}
