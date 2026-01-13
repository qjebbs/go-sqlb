package dialect

import (
	"fmt"

	"github.com/qjebbs/go-sqlf/v4/dialect"
)

var _ Dialect = SQLServer{}

// SQLServer is the ANSI SQL dialect.
type SQLServer struct {
	dialect.SQLServer
}

// Capabilities returns the capabilities of the SQLServer dialect.
func (SQLServer) Capabilities() Capabilities {
	return Capabilities{
		SupportsReturning:      false,
		SupportsOutputInserted: true,

		SupportsInsertDefault:         true,
		SupportsOnConflict:            false,
		SupportsOnConflictSetExcluded: false,
		SupportsOnDuplicateKeyUpdate:  false,

		SupportsUpdateFrom: true,
		SupportsUpdateJoin: false,
	}
}

// CastType casts the given type to the dialect-specific type.
func (SQLServer) CastType(typ string) string {
	return fmt.Sprintf("CAST(? AS %s)", typ)
}
