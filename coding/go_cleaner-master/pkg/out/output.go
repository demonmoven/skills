package out

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const (
	escapedNewLineSuffix = "\\++"
)

var (
	Std = output{out: os.Stdout, err: os.Stderr}

	keyEscaper = strings.NewReplacer(
		"::", "#@double-colon@#",
		"\n", "#@newline@#",
	)
	valueEscaper = strings.NewReplacer(
		"\n", escapedNewLineSuffix+"\n",
	)
)

type output struct {
	out io.Writer
	err io.Writer

	onExit []func()
}

type Procedure struct {
	name      string
	id        int
	startTime time.Time
	flow      *flow
}

type flow struct {
	name   string
	nextID int
	o      *output
	cur    *Procedure
}

func (p *Procedure) Next(name string) *Procedure {
	endProc := p.flow.cur
	if p.flow.name != "" {
		p.flow.o.SetKV(fmt.Sprintf("flow:%s,proc:%s,id:%d#cost", p.flow.name, endProc.name, time.Since(endProc.startTime)), time.Since(p.startTime).String())
	} else {
		p.flow.o.SetKV(fmt.Sprintf("proc:%s,id:%d#cost", endProc.name, time.Since(endProc.startTime)), time.Since(p.startTime).String())
	}
	next := &Procedure{
		name:      name,
		startTime: time.Now(),
		id:        p.flow.nextID,
		flow:      p.flow,
	}
	p.flow.cur = next
	p.flow.nextID++
	return next
}

func (p *Procedure) Done() {
	endProc := p.flow.cur
	if p.flow.name != "" {
		p.flow.o.SetKV(fmt.Sprintf("flow:%s,proc:%s,id:%d#cost", p.flow.name, endProc.name, endProc.id), time.Since(p.startTime).String())
	} else {
		p.flow.o.SetKV(fmt.Sprintf("proc:%s,id:%d#cost", endProc.name, endProc.id), time.Since(p.startTime).String())
	}
	p.flow.cur = nil
	p.flow.nextID++
}

func (o *output) Procedure(name string) *Procedure {
	p := &Procedure{
		name:      name,
		startTime: time.Now(),
		id:        1,
	}
	f := &flow{
		nextID: 2,
		o:      o,
		cur:    p,
	}
	p.flow = f
	o.onExit = append(o.onExit, func() {
		if f.cur != nil {
			f.cur.Done()
		}
	})
	return p
}

// ::set-output::escape_key(key)::escaped_value(value)
func (o *output) SetKV(k, v string) {
	_, _ = fmt.Fprintf(o.out, "\n::set-output::%s::%s\n", keyEscaper.Replace(k), valueEscaper.Replace(v))
}

// ::set-warn::escaped_value(msg)
func (o *output) SetWarn(msg string) {
	_, _ = fmt.Fprintf(o.out, "\n::set-warn::%s\n", valueEscaper.Replace(msg))
}

// ::set-warn::escaped_value(msg)
func (o *output) SetWarnf(format string, args ...interface{}) {
	o.SetWarn(fmt.Sprintf(format, args...))
}

// ::set-error::escaped_value(msg)
func (o *output) SetError(err error) {
	_, _ = fmt.Fprintf(o.err, "\n::set-error::%s\n", valueEscaper.Replace(err.Error()))
}

// ::set-error::escaped_value(msg)
func (o *output) SetErrorf(format string, args ...interface{}) {
	o.SetError(fmt.Errorf(format, args...))
}

// ::set-fatal::escaped_value(msg) then os.Exit(1)
func (o *output) Fatalf(format string, args ...interface{}) {
	o.Fatal(fmt.Errorf(format, args...))
}

// ::set-fatal::escaped_value(msg) then os.Exit(1)
func (o *output) Fatal(err error) {
	_, _ = fmt.Fprintf(o.err, "\n::set-fatal::%s\n", valueEscaper.Replace(err.Error()))
	for _, fn := range o.onExit {
		fn()
	}
	os.Exit(1)
}

// ::set-info::escaped_value(msg)
func (o *output) SetInfo(msg string) {
	_, _ = fmt.Fprintf(o.err, "\n::set-info::%s\n", valueEscaper.Replace(msg))
}

// ::set-info::escaped_value(msg)
func (o *output) SetInfof(format string, args ...interface{}) {
	o.SetInfo(fmt.Sprintf(format, args...))
}

func (o *output) SetOnExit(fn func()) {
	o.onExit = append(o.onExit, fn)
}

func NewOutput(out, err io.Writer) *output {
	return &output{out: out, err: err}
}

func NewBytesOutput(out, err *[]byte) *output {
	return &output{out: &bytesWriter{out}, err: &bytesWriter{err}}
}

type bytesWriter struct {
	buf *[]byte
}

func (b *bytesWriter) Write(p []byte) (n int, err error) {
	*b.buf = append(*b.buf, p...)
	return len(p), nil
}
