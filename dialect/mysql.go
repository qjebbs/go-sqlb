package dialect

import (
	"reflect"
	"time"

	"github.com/qjebbs/go-sqlf/v4"
	"github.com/qjebbs/go-sqlf/v4/dialect"
)

var _ Dialect = MySQL{}

// MySQL is the ANSI SQL dialect.
type MySQL struct {
	dialect.MySQL
}

// Capabilities returns the capabilities of the MySQL dialect.
func (MySQL) Capabilities() Capabilities {
	return Capabilities{
		SupportsReturning:      false,
		SupportsOutputInserted: false,

		SupportsInsertDefault:         true,
		SupportsOnConflict:            false,
		SupportsOnConflictSetExcluded: false,
		SupportsOnDuplicateKeyUpdate:  true,

		SupportsUpdateFrom: false,
		SupportsUpdateJoin: true,
	}
}

// NullCoalesce provides a MySQL specific implementation for COALESCE, especially for time.Time type.
func (d MySQL) NullCoalesce(column sqlf.Builder, goType reflect.Type) (sqlf.Builder, error) {
	if !CheckNullCoalesceable(goType) {
		return nil, nil
	}
	// Use AssignableTo to handle custom type aliases.
	timeType := reflect.TypeOf(time.Time{})
	if goType.AssignableTo(timeType) {
		// Use the standard SQL TIMESTAMP literal format.
		// MySQL correctly interprets this as a UTC timestamp, which is crucial for
		// TIMESTAMP columns and avoids ambiguity with DATETIME columns.
		// This is the most robust way to represent Go's zero time (UTC).
		return sqlf.F("COALESCE(?, TIMESTAMP '0001-01-01 00:00:00')", column), nil
	}
	if goType.Kind() == reflect.Bool {
		// MySQL does not have a native boolean type, it uses TINYINT(1) where 0 is false and 1 is true.
		return sqlf.F("COALESCE(?, 0)", column), nil
	}
	// Fallback to the generic ANSI implementation for other types
	return AnsiSQL{}.NullCoalesce(column, goType)
}
