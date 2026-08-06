package out

import (
	"fmt"
)

func Fatal(code string, msg string) {
	Std.Fatalf("(%s)%s", code, msg)
}
func Fatalf(code string, format string, args ...interface{}) {
	Fatal(code, fmt.Sprintf(format, args...))
}

func Warn(str string) {
	Std.SetWarn(str)
}
func Warnf(format string, args ...interface{}) {
	Std.SetWarn(fmt.Sprintf(format, args...))
}
func SetOutput(name, value string) {
	Std.SetKV(name, value)
}
func SetOutputf(name, format string, args ...interface{}) {
	Std.SetKV(name, fmt.Sprintf(format, args...))
}
func SetInfo(msg string) {
	Std.SetInfo(msg)
}
func SetInfof(format string, args ...interface{}) {
	Std.SetInfo(fmt.Sprintf(format, args...))
}
