package dialect

import (
	"reflect"
	"time"

	"github.com/qjebbs/go-sqlf/v4"
	"github.com/qjebbs/go-sqlf/v4/dialect"
)

var _ Dialect = Oracle{}

// Oracle is the Oracle dialect.
type Oracle struct {
	dialect.Oracle

	// BoolFalseValue specifies the value used to represent
	// FALSE in NVL/COALESCE expressions, e.g., 0, "N", or "F",
	// defaulting to 0 if not set.
	//
	// This is necessary because Oracle does not have a native boolean type.
	// It is the user's responsibility to ensure their application's
	// scanning logic (e.g., via a custom sql.Scanner implementation)
	// can correctly handle both the original non-NULL values from
	// the column and the configured FALSE value that is
	// returned for NULLs.
	BoolFalseValue any
}

// OracleBoolKind defines the underlying type for boolean representation in Oracle.
type OracleBoolKind int

const (
	// BoolAsNumber represents booleans as NUMBER(1) with values 0 (false) and 1 (true). This is the default.
	BoolAsNumber OracleBoolKind = iota
	// BoolAsCharYN represents booleans as CHAR(1) with values 'N' (false) and 'Y' (true).
	BoolAsCharYN
	// BoolAsCharTF represents booleans as CHAR(1) with values 'F' (false) and 'T' (true).
	BoolAsCharTF
)

// Capabilities returns the capabilities of the Oracle dialect.
func (Oracle) Capabilities() Capabilities {
	return Capabilities{
		SupportsReturning:      true,
		SupportsOutputInserted: false,

		SupportsInsertDefault:         true,
		SupportsOnConflict:            false,
		SupportsOnConflictSetExcluded: false,
		SupportsOnDuplicateKeyUpdate:  false,

		SupportsUpdateFrom: false,
		SupportsUpdateJoin: false,
	}
}

// NullCoalesce provides a Oracle specific implementation for COALESCE, especially for time.Time type.
// Oracle uses NVL function which is similar to COALESCE but takes only two arguments.
func (d Oracle) NullCoalesce(column sqlf.Builder, goType reflect.Type) (sqlf.Builder, error) {
	if !CheckNullCoalesceable(goType) {
		return nil, nil
	}
	switch goType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return sqlf.F("NVL(?, 0)", column), nil
	case reflect.String:
		return sqlf.F("NVL(?, '')", column), nil
	case reflect.Bool:
		if d.BoolFalseValue == nil {
			return sqlf.F("NVL(?, 0)", column), nil
		}
		return sqlf.F("NVL(?, ?)", column, d.BoolFalseValue), nil
	}

	// Use AssignableTo to handle custom type aliases.
	timeType := reflect.TypeOf(time.Time{})
	if goType.AssignableTo(timeType) {
		// Use Oracle's TO_TIMESTAMP_TZ with an explicit UTC timezone ('+00:00').
		// The format mask must match the provided string literal.
		return sqlf.F("NVL(?, TO_TIMESTAMP_TZ('0001-01-01 00:00:00 +00:00', 'YYYY-MM-DD HH24:MI:SS TZH:TZM'))", column), nil
	}

	return nil, ErrUnsupportedNullCoalesceType
}
