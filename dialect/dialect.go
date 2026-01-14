package dialect

import (
	"github.com/qjebbs/go-sqlf/v4/dialect"
)

// Dialect extends dialect.Dialect with additional capabilities.
type Dialect interface {
	dialect.Dialect

	// Capabilities returns the SQL capabilities of the dialect.
	Capabilities() Capabilities

	// CastType casts the given type to the dialect-specific type.
	// Exactly one placeholder "?" is expected in the returned string,
	// which will be replaced with the value to be casted.
	//
	// For example,
	//   PostgreSQL.CastType("TEXT") // "?::TEXT"
	//   SQLite.CastType("TEXT")     // "CAST(? AS TEXT)"
	CastType(typ string) string
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
