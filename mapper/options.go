package mapper

import (
	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/internal/dialects"
	"github.com/qjebbs/go-sqlb/internal/util"
	"github.com/qjebbs/go-sqlf/v4"
)

// Options defines options for scanning.
type Options struct {
	debug   bool
	style   sqlf.BindStyle
	tags    []string
	dialect dialects.Dialect

	nullZeroTables []string
}

func (o *Options) matchTag(onTags []string) bool {
	if len(onTags) == 0 {
		return true
	}
	for _, tag := range o.tags {
		if util.Index(onTags, tag) >= 0 {
			return true
		}
	}
	return false
}
func (o *Options) enableNullZero(name string) bool {
	return util.Index(o.nullZeroTables, name) >= 0
}

// Option defines a function type for setting Options.
type Option func(*Options)

// WithDebug enables debug logging with an optional name.
func WithDebug() Option {
	return func(o *Options) {
		o.debug = true
	}
}

// WithDialect sets the SQL dialect for scanning.
func WithDialect(dialect dialects.Dialect) Option {
	return func(o *Options) {
		o.dialect = dialect
	}
}

// WithBindStyle sets the bind style for scanning.
func WithBindStyle(style sqlf.BindStyle) Option {
	return func(o *Options) {
		o.style = style
	}
}

// WithTags sets the scan tags for scanning.
func WithTags(tags ...string) Option {
	return func(o *Options) {
		o.tags = tags
	}
}

// WithNullZeroTables sets the tables for which null-zero agents should be enabled.
// To decide whether to enable null-zero agent for a field, it matches the table.Name
// against the effective `table` key value (e.g. `sqlb:table:foo`) of the field.
//
// Example:
//
//	type Foo struct {
//		ID  int64  `sqlb:"col:id;table:foo;from:f"`
//		Bar string `sqlb:"col:bar"`
//	}
//	// All fields of *Foo will use null-zero agents.
//	// *Foo.ID will be set to 0 if NULL is scanned from DB.
//	// *Foo.Bar will be set to "" if NULL is scanned from DB.
//	mapper.Select[*Foo](db, builder,mapper.WithNullZeroTables("foo"))
//
// Enable only when it's not used against massive rows and it's known that the
// table fields could be NULL, e.g., when LEFT JOIN is used.
func WithNullZeroTables(tables ...sqlb.Table) Option {
	return func(o *Options) {
		o.nullZeroTables = util.Map(tables, func(t sqlb.Table) string {
			return t.Name
		})
	}
}

func newDefaultOptions() *Options {
	return &Options{
		dialect: dialects.DialectPostgreSQL,
		style:   sqlf.BindStyleDollar,
		tags:    nil,
	}
}

func mergeOptions(opts ...Option) *Options {
	options := newDefaultOptions()
	for _, opt := range opts {
		opt(options)
	}
	return options
}
