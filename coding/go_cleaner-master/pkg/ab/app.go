package ab

import (
	"code.byted.org/analyzers/go_cleaner/pkg/ab/internal"
	"code.byted.org/analyzers/go_cleaner/pkg/ab/internal/model"
)

type Entry = model.ABEntry

type Output = internal.ABOutput

type Opt struct {
	Opt    internal.Opt
	Output bool
	Clean  bool
	Force  bool

	Debug bool

	PrintInstr bool
}

type Option func(o *Opt)

func WithExpiredKey(expiredKey []string) Option {
	return func(o *Opt) {
		o.Opt.ExpiredKey = expiredKey
	}
}

func WithOnly(only []string) Option {
	return func(o *Opt) {
		o.Opt.Only = only
	}
}

func WithOutput() Option {
	return func(o *Opt) {
		o.Output = true
	}
}

func WithDebug() Option {
	return func(o *Opt) {
		o.Debug = true
	}
}

func WithPrintInstruction() Option {
	return func(o *Opt) {
		o.PrintInstr = true
	}
}

func WithBefore(month int) Option {
	return func(o *Opt) {
		o.Opt.Before = month
	}
}

func WithClean() Option {
	return func(o *Opt) {
		o.Clean = true
	}
}

func WithForce() Option {
	return func(o *Opt) {
		o.Force = true
	}
}

func WithEntry(entry []*Entry) Option {
	return func(o *Opt) {
		o.Opt.Entry = entry
	}
}

func Run(opts ...Option) (out []*Output) {
	o := &Opt{}
	for _, v := range opts {
		v(o)
	}
	o.Opt.ModifiedFunction = internal.NewModifiedFunctionTracker()
	if !o.Force && o.Clean {
		internal.MustHasGitAndAllCommitted(".")
	}
	for i := 0; i < 2; i++ {
		d := internal.NewDetector(o.Opt)
		d.Load()
		d.CollectAll()
		// d.DebugArgValue()
		if o.PrintInstr {
			d.PrintSSAInstruction()
		}
		if o.Debug {
			d.DebugNodeVal()
		}
		if o.Output {
			out = d.DebugABEntryArg()
		}
		if o.Clean {
			d.Refactor()
		}
		if !o.Clean {
			break
		}
	}
	return
}
