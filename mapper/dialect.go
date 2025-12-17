package mapper

import "strings"

// Dialect represents SQL dialects.
type Dialect int

// Dialect constants.
const (
	DialectUnknown Dialect = iota
	DialectOracle
	DialectPostgreSQL
	DialectMySQL
	DialectSQLite
	DialectSQLServer
)

// QuoteIdentifier quotes an identifier based on the dialect.
func (d Dialect) QuoteIdentifier(identifier string) string {
	if !d.isReservedWord(d, identifier) {
		return identifier
	}
	switch d {
	case DialectPostgreSQL, DialectSQLite:
		return `"` + identifier + `"`
	case DialectMySQL:
		return "`" + identifier + "`"
	case DialectSQLServer:
		return "[" + identifier + "]"
	case DialectOracle:
		return `"` + identifier + `"`
	default:
		return identifier
	}
}

// IsReservedWord checks if the given word is a reserved word in the specified dialect.
func (d Dialect) isReservedWord(dialect Dialect, word string) bool {
	m, ok := reservedWords[dialect]
	if !ok {
		return false
	}
	return m[strings.ToLower(word)]
}
