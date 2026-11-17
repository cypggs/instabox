//go:build linux

package agentserver

import (
	"os"
	"syscall"
	"unsafe"
)

type winsize struct {
	Rows, Cols, Xpixel, Ypixel uint16
}

func setWinsize(f *os.File, rows, cols int) {
	ws := &winsize{Rows: uint16(rows), Cols: uint16(cols)}
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TIOCSWINSZ, uintptr(unsafe.Pointer(ws)))
}
