package scanner

import "github.com/qjebbs/go-sqlf/v4"

// Options defines options for scanning.
type Options struct {
	style sqlf.BindStyle
	tags  []string
}

// Option defines a function type for setting Options.
type Option func(*Options)

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

func mergeOptions(opts ...Option) *Options {
	options := &Options{
		style: sqlf.BindStyleQuestion,
		tags:  nil,
	}
	for _, opt := range opts {
		opt(options)
	}
	return options
}
