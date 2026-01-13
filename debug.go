package sqlb

import (
	"fmt"
	"strings"

	"github.com/qjebbs/go-sqlf/v4"
	"github.com/qjebbs/go-sqlf/v4/util"
)

type debugger struct {
	debug bool // debug mode
	name  string
}

// Debug enables debug mode which prints the interpolated query to stdout.
func (b *debugger) Debug(name ...string) {
	b.debug = true
	if len(name) == 0 {
		b.name = "sqlb"
		return
	}
	b.name = strings.Replace(strings.Join(name, "_"), " ", "_", -1)
}

// printDebugQuery prints the debug query to stdout.
func (b *debugger) printIfDebug(ctx *sqlf.Context, query string, args []any) {
	if !b.debug {
		return
	}
	prefix := b.name
	if prefix == "" {
		prefix = "sqlb"
	}
	interpolated, err := util.Interpolate(ctx.Dialect(), query, args)
	if err != nil {
		fmt.Printf("[%s] interpolating: %s\n", prefix, err)
	}
	fmt.Printf("[%s] %s\n", prefix, interpolated)
}
