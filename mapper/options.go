package mapper

import (
	"io"
	"os"

	"github.com/qjebbs/go-sqlb"
	"github.com/qjebbs/go-sqlb/internal/util"
)

// Options defines options for scanning.
type Options struct {
	debug       bool
	debugTime   bool
	debugWriter io.Writer

	selectTags           []string
	selectNullZeroTables []string
}

func (o *Options) matchTag(onTags []string) bool {
	if len(onTags) == 0 {
		return true
	}
	for _, tag := range o.selectTags {
		if util.Index(onTags, tag) >= 0 {
			return true
		}
	}
	return false
}
func (o *Options) enableNullZero(name string) bool {
	return util.Index(o.selectNullZeroTables, name) >= 0
}

// Option defines a function type for setting Options.
type Option func(*Options)

// WithDebug enables debug logging with an optional name.
// This option applies only to sqlb builders who print built queries in debug mode.
func WithDebug(writer ...io.Writer) Option {
	return func(o *Options) {
		var w io.Writer
		switch len(writer) {
		case 0:
			w = os.Stdout
		case 1:
			w = writer[0]
		default:
			w = io.MultiWriter(writer...)
		}
		o.debug = true
		o.debugWriter = w
	}
}

// WithDebugTime enables debug logging with time measurement.
func WithDebugTime(writer ...io.Writer) Option {
	return func(o *Options) {
		WithDebug(writer...)(o)
		o.debugTime = true
	}
}

// WithSelectTags is an option for Select() which sets the scan field tags to select.
func WithSelectTags(tags ...string) Option {
	return func(o *Options) {
		o.selectTags = tags
	}
}

// WithSelectCoalesce is an option for Select() which enables COALESCE for fields of the specified tables.
// The zero value is used as the default value for NULL fields.
// To decide whether to enable COALESCE for a field, it matches the tables.Name
// against the effective `table` key value (e.g. `sqlb:"table:foo"`) of the field.
//
// Example:
//
//	type Foo struct {
//		ID  int64  `sqlb:"col:id;table:foo;from:f"`
//		Bar string `sqlb:"col:bar"`
//	}
//	foo := sqlb.NewTable("foo", "f")
//	// SELECT COALESCE("f"."id", 0), COALESCE("f"."bar", '') FROM ...
//	mapper.Select[*Foo](..., mapper.WithSelectCoalesce(foo))
func WithSelectCoalesce(tables ...sqlb.Table) Option {
	return func(o *Options) {
		o.selectNullZeroTables = util.Map(tables, func(t sqlb.Table) string {
			return t.Name
		})
	}
}

func newDefaultOptions() *Options {
	return &Options{
		debugWriter: os.Stdout,
	}
}

func mergeOptions(opts ...Option) *Options {
	options := newDefaultOptions()
	for _, opt := range opts {
		opt(options)
	}
	return options
}
