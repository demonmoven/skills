package logger

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/term"
)

type SymbolFormatter struct {
	CallerPathSegments int
	TimeLayout         string
	Location           *time.Location
	EnableColor        bool
}

func colorize(s, color string, enable bool) string {
	if !enable {
		return s
	}
	return fmt.Sprintf("%s%s\033[0m", color, s)
}

func (f *SymbolFormatter) Format(e *logrus.Entry) ([]byte, error) {
	var b bytes.Buffer

	// 检测是不是 TTY（终端）
	enableColor := f.EnableColor && term.IsTerminal(int(os.Stdout.Fd()))

	// Level 符号 + 颜色
	var sym string
	switch e.Level {
	case logrus.InfoLevel:
		sym = colorize("[+]", "\033[32m", enableColor) // green
	case logrus.WarnLevel:
		sym = colorize("[*]", "\033[33m", enableColor) // yellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		sym = colorize("[-]", "\033[31m", enableColor) // red
	default:
		sym = colorize("[#]", "\033[34m", enableColor) // blue
	}

	// 时间
	loc := f.Location
	if loc == nil {
		loc = time.Local
	}
	layout := f.TimeLayout
	if layout == "" {
		layout = "2006/01/02 15:04:05"
	}
	ts := e.Time.In(loc).Format(layout)

	// 调用位置
	var fileLine string
	if e.HasCaller() {
		file := filepath.ToSlash(e.Caller.File)
		if f.CallerPathSegments > 0 {
			parts := strings.Split(file, "/")
			if len(parts) > f.CallerPathSegments {
				parts = parts[len(parts)-f.CallerPathSegments:]
			}
			file = strings.Join(parts, "/")
		}
		fileLine = fmt.Sprintf("%s:%d", file, e.Caller.Line)
		fileLine = colorize(fileLine, "\033[90m", enableColor) // cyan
	}

	// 拼接
	if fileLine != "" {
		fmt.Fprintf(&b, "%s %s %s %s", sym, ts, fileLine, e.Message)
	} else {
		fmt.Fprintf(&b, "%s %s %s", sym, ts, e.Message)
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

// Init 全局使用
func Init(level logrus.Level) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Fatalf("Time zone loading failed: %v", err)
	}
	time.Local = loc
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&SymbolFormatter{
		CallerPathSegments: 2,
		TimeLayout:         "2006/01/02 15:04:05",
		EnableColor:        true, // 控制是否允许彩色
	})
	logrus.SetLevel(level)
}
