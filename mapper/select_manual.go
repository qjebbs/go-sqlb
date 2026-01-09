package mapper

import (
	"context"
	"database/sql"

	"github.com/qjebbs/go-sqlb"
)

// SelectOneManual executes a query and scans the results using a provider function.
// The provider fn is called for each row to get the destination value and scan fields.
// Unlike SelectOne, it doesn't limit the query to 1 row automatically.
func SelectOneManual[T any](db QueryAble, b sqlb.Builder, fn func() (T, []any), options ...Option) (T, error) {
	r, err := selectManual("SelectOneManual", db, b, fn, options...)
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

// SelectManual executes a query and scans the results using a provider function.
// The provider fn is called for each row to get the destination value and scan fields.
func SelectManual[T any](db QueryAble, b sqlb.Builder, fn func() (T, []any), options ...Option) ([]T, error) {
	return selectManual("SelectManual", db, b, fn, options...)
}

func selectManual[T any](name string, db QueryAble, b sqlb.Builder, fn func() (T, []any), options ...Option) ([]T, error) {
	opt := mergeOptions(options...)
	var debugger *debugger
	if opt.debug {
		value, _ := fn()
		debugger = newDebugger(name, value, opt)
		defer debugger.print()
	}
	query, args, err := b.BuildQueryContext(context.Background(), opt.style)
	if err != nil {
		return nil, err
	}
	if debugger != nil {
		debugger.onBuilt(query, args)
	}
	if db == nil {
		return nil, ErrNilDB
	}
	r, err := scan(db, query, args, debugger, fn)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// scan scans query rows with scanner
func scan[T any](db QueryAble, query string, args []any, debugger *debugger, fn func() (T, []any)) ([]T, error) {
	rows, err := db.Query(query, args...)
	if debugger != nil {
		debugger.onExec(err)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []T
	for rows.Next() {
		dest, fields := fn()
		err = rows.Scan(fields...)
		if err != nil {
			return nil, err
		}
		results = append(results, dest)
	}
	if debugger != nil {
		debugger.onScan(len(results), err)
	}
	return results, rows.Err()
}

// // ScanRow scans a single row to dest, unlike rows.Scan(), it drops the extra columns.
// // It's useful when *sqlb.SelectBuilder.OrderBy() add extra column to the query.
// func scanRow(rows *sql.Rows, dest ...any) error {
// 	cols, err := rows.Columns()
// 	if err != nil {
// 		return err
// 	}
// 	nBlacholes := len(cols) - len(dest)
// 	bh := &blackhole{}
// 	for i := 0; i < nBlacholes; i++ {
// 		dest = append(dest, &bh)
// 	}
// 	return rows.Scan(dest...)
// }

// type blackhole struct{}

// func (b *blackhole) Scan(_ any) error { return nil }
