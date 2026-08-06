package endpoint

import (
	"code.byted.org/analyzers/go_cleaner/pkg/endpoint/internal"
	"github.com/sirupsen/logrus"
)

type option struct {
	rm          bool
	handlerPath string
	appRoot     string
}

type Option func(*option)

func WithRM() Option {
	return func(o *option) {
		o.rm = true
	}
}

func WithHandlerPath(handlerPath string) Option {
	return func(o *option) {
		o.handlerPath = handlerPath
	}
}

func WithAppRoot(appRoot string) Option {
	return func(o *option) {
		o.appRoot = appRoot
	}
}

func Run(eps []string, opts ...Option) error {
	opt := option{}
	for _, o := range opts {
		o(&opt)
	}

	app := internal.MustNewApp()
	app.Project.AppRoot = opt.appRoot

	if opt.rm {
		if err := app.Remove(eps); err != nil {
			logrus.Infof("remove endpoint err: %v", err)
			return err
		}
	} else {
		if err := app.Clear(eps, opt.handlerPath); err != nil {
			logrus.Infof("clear endpoint err: %v", err)
			return err
		}
	}
	return nil
}
