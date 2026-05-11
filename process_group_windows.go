//go:build windows

package main

import (
	"os/exec"
	"strconv"
	"syscall"
)

// Windows: 用 CREATE_NEW_PROCESS_GROUP 创建独立进程组
func setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

// Windows: 用 taskkill /T /F 杀掉整个进程树
func killProcessTree(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	kill := exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid))
	return kill.Run()
}
