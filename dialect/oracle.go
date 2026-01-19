package dialect

import (
	"github.com/qjebbs/go-sqlf/v4/dialect"
)

var _ Dialect = Oracle{}

// Oracle is the ANSI SQL dialect.
type Oracle struct {
	dialect.Oracle
}

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
