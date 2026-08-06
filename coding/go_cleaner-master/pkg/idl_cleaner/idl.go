package idl_cleaner

import "fmt"

type Params struct {
	Root    string
	File    string
	Methods []string
	Remove  bool // 删除相关内容，否则只注释

	Write func(file string, cnt []byte) error
}

type Result struct {
	RemovedMethods []string
	MissMethods    []string
	Err            error
}

func (r Result) SetRemoved(m []string) Result { r.RemovedMethods = m; return r }
func (r Result) SetMiss(m []string) Result    { r.MissMethods = m; return r }
func (r Result) SetErr(err error) Result      { r.Err = err; return r }
func (r Result) SetErrf(format string, args ...interface{}) Result {
	r.Err = fmt.Errorf(format, args...)
	return r
}

type Cleaner interface {
	Clean(p *Params) Result
}
