package dialect

import (
	"reflect"
	"time"

	"github.com/qjebbs/go-sqlf/v4"
	"github.com/qjebbs/go-sqlf/v4/dialect"
)

var _ Dialect = SQLite{}

// SQLite is the SQLite dialect.
type SQLite struct {
	dialect.SQLite
}

// Capabilities returns the capabilities of the SQLite dialect.
func (SQLite) Capabilities() Capabilities {
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

// NullCoalesce provides a SQLite specific implementation for COALESCE, especially for time.Time type.
func (d SQLite) NullCoalesce(column sqlf.Builder, goType reflect.Type) (sqlf.Builder, error) {
	if !CheckNullCoalesceable(goType) {
		return nil, nil
	}
	// Use AssignableTo to handle custom type aliases.
	timeType := reflect.TypeOf(time.Time{})
	if goType.AssignableTo(timeType) {
		// SQLite uses a string representation for datetime.
		return sqlf.F("COALESCE(?, '0001-01-01 00:00:00Z')", column), nil
	}
	// Fallback to the generic ANSI implementation for other types
	return AnsiSQL{}.NullCoalesce(column, goType)
}
