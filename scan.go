package sqlb

import (
	"database/sql"

	"github.com/qjebbs/go-sqlf/v4"
)

// QueryAble is the interface for query-able *sql.DB, *sql.Tx, etc.
type QueryAble interface {
	Exec(query string, args ...any) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// Query queries the built query and scans rows into a slice.
//
// The main difference in behavior from *sql.Rows.Scan() is that
// it will discard extra columns if there are not enough scan destinations.
//
// This is useful when work with *QueryBuilder who may add extra select
// columns (on SELECT DISTINCT + ORDER BY), and Query will ignore those
// columns instead of reporting short-scan-destination errors.
func Query[T any](db QueryAble, b Builder, style sqlf.BindStyle, fn func() (T, []any)) ([]T, error) {
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
		err = scanRow(rows, fields...)
		if err != nil {
			return nil, err
		}
		results = append(results, dest)
	}
	return results, nil
}

// ScanRow scans a single row to dest, unlike rows.Scan(), it drops the extra columns.
// It's useful when *sqlb.QueryBuilder.OrderBy() add extra column to the query.
func scanRow(rows *sql.Rows, dest ...any) error {
	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	nBlacholes := len(cols) - len(dest)
	bh := &blackhole{}
	for i := 0; i < nBlacholes; i++ {
		dest = append(dest, &bh)
	}
	return rows.Scan(dest...)
}

type blackhole struct{}

func (b *blackhole) Scan(_ any) error { return nil }
