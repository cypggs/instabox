//go:build !linux

package agentserver

import "os"

func setWinsize(f *os.File, rows, cols int) {}
