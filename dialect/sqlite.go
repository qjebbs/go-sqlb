package dialect

import (
	"fmt"

	"github.com/qjebbs/go-sqlf/v4/dialect"
)

var _ Dialect = SQLite{}

// SQLite is the ANSI SQL dialect.
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
	}
}

// CastType casts the given type to the dialect-specific type.
func (SQLite) CastType(typ string) string {
	return fmt.Sprintf("CAST(? AS %s)", typ)
}
