//go:build !windows

package main

import "syscall"

func hiddenProcess() *syscall.SysProcAttr { return &syscall.SysProcAttr{} }
