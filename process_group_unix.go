//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// 把 cmd 放进独立的进程组，方便后续整组一起杀
func setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// 杀掉 cmd 所在的整个进程组（包括 yt-dlp 派生的 ffmpeg 等子进程）
func killProcessTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	// 负 pid 表示进程组
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
