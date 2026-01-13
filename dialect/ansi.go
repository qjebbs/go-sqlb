package dialect

import (
	"fmt"

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

// CastType casts the given type to the dialect-specific type.
func (AnsiSQL) CastType(typ string) string {
	return fmt.Sprintf("CAST(? AS %s)", typ)
}
