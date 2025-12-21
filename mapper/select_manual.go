package mapper

import (
	"database/sql"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlf/v4"
)

// QueryAble is the interface for query-able *sql.DB, *sql.Tx, etc.
type QueryAble interface {
	Exec(query string, args ...any) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// SelectOneManual executes a query and scans the results using a provider function.
// The provider fn is called for each row to get the destination value and scan fields.
// Unlike SelectOne, it doesn't limit the query to 1 row automatically.
func SelectOneManual[T any](db QueryAble, b sqlb.Builder, style sqlf.BindStyle, fn func() (T, []any)) (T, error) {
	r, err := SelectManual(db, b, style, fn)
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
func SelectManual[T any](db QueryAble, b sqlb.Builder, style sqlf.BindStyle, fn func() (T, []any)) ([]T, error) {
	query, args, err := b.BuildQuery(style)
	if err != nil {
		return nil, err
	}
	return scan(db, query, args, fn)
}

// scan scans query rows with scanner
func scan[T any](db QueryAble, query string, args []any, fn func() (T, []any)) ([]T, error) {
	rows, err := db.Query(query, args...)
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
	return results, nil
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
