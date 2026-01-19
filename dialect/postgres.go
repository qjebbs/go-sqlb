package dialect

import (
	"github.com/qjebbs/go-sqlf/v4/dialect"
)

var _ Dialect = PostgreSQL{}

// PostgreSQL is the ANSI SQL dialect.
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
