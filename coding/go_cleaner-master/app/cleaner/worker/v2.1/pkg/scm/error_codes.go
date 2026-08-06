package scm

import "fmt"

const (
	ErrCodeDataNotFoundInReponse          = 51
	ErrCodeCompilerVersionNotFoundInImage = 52
	ErrCodeFailedToGetTCEInfoByPSM        = 53
	ErrCodeBuildScriptNotFound            = 54
)

var (
	ErrDataNotFoundInResponse         = New(ErrCodeDataNotFoundInReponse, "No data found in response")
	ErrCompilerVersionNotFoundInImage = New(ErrCodeCompilerVersionNotFoundInImage, "Compiler version not found in image")
	ErrFailedToGetTCEInfoByPSM        = New(ErrCodeFailedToGetTCEInfoByPSM, "Get TCE info by PSM fail")
	ErrBuildScriptNotFound            = New(ErrCodeBuildScriptNotFound, "Build script not found")
)

type ErrorCode struct {
	err  error
	msg  string
	code int
}

func (e *ErrorCode) Error() string {
	if e.err == nil {
		return e.msg
	}
	return fmt.Sprintf("%v, err = %v", e.msg, e.err)
}

func (e *ErrorCode) With(err error) *ErrorCode {
	e.err = err
	return e
}

func New(code int, msg string) *ErrorCode {
	return &ErrorCode{
		msg: msg,
	}
}
