//go:build linux || darwin

package output

import (
	"syscall"
	"unsafe"
)

func ioctlWidth(fd uintptr) int {
	var ws winsize
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, tiocgwinsz, uintptr(unsafe.Pointer(&ws)))
	if errno != 0 {
		return 0
	}
	return int(ws.cols)
}
