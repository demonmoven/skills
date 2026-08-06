//go:build darwin || freebsd || openbsd || netbsd

package acp

import (
	"os"
	"syscall"
)

// setPlatformChildAttr 让 agent 子进程在 daemon 死时一起死。
// 非 Linux 的 Unix 无 Pdeathsig，用 Setpgid 把子进程放进独立进程组；
// daemon 退出时 ctx cancel → exec.CommandContext 发 SIGKILL 到 cmd.Process，
// 但孙进程可能孤儿化。Close 路径会向整组 -pgid 发信号兜底。
func setPlatformChildAttr(attr *syscall.SysProcAttr) {
	attr.Setpgid = true
}

// killProcessGroup 杀掉 agent 子进程所在的整个进程组（含孙进程）。
func killProcessGroup(p *os.Process) {
	if p == nil {
		return
	}
	_ = syscall.Kill(-p.Pid, syscall.SIGKILL)
	_ = p.Kill()
}

// setChildMemoryLimit 非 Linux 平台暂不支持子进程内存限制。
func setChildMemoryLimit(_ *os.Process) {}
