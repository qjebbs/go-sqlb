package sqlb

import (
	"github.com/qjebbs/go-sqlf/v4"
)

var _ (sqlf.Builder) = Table{}

// BuildTo implements sqlf.Builder
func (t Table) BuildTo(ctx sqlf.Context) (query string, err error) {
	uCtx, err := contextUpgrade(ctx)
	if err != nil {
		return "", err
	}
	if deps := dependenciesFromContext(uCtx); deps != nil {
		// collecting
		deps.Tables[t] = true
	}
	return sqlf.Identifier(t.AppliedName()).BuildTo(uCtx)
}

// Table is the table name with optional alias.
type Table struct {
	Name, Alias string
}

// NewTable returns a new Table.
//
// Table is a sqlf.Builder, but builds only the applied name,
// since it's more common to use it to build column references, e.g.:
//
//	t := NewTable("table", "t")
//	sqlf.F("?.id", t)  // t.id
//
// If you want to build fragments like `table As t`, use t.TableAs().
//
//	sqlf.F("LEFT JOIN ?", t.TableAs()) // LEFT JOIN table AS t
func NewTable(name string, alias ...string) Table {
	aliasName := ""
	if len(alias) > 0 {
		aliasName = alias[0]
	}
	return Table{
		Name:  name,
		Alias: aliasName,
	}
}

// IsZero reports whether the table is zero.
func (t Table) IsZero() bool {
	return t.Name == "" && t.Alias == ""
}

// WithAlias returns a new Table with updated alias.
func (t Table) WithAlias(alias string) Table {
	return Table{
		Name:  t.Name,
		Alias: alias,
	}
}

// AppliedName returns the alias if it is not empty, otherwise returns the name.
func (t Table) AppliedName() string {
	if t.Alias != "" {
		return t.Alias
	}
	return t.Name
}

// Column returns a column of the table.
// It adds table prefix to the column name, e.g.: "id" -> "t.id".
//
// For example:
//
//	t := NewTable("table", "t")
//	t.Column("id")  // "t.id"
func (t Table) Column(name string) sqlf.Builder {
	// TODO: build with proper quoting
	return sqlf.F("?.?", t, sqlf.Identifier(name))
}

// AllColumns returns all columns of the table, e.g.: "t.*".
//
// For example:
//
//	t := NewTable("table", "t")
//	t.AllColumns()  // "t.*"
func (t Table) AllColumns() sqlf.Builder {
	return sqlf.F("?.*", t)
}

// TableAs returns a new builder that builds t into fragment like `table AS t`
func (t Table) TableAs() sqlf.Builder {
	return sqlf.Func(func(ctx sqlf.Context) (query string, err error) {
		// report dependency
		t.BuildTo(ctx)
		if t.Alias == "" {
			return sqlf.Identifier(t.Name).BuildTo(ctx)
		}
		return sqlf.F(
			"? AS ?",
			sqlf.Identifier(t.Name),
			sqlf.Identifier(t.Alias),
		).BuildTo(ctx)
	})
}

// Columns returns columns of the table from names.
// It adds table prefix to the column name, e.g.: "id" -> "t.id".
//
// For example:
//
//	t := NewTable("table", "t")
//	t.Columns("id", "name")   // "t.id", "t.name"
func (t Table) Columns(names ...string) []sqlf.Builder {
	r := make([]sqlf.Builder, 0, len(names))
	for _, name := range names {
		r = append(r, t.Column(name))
	}
	return r
}
