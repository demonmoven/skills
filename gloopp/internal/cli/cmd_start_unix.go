//go:build linux || darwin || freebsd || openbsd || netbsd

package cli

import (
	"syscall"
)

// setPlatformDetachAttr 给 Unix 系平台设置 Setsid（新建 session，脱离终端）。
func setPlatformDetachAttr(attr *syscall.SysProcAttr) {
	attr.Setsid = true
}
