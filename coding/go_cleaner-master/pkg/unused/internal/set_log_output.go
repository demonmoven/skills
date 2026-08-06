package internal

import (
	"io"
	"sync"

	"github.com/sirupsen/logrus"
)

var OutputChangesToStdout = false
var once sync.Once

func DisableLoggingAndOutputChanges() {
	once.Do(func() {
		logrus.SetOutput(io.Discard)
		OutputChangesToStdout = true
	})
}
