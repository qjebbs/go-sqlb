package dialect

import (
	"github.com/qjebbs/go-sqlf/v4/dialect"
)

var _ Dialect = MySQL{}

// MySQL is the ANSI SQL dialect.
type MySQL struct {
	dialect.MySQL
}

// Capabilities returns the capabilities of the MySQL dialect.
func (MySQL) Capabilities() Capabilities {
	return Capabilities{
		SupportsReturning:      false,
		SupportsOutputInserted: false,

		SupportsInsertDefault:         true,
		SupportsOnConflict:            false,
		SupportsOnConflictSetExcluded: false,
		SupportsOnDuplicateKeyUpdate:  true,

		SupportsUpdateFrom: false,
		SupportsUpdateJoin: true,
	}
}
