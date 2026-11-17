// Port forwarding into sandboxes. With network=host, the sandbox processes bind
// the VM's network directly, so forwarding = listening on a host-side port and
// proxying to 127.0.0.1:{sandbox port} inside the VM (loopback is shared in
// host-net mode). The agent listens on an ephemeral port in the host-forwarded
// range and reports it to the control plane.
package sandbox

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

// ForwardBase is the start of the agent's host-side forward port range.
const ForwardBase = 20000
const ForwardMax = 21000

type forward struct {
	ln     net.Listener
	sandboxPort int
	done   chan struct{}
}

// PublishPort starts a forward hostPort -> sandbox 127.0.0.1:port.
func (m *Manager) PublishPort(sb *Sandbox, port int) (int, error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	if f, ok := sb.forwards[port]; ok && f != nil {
		return f.ln.Addr().(*net.TCPAddr).Port, nil
	}
	if sb.State != StateRunning && sb.State != StatePaused {
		return 0, errors.New("machine_not_running")
	}
	var ln net.Listener
	var err error
	for p := ForwardBase; p < ForwardMax; p++ {
		ln, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err == nil {
			break
		}
	}
	if ln == nil {
		return 0, fmt.Errorf("no forward port available: %w", err)
	}
	f := &forward{ln: ln, sandboxPort: port, done: make(chan struct{})}
	sb.forwards[port] = f
	go m.acceptLoop(sb, f)
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func (m *Manager) acceptLoop(sb *Sandbox, f *forward) {
	for {
		c, err := f.ln.Accept()
		if err != nil {
			select {
			case <-f.done:
				return
			default:
				return
			}
		}
		go proxyConn(c, "127.0.0.1:"+strconv.Itoa(f.sandboxPort))
	}
}

func proxyConn(downstream net.Conn, upstreamAddr string) {
	defer downstream.Close()
	up, err := net.DialTimeout("tcp", upstreamAddr, 10*time.Second)
	if err != nil {
		return
	}
	defer up.Close()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(up, downstream)
		if c, ok := up.(*net.TCPConn); ok {
			c.CloseWrite()
		}
	}()
	go func() {
		defer wg.Done()
		io.Copy(downstream, up)
		if c, ok := downstream.(*net.TCPConn); ok {
			c.CloseWrite()
		}
	}()
	wg.Wait()
}

// UnpublishPort stops a forward.
func (m *Manager) UnpublishPort(sb *Sandbox, port int) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	m.unforwardLocked(sb, port)
}

func (m *Manager) unforwardLocked(sb *Sandbox, port int) {
	if f, ok := sb.forwards[port]; ok {
		close(f.done)
		f.ln.Close()
		delete(sb.forwards, port)
	}
}

// LookupForward returns the host port for a published port.
func (m *Manager) LookupForward(sb *Sandbox, port int) int {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	if f, ok := sb.forwards[port]; ok {
		return f.ln.Addr().(*net.TCPAddr).Port
	}
	return 0
}
