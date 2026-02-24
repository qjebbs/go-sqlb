package dialect

import (
	"database/sql"
	"errors"
	"reflect"

	"github.com/qjebbs/go-sqlf/v4"
	"github.com/qjebbs/go-sqlf/v4/dialect"
)

var (
	// ErrUnsupportedNullCoalesceType is returned when NullCoalesce is called with a Go type that is not supported by the dialect.
	ErrUnsupportedNullCoalesceType = errors.New("unsupported type for NullCoalesce")
)

// Dialect extends dialect.Dialect with additional capabilities.
type Dialect interface {
	dialect.Dialect

	// Capabilities returns the SQL capabilities of the dialect.
	Capabilities() Capabilities

	// NullCoalesce returns a dialect-specific COALESCE expression for a given Go type.
	// This function is only called for types that cannot handle NULLs natively
	// (i.e., non-pointer types that do not implement sql.Scanner).
	// Its purpose is to prevent runtime errors when scanning a NULL database value
	// into a non-nullable Go type.
	//
	// It should return (builder, nil) on success, where builder is the new COALESCE expression.
	// It should return (nil, error) if the dialect cannot provide a zero-value for the given
	// goType (e.g., for time.Time in some dialects).
	NullCoalesce(column sqlf.Builder, goType reflect.Type) (sqlf.Builder, error)
}

// Capabilities represents the SQL capabilities of a dialect.
type Capabilities struct {
	// SupportsReturning indicates whether the dialect supports RETURNING clause.
	SupportsReturning bool
	// SupportsOutputInserted indicates whether the dialect supports OUTPUT clause.
	SupportsOutputInserted bool

	// SupportsInsertDefault indicates whether the dialect supports DEFAULT keyword in INSERT statements.
	SupportsInsertDefault bool
	// SupportsOnConflict indicates whether the dialect supports CONFLICT clause.
	SupportsOnConflict bool
	// SupportsOnConflictSetExcluded indicates whether the dialect supports EXCLUDED keyword in CONFLICT clauses.
	SupportsOnConflictSetExcluded bool
	// SupportsOnDuplicateKeyUpdate indicates whether the dialect supports ON DUPLICATE KEY UPDATE clause.
	SupportsOnDuplicateKeyUpdate bool

	// SupportsUpdateJoin indicates whether the dialect supports JOIN clause in UPDATE statements.
	//
	// For example (MySQL),
	//   UPDATE foo JOIN bar ON foo.id = bar.id SET foo.val = bar.val
	SupportsUpdateJoin bool
	// SupportsUpdateFrom indicates whether the dialect supports FROM clause in UPDATE statements.
	//
	// For example (PostgreSQL),
	//   UPDATE foo SET val = bar.val FROM bar WHERE foo.id = bar.id
	SupportsUpdateFrom bool
}

var scannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()

// CheckNullCoalesceable checks if a type is a candidate for NullCoalesce.
// It returns false if the type is a pointer or implements sql.Scanner,
// as these types can handle NULLs natively.
func CheckNullCoalesceable(goType reflect.Type) bool {
	if goType.Kind() == reflect.Ptr {
		return false
	}
	if goType.Implements(scannerType) {
		return false
	}
	return true
}

// Upgrade attempts to upgrade a sqlf/dialect.Dialect to a sqlb/dialect.Dialect.
func Upgrade(d dialect.Dialect) (Dialect, bool) {
	if dialect, ok := d.(Dialect); ok {
		return dialect, true
	}
	switch v := d.(type) {
	case dialect.PostgreSQL:
		return PostgreSQL{
			PostgreSQL: v,
		}, true
	case dialect.SQLite:
		return SQLite{
			SQLite: v,
		}, true
	case dialect.Oracle:
		return Oracle{
			Oracle: v,
		}, true
	case dialect.SQLServer:
		return SQLServer{
			SQLServer: v,
		}, true
	case dialect.AnsiSQL:
		return AnsiSQL{
			AnsiSQL: v,
		}, true
	case dialect.MySQL:
		return MySQL{
			MySQL: v,
		}, true
	}
	return nil, false
}
