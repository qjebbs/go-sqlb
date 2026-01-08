package mapper

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/qjebbs/go-sqlf/v4/util"
)

type debugger struct {
	name        string
	measureTime bool

	query string
	args  []any
	msgs  []string

	start time.Time
}

func newDebugger(funcName string, value any, measureTime bool) *debugger {
	return &debugger{
		name:        fmt.Sprintf("%s(%T)", funcName, value),
		measureTime: measureTime,
		start:       time.Now(),
	}
}

func (d *debugger) print() {
	query, err := util.Interpolate(d.query, d.args)
	if err != nil {
		d.msgs = append(d.msgs, fmt.Sprintf(
			"interpolate fail: %s", err,
		))
		query = fmt.Sprintf("%s; %v", d.query, d.args)
	}
	if len(d.msgs) == 0 {
		fmt.Printf("[%s] %s\n", d.name, query)
		return
	}
	fmt.Printf(
		"[%s] %s: %s\n",
		d.name,
		strings.Join(d.msgs, ": "),
		query,
	)
}

func (d *debugger) onQuery(query string, args []any) {
	d.query = query
	d.args = args
	if !d.measureTime {
		return
	}
	elapsed := time.Since(d.start)
	d.msgs = append(d.msgs, fmt.Sprintf(
		"build %s", elapsed,
	))
	d.start = time.Now()
}

func (d *debugger) onExec(err error) {
	if err != nil {
		d.msgs = append(d.msgs, fmt.Sprintf(
			"exec failed: %s", err,
		))
		return
	}
	if !d.measureTime {
		return
	}
	elapsed := time.Since(d.start)
	d.msgs = append(d.msgs, fmt.Sprintf(
		"exec %s", elapsed,
	))
	d.start = time.Now()
}

func (d *debugger) onPostExec(err error) {
	if err != nil {
		d.msgs = append(d.msgs, fmt.Sprintf(
			"post exec failed: %s", err,
		))
		return
	}
	if !d.measureTime {
		return
	}
	elapsed := time.Since(d.start)
	d.msgs = append(d.msgs, fmt.Sprintf(
		"post exec %s", elapsed,
	))
	d.start = time.Now()
}

func wrapErrWithDebugName(funcName string, value any, err error) error {
	if err == nil {
		return err
	}
	// not wrapping well known errors for easier checking
	if errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return fmt.Errorf("%s(%T): %w", funcName, value, err)
}
