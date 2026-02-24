package dialect

import (
	"reflect"
	"time"

	"github.com/qjebbs/go-sqlf/v4"
	"github.com/qjebbs/go-sqlf/v4/dialect"
)

var _ Dialect = SQLServer{}

// SQLServer is the SQLServer dialect.
type SQLServer struct {
	dialect.SQLServer
}

// Capabilities returns the capabilities of the SQLServer dialect.
func (SQLServer) Capabilities() Capabilities {
	return Capabilities{
		SupportsReturning:      false,
		SupportsOutputInserted: true,

		SupportsInsertDefault:         true,
		SupportsOnConflict:            false,
		SupportsOnConflictSetExcluded: false,
		SupportsOnDuplicateKeyUpdate:  false,

		SupportsUpdateFrom: true,
		SupportsUpdateJoin: false,
	}
}

// NullCoalesce provides a SQLServer specific implementation for COALESCE, especially for time.Time type.
func (d SQLServer) NullCoalesce(column sqlf.Builder, goType reflect.Type) (sqlf.Builder, error) {
	if !CheckNullCoalesceable(goType) {
		return nil, nil
	}
	// Use AssignableTo to handle custom type aliases.
	timeType := reflect.TypeOf(time.Time{})
	if goType.AssignableTo(timeType) {
		// Use ISO 8601 format with UTC timezone ('Z') for robust handling.
		// SQL Server understands this format for both DATETIME2 and DATETIMEOFFSET.
		// Using '0001-01-01T00:00:00Z' is safer than '... 00:00:00Z'.
		return sqlf.F("COALESCE(?, CAST('0001-01-01T00:00:00Z' AS DATETIME2))", column), nil
	}
	if goType.Kind() == reflect.Bool {
		// SQL Server does not have a native boolean type, it uses BIT where 0 is false and 1 is true.
		return sqlf.F("COALESCE(?, 0)", column), nil
	}
	// Fallback to the generic ANSI implementation for other types
	return AnsiSQL{}.NullCoalesce(column, goType)
}
