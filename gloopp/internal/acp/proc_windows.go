//go:build windows

package acp

import (
	"os"
	"syscall"
)

// setPlatformChildAttr 让 agent 子进程在 daemon 死时一起死。
// Windows 用 CREATE_NEW_PROCESS_GROUP (0x200) + CREATE_NO_WINDOW (0x08000000)。
// Windows 无 Pdeathsig 等价物；Job Object 是更彻底的方案但改动大，
// 当前依赖 ctx cancel 触发 cmd.Process.Kill。
func setPlatformChildAttr(attr *syscall.SysProcAttr) {
	attr.CreationFlags = 0x00000200 | 0x08000000
}

// killProcessGroup Windows 无进程组信号语义，退回杀主进程。
func killProcessGroup(p *os.Process) {
	if p == nil {
		return
	}
	_ = p.Kill()
}

// setChildMemoryLimit Windows 平台暂不支持子进程内存限制。
func setChildMemoryLimit(_ *os.Process) {}
