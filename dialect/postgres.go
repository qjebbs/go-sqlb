package dialect

import (
	"reflect"
	"time"

	"github.com/qjebbs/go-sqlf/v4"
	"github.com/qjebbs/go-sqlf/v4/dialect"
)

var _ Dialect = PostgreSQL{}

// PostgreSQL is the PostgreSQL dialect.
type PostgreSQL struct {
	dialect.PostgreSQL
}

// Capabilities returns the capabilities of the PostgreSQL dialect.
func (PostgreSQL) Capabilities() Capabilities {
	return Capabilities{
		SupportsReturning:      true,
		SupportsOutputInserted: false,

		SupportsInsertDefault:         true,
		SupportsOnConflict:            true,
		SupportsOnConflictSetExcluded: true,
		SupportsOnDuplicateKeyUpdate:  false,

		SupportsUpdateFrom: true,
		SupportsUpdateJoin: false,
	}
}

// NullCoalesce provides a PostgreSQL specific implementation for COALESCE, especially for time.Time type.
func (PostgreSQL) NullCoalesce(column sqlf.Builder, goType reflect.Type) (sqlf.Builder, error) {
	if !CheckNullCoalesceable(goType) {
		return nil, nil
	}
	// Use AssignableTo to handle custom type aliases like `type MyTime time.Time`.
	timeType := reflect.TypeOf(time.Time{})
	if goType.AssignableTo(timeType) {
		// Use ISO 8601 format with UTC timezone ('Z') for better compatibility,
		// especially with 'timestamp with time zone' (timestamptz) columns.
		return sqlf.F("COALESCE(?, '0001-01-01 00:00:00Z'::timestamptz)", column), nil
	}
	// Fallback to the generic ANSI implementation for other types
	return AnsiSQL{}.NullCoalesce(column, goType)
}
