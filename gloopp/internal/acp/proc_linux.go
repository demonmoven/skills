//go:build linux

package acp

import (
	"os"
	"syscall"
	"unsafe"
)

// maxAgentVirtualMemBytes 限制 agent 子进程的虚拟地址空间上限（16GB）。
// 防止 agent 进程内存无限膨胀导致宿主机 OOM。
// 4GB 对 Rust/tokio 类 agent（如 traex）太低——tokio 线程栈和 jemalloc
// 惰性映射常触发 RLIMIT_AS 导致 ENOMEM 卡死。16GB 足够正常工作。
const maxAgentVirtualMemBytes uint64 = 16 * 1024 * 1024 * 1024

// setPlatformChildAttr 让 agent 子进程在 daemon 死时一起死。
// Linux 用 Pdeathsig：父进程退出时内核直接发 SIGKILL 给子进程，
// 零成本、零遗漏，不依赖 ctx cancel 或显式 kill。
// 同时 Setpgid 把子进程放进独立进程组，便于显式 kill 整组（超时强制收尾时）。
func setPlatformChildAttr(attr *syscall.SysProcAttr) {
	attr.Setpgid = true
	attr.Pdeathsig = syscall.SIGKILL
}

// setChildMemoryLimit 通过 prlimit64 系统调用限制子进程的虚拟地址空间。
// 必须在 cmd.Start() 之后调用。
func setChildMemoryLimit(p *os.Process) {
	if p == nil {
		return
	}
	lim := syscall.Rlimit{Cur: maxAgentVirtualMemBytes, Max: maxAgentVirtualMemBytes}
	// RLIMIT_AS = 9 on linux
	const rlimitAS = 9
	_, _, _ = syscall.RawSyscall6(
		syscall.SYS_PRLIMIT64,
		uintptr(p.Pid),
		uintptr(rlimitAS),
		uintptr(unsafe.Pointer(&lim)),
		0, 0, 0,
	)
}

// killProcessGroup 杀掉 agent 子进程所在的整个进程组（含孙进程）。
// Setpgid 使子进程成为组长，pgid == pid，负 pid 表示杀整组。
func killProcessGroup(p *os.Process) {
	if p == nil {
		return
	}
	_ = syscall.Kill(-p.Pid, syscall.SIGKILL)
	// 兜底：组 kill 失败时杀主进程
	_ = p.Kill()
}
