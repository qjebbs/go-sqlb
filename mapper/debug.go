package mapper

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/qjebbs/go-sqlf/v4/dialect"
	"github.com/qjebbs/go-sqlf/v4/util"
)

type debugger struct {
	name        string
	measureTime bool
	writer      io.Writer

	query string
	args  []any
	msgs  []string

	start time.Time
}

func newDebugger(funcName string, value any, opt *Options) *debugger {
	if opt.debugWriter == nil {
		opt.debugWriter = os.Stdout
	}
	return &debugger{
		name:        fmt.Sprintf("%s(%T)", funcName, value),
		measureTime: opt.debugTime,
		writer:      opt.debugWriter,
		start:       time.Now(),
	}
}

func (d *debugger) print(dialect dialect.Dialect) {
	query, err := util.Interpolate(dialect, d.query, d.args)
	if err != nil {
		d.msgs = append(d.msgs, fmt.Sprintf(
			"interpolate fail: %s", err,
		))
		query = fmt.Sprintf("%s; %v", d.query, d.args)
	}
	d.writer.Write([]byte{'['})
	d.writer.Write([]byte(d.name))
	d.writer.Write([]byte{']', ' '})
	if len(d.msgs) > 0 {
		for _, msg := range d.msgs {
			d.writer.Write([]byte(msg))
			d.writer.Write([]byte{':', ' '})
		}
	}
	d.writer.Write([]byte(query))
	d.writer.Write([]byte{'\n'})
}

func (d *debugger) onBuilt(query string, args []any) {
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

func (d *debugger) onScan(n int, err error) {
	if err != nil {
		d.msgs = append(d.msgs, fmt.Sprintf(
			"scan failed: %s", err,
		))
		return
	}
	if !d.measureTime {
		return
	}
	elapsed := time.Since(d.start)
	d.msgs = append(d.msgs, fmt.Sprintf(
		"scan %d rows %s", n, elapsed,
	))
	d.start = time.Now()
}

func (d *debugger) onPostScan(err error) {
	if err != nil {
		d.msgs = append(d.msgs, fmt.Sprintf(
			"post scan failed: %s", err,
		))
		return
	}
	if !d.measureTime {
		return
	}
	elapsed := time.Since(d.start)
	d.msgs = append(d.msgs, fmt.Sprintf(
		"post scan %s", elapsed,
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
