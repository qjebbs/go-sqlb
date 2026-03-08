package sqlb

import "github.com/qjebbs/go-sqlf/v4"

var _ (sqlf.Builder) = (*Column)(nil)

// Column represents a column reference, which can be used in sqlf.Builder to build column references like `table.column`.
type Column struct {
	Table Table
	Name  string
}

// BuildTo implements sqlf.Builder
func (c *Column) BuildTo(ctx sqlf.Context) (query string, err error) {
	if c == nil {
		return "", nil
	}
	if c.Name == "*" {
		return sqlf.F("?.*", c.Table).BuildTo(ctx)
	}
	return sqlf.F("?.?", c.Table, sqlf.Identifier(c.Name)).BuildTo(ctx)
}
