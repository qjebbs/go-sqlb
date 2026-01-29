package sqlb

import (
	"github.com/qjebbs/go-sqlf/v4"
)

var _ sqlf.Builder = (*clauseList)(nil)

// clauseList represents a SQL clause that consists of multiple builders
// prefixed with a clause keyword, e.g., SELECT, WHERE, HAVING, GROUP BY, UNION, etc.
type clauseList struct {
	prefix    string
	separator string
	elements  []sqlf.Builder // where conditions, joined with AND.
}

// newPrefixedList creates a new WhereLike instance.
func newPrefixedList(clause, separator string) *clauseList {
	return &clauseList{
		prefix:    clause,
		separator: separator,
	}
}

// SetPrefix sets the clause prefix.
func (b *clauseList) SetPrefix(clause string) *clauseList {
	b.prefix = clause
	return b
}

// Append add an element. e.g.:
//
//	b.Where(sqlf.F(
//		"? = ?",
//		foo.Column("id"), 1,
//	))
func (b *clauseList) Append(s ...sqlf.Builder) *clauseList {
	if s == nil {
		return b
	}
	b.elements = append(b.elements, s...)
	return b
}

// Replace replaces all existing elements with the given ones.
func (b *clauseList) Replace(elements []sqlf.Builder) *clauseList {
	b.elements = elements
	return b
}

// Empty returns whether there is no element.
func (b *clauseList) Empty() bool {
	return b == nil || len(b.elements) == 0
}

// Build implements sqlf.Builder
func (b *clauseList) BuildTo(ctx sqlf.Context) (string, error) {
	if b == nil || len(b.elements) == 0 {
		return "", nil
	}
	if b.prefix == "" {
		return sqlf.Join(b.elements, b.separator).BuildTo(ctx)
	}
	return sqlf.Prefix(
		b.prefix,
		sqlf.Join(b.elements, b.separator),
	).BuildTo(ctx)
}
