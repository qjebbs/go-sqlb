package mapper

import "github.com/qjebbs/go-sqlf/v4"

// Options defines options for scanning.
type Options struct {
	style   sqlf.BindStyle
	tags    []string
	dialect Dialect
}

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

// Option defines a function type for setting Options.
type Option func(*Options)

// WithDialect sets the SQL dialect for scanning.
func WithDialect(dialect Dialect) Option {
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

func newDefaultOptions() *Options {
	return &Options{
		dialect: DialectPostgreSQL,
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
