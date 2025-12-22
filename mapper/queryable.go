package mapper

import (
	"database/sql"
)

// QueryAble is the interface for query-able *sql.DB, *sql.Tx, etc.
type QueryAble interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	// Prepare(query string) (*sql.Stmt, error)
}
