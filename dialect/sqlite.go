package dialect

import (
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
