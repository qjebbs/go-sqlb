package dialect

import (
	"reflect"

	"github.com/qjebbs/go-sqlf/v4"
	"github.com/qjebbs/go-sqlf/v4/dialect"
)

var _ Dialect = AnsiSQL{}

// AnsiSQL is the ANSI SQL dialect.
type AnsiSQL struct {
	dialect.AnsiSQL
}

// Capabilities returns the capabilities of the ANSI SQL dialect.
func (AnsiSQL) Capabilities() Capabilities {
	return Capabilities{
		SupportsReturning:      false,
		SupportsOutputInserted: false,

		SupportsInsertDefault:         true,
		SupportsOnConflict:            false,
		SupportsOnConflictSetExcluded: false,
		SupportsOnDuplicateKeyUpdate:  false,

		SupportsUpdateFrom: false,
		SupportsUpdateJoin: false,
	}
}

// NullCoalesce provides a basic implementation for COALESCE.
// It does not handle time.Time correctly and should be overridden by specific dialects.
func (d AnsiSQL) NullCoalesce(column sqlf.Builder, goType reflect.Type) (sqlf.Builder, error) {
	if !CheckNullCoalesceable(goType) {
		return nil, nil
	}
	switch goType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return sqlf.F("COALESCE(?, 0)", column), nil
	case reflect.String:
		return sqlf.F("COALESCE(?, '')", column), nil
	case reflect.Bool:
		// Note: 'FALSE' is standard SQL
		return sqlf.F("COALESCE(?, FALSE)", column), nil
	}
	return nil, ErrUnsupportedNullCoalesceType
}
