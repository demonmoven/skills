//go:build windows

package cli

import (
	"syscall"
)

// setPlatformDetachAttr 给 Windows 设置无窗口 + 新建进程组。
// CREATE_NEW_PROCESS_GROUP = 0x200, CREATE_NO_WINDOW = 0x08000000
func setPlatformDetachAttr(attr *syscall.SysProcAttr) {
	attr.HideWindow = true
	attr.CreationFlags = 0x00000200 | 0x08000000
}
