package clauses

import (
	"github.com/qjebbs/go-sqlf/v4"
)

var _ sqlf.Builder = (*PrefixedList)(nil)

// PrefixedList represents a SQL clause that consists of multiple builders
// prefixed with a clause keyword, e.g., SELECT, WHERE, HAVING, GROUP BY, UNION, etc.
type PrefixedList struct {
	prefix    string
	separator string
	elements  []sqlf.Builder // where conditions, joined with AND.
}

// NewPrefixedList creates a new WhereLike instance.
func NewPrefixedList(clause, separator string) *PrefixedList {
	return &PrefixedList{
		prefix:    clause,
		separator: separator,
	}
}

// SetPrefix sets the clause prefix.
func (b *PrefixedList) SetPrefix(clause string) *PrefixedList {
	b.prefix = clause
	return b
}

// Append add an element. e.g.:
//
//	b.Where(sqlf.F(
//		"? = ?",
//		foo.Column("id"), 1,
//	))
func (b *PrefixedList) Append(s ...sqlf.Builder) *PrefixedList {
	if s == nil {
		return b
	}
	b.elements = append(b.elements, s...)
	return b
}

// Replace replaces all existing elements with the given ones.
func (b *PrefixedList) Replace(elements []sqlf.Builder) *PrefixedList {
	b.elements = elements
	return b
}

// Empty returns whether there is no element.
func (b *PrefixedList) Empty() bool {
	return b == nil || len(b.elements) == 0
}

// Build implements sqlf.Builder
func (b *PrefixedList) Build(ctx *sqlf.Context) (string, error) {
	if b == nil || len(b.elements) == 0 {
		return "", nil
	}
	if b.prefix == "" {
		return sqlf.Join(b.separator, b.elements...).Build(ctx)
	}
	return sqlf.Prefix(
		b.prefix,
		sqlf.Join(b.separator, b.elements...),
	).Build(ctx)
}
