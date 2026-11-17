//go:build !linux

package sandbox

import (
	"errors"
	"os"
)

func openPTYImpl() (master, slave *os.File, err error) {
	return nil, nil, errors.New("PTY is only supported on Linux (the agent runs in the VM)")
}
