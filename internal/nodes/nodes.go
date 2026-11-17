// Package nodes manages node agent connections: registration, RPC dispatch,
// and streaming attachments (terminal, metrics).
package nodes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/ibox/ibox/internal/model"
)

// Request is an RPC request from control plane to agent.
type Request struct {
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	Machine string          `json:"machine,omitempty"`
	Stream  string          `json:"stream,omitempty"`
}

// Response is an RPC response from agent.
type Response struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *model.Error    `json:"error,omitempty"`
}

// Event is an async event from agent (state change, metric sample, logs).
type Event struct {
	Event     string          `json:"event"`
	Machine   string          `json:"machine,omitempty"`
	Node      string          `json:"node,omitempty"`
	State     string          `json:"state,omitempty"`
	Metric    *model.MetricSample `json:"metric,omitempty"`
	Console   string          `json:"console,omitempty"`
	Timestamp time.Time       `json:"-"`
}

// SandboxSpec is the create/restore payload for a sandbox.
type SandboxSpec struct {
	MachineID   string            `json:"machineId"`
	Name        string            `json:"name"`
	RootFSDir   string            `json:"rootfsDir,omitempty"`   // explicit rootfs (fork/restore)
	TemplateDir string            `json:"templateDir,omitempty"` // template CAS dir
	DataDir     string            `json:"dataDir,omitempty"`     // explicit data dir
	VCPUs       int               `json:"vcpus"`
	MemMiB      int               `json:"memMiB"`
	RootDiskGiB int               `json:"rootDiskGiB"`
	DiskGiB     int               `json:"diskGiB"`
	GuestUser   string            `json:"guestUser"`
	Env         map[string]string `json:"env,omitempty"`
	Runtime     string            `json:"runtime,omitempty"` // runsc|runc
	NetworkMode string            `json:"networkMode,omitempty"`
}

// SandboxInfo describes a running sandbox on an agent.
type SandboxInfo struct {
	MachineID string `json:"machineId"`
	State     string `json:"state"` // running|paused|stopped|unknown
	RootDir   string `json:"rootDir"`
	DataDir   string `json:"dataDir"`
	PID       int    `json:"pid,omitempty"`
}

// ExecParams for sandbox exec.
type ExecParams struct {
	MachineID string            `json:"machineId"`
	Argv      []string          `json:"argv"`
	Cwd       string            `json:"cwd,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	TimeoutS  int               `json:"timeoutSeconds,omitempty"`
	Stdin     []byte            `json:"stdin,omitempty"`
	Shell     bool              `json:"shell,omitempty"` // run argv through /bin/sh -c
}

// ExecOutcome is the result of a foreground exec.
type ExecOutcome struct {
	ExitCode  int    `json:"exitCode"`
	Stdout    []byte `json:"stdout"`
	Stderr    []byte `json:"stderr"`
	Truncated bool   `json:"truncated"`
}

// PortForwardInfo tells the control plane how to reach a published port.
type PortForwardInfo struct {
	HostPort int    `json:"hostPort"` // port agent listens on (inside VM), forwarded to host
	SandboxIP string `json:"sandboxIp,omitempty"`
}

// Node is a connected execution node.
type Node struct {
	ID       string
	Hostname string
	Capacity NodeCapacity

	mu       sync.Mutex
	ws       *websocket.Conn
	pending  map[string]chan Response
	streams  map[string]chan []byte
	nodeCtx  context.Context
	cancel   context.CancelFunc
	registry *Registry
}

// OpenStream registers a stream and notifies the agent to open it.
// Frames afterward are exchanged with StreamSend/StreamRecv.
func (n *Node) OpenStream(ctx context.Context, method string, params any, machine string, timeout time.Duration) (string, *model.Error) {
	streamID := fmt.Sprintf("s%d", time.Now().UnixNano())
	ch := make(chan []byte, 256)
	n.mu.Lock()
	if n.nodeCtx.Err() != nil {
		n.mu.Unlock()
		return "", &model.Error{Code: "host_unavailable", Message: "node disconnected", Retryable: true}
	}
	n.streams[streamID] = ch
	n.mu.Unlock()

	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return "", &model.Error{Code: "bad_request", Message: err.Error()}
		}
		raw = b
	}
	req := Request{ID: streamID, Method: method, Params: raw, Machine: machine, Stream: streamID}
	b, err := json.Marshal(req)
	if err != nil {
		return "", &model.Error{Code: "internal", Message: err.Error()}
	}
	wctx, cancel := context.WithTimeout(ctx, timeout+10*time.Second)
	defer cancel()
	if err := n.ws.Write(wctx, websocket.MessageText, b); err != nil {
		n.mu.Lock()
		delete(n.streams, streamID)
		n.mu.Unlock()
		return "", &model.Error{Code: "host_unavailable", Message: err.Error(), Retryable: true}
	}
	return streamID, nil
}

// StreamSend sends one binary frame to the agent stream.
func (n *Node) StreamSend(streamID string, data []byte) error {
	n.mu.Lock()
	if _, ok := n.streams[streamID]; !ok {
		n.mu.Unlock()
		return ErrNodeGone
	}
	n.mu.Unlock()
	msg, _ := json.Marshal(map[string]any{"stream": streamID, "data": base64.StdEncoding.EncodeToString(data)})
	wctx, cancel := context.WithTimeout(n.nodeCtx, 15*time.Second)
	defer cancel()
	return n.ws.Write(wctx, websocket.MessageText, msg)
}

// StreamRecv returns the channel of inbound frames for a stream.
func (n *Node) StreamRecv(streamID string) (<-chan []byte, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	ch, ok := n.streams[streamID]
	return ch, ok
}

// CloseStream removes a stream and notifies the agent.
func (n *Node) CloseStream(streamID string) {
	n.mu.Lock()
	ch, ok := n.streams[streamID]
	delete(n.streams, streamID)
	n.mu.Unlock()
	if ok {
		close(ch)
	}
	msg, _ := json.Marshal(map[string]any{"stream": streamID, "close": true})
	wctx, cancel := context.WithTimeout(n.nodeCtx, 5*time.Second)
	defer cancel()
	_ = n.ws.Write(wctx, websocket.MessageText, msg)
}

type NodeCapacity struct {
	VCPUs int `json:"vcpus"`
	MemMiB int `json:"memMiB"`
	DiskGiB int `json:"diskGiB"`
}

// Registry tracks connected nodes.
type Registry struct {
	mu    sync.RWMutex
	nodes map[string]*Node
	// OnEvent receives async agent events.
	OnEvent func(*Event)
}

func NewRegistry(onEvent func(*Event)) *Registry {
	return &Registry{nodes: map[string]*Node{}, OnEvent: onEvent}
}

var ErrNoNode = errors.New("no execution node connected")
var ErrNodeGone = errors.New("node disconnected")

// Connect registers a node over an already-accepted websocket.
func (r *Registry) Connect(ctx context.Context, id, hostname string, cap NodeCapacity, ws *websocket.Conn) (*Node, error) {
	nctx, cancel := context.WithCancel(context.Background())
	n := &Node{
		ID: id, Hostname: hostname, Capacity: cap,
		ws: ws, pending: map[string]chan Response{}, streams: map[string]chan []byte{},
		nodeCtx: nctx, cancel: cancel, registry: r,
	}
	r.mu.Lock()
	if old, ok := r.nodes[id]; ok {
		old.cancel()
	}
	r.nodes[id] = n
	r.mu.Unlock()

	go n.readLoop()
	log.Printf("node %s (%s) connected: %d vCPU / %d MiB", id, hostname, cap.VCPUs, cap.MemMiB)
	return n, nil
}

func (n *Node) readLoop() {
	defer func() {
		n.cancel()
		n.mu.Lock()
		for id, ch := range n.pending {
			ch <- Response{ID: id, Error: &model.Error{Code: "host_unavailable", Message: "node disconnected", Retryable: true}}
			close(ch)
			delete(n.pending, id)
		}
		for id, ch := range n.streams {
			close(ch)
			delete(n.streams, id)
		}
		n.ws.CloseNow()
		n.mu.Unlock()
	}()
	for {
		_, data, err := n.ws.Read(n.nodeCtx)
		if err != nil {
			return
		}
		var msg struct {
			ID      string          `json:"id"`
			Result  json.RawMessage `json:"result"`
			Error   *model.Error    `json:"error"`
			Event   string          `json:"event"`
			Machine string          `json:"machine"`
			State   string          `json:"state"`
			Metric  *model.MetricSample `json:"metric"`
			Stream  string          `json:"stream"`
			Data    string          `json:"data"`
			Close   bool            `json:"close"`
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg.Stream != "" {
			n.mu.Lock()
			ch, ok := n.streams[msg.Stream]
			n.mu.Unlock()
			if ok {
				if msg.Close {
					n.mu.Lock()
					delete(n.streams, msg.Stream)
					n.mu.Unlock()
					close(ch)
				} else if msg.Data != "" {
					b, err := base64.StdEncoding.DecodeString(msg.Data)
					if err == nil {
						select {
						case ch <- b:
						default: // drop if backpressured
						}
					}
				}
			}
			continue
		}
		if msg.ID != "" {
			n.mu.Lock()
			ch, ok := n.pending[msg.ID]
			if ok {
				delete(n.pending, msg.ID)
			}
			n.mu.Unlock()
			if ok {
				ch <- Response{ID: msg.ID, Result: msg.Result, Error: msg.Error}
				close(ch)
			}
			continue
		}
		if msg.Event != "" {
			n.dispatchEvent(&Event{
				Event: msg.Event, Machine: msg.Machine, Node: n.ID,
				State: msg.State, Metric: msg.Metric, Timestamp: time.Now(),
			})
		}
	}
}

func (n *Node) dispatchEvent(ev *Event) {
	if n.registry != nil && n.registry.OnEvent != nil {
		n.registry.OnEvent(ev)
	}
}

func (n *Node) Remove() {
	n.cancel()
	n.mu.Lock()
	n.ws.CloseNow()
	n.mu.Unlock()
}

// Call performs an RPC with timeout.
func (n *Node) Call(ctx context.Context, method string, params any, machine string, timeout time.Duration) (json.RawMessage, *model.Error) {
	reqID := fmt.Sprintf("r%d", time.Now().UnixNano())
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, &model.Error{Code: "bad_request", Message: err.Error()}
		}
		raw = b
	}
	req := Request{ID: reqID, Method: method, Params: raw, Machine: machine}
	ch := make(chan Response, 1)
	n.mu.Lock()
	if n.nodeCtx.Err() != nil {
		n.mu.Unlock()
		return nil, &model.Error{Code: "host_unavailable", Message: "node disconnected", Retryable: true}
	}
	n.pending[reqID] = ch
	n.mu.Unlock()
	defer func() {
		n.mu.Lock()
		delete(n.pending, reqID)
		n.mu.Unlock()
	}()

	b, err := json.Marshal(req)
	if err != nil {
		return nil, &model.Error{Code: "internal", Message: err.Error()}
	}
	wctx, cancel := context.WithTimeout(ctx, timeout+10*time.Second)
	defer cancel()
	if err := n.ws.Write(wctx, websocket.MessageText, b); err != nil {
		return nil, &model.Error{Code: "host_unavailable", Message: err.Error(), Retryable: true}
	}

	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case resp := <-ch:
		return resp.Result, resp.Error
	case <-t.C:
		return nil, &model.Error{Code: "host_unavailable", Message: "node RPC timeout", Retryable: true}
	case <-n.nodeCtx.Done():
		return nil, &model.Error{Code: "host_unavailable", Message: "node disconnected", Retryable: true}
	}
}

// AnyNode returns a connected node (single-node deployment).
func (r *Registry) AnyNode() (*Node, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, n := range r.nodes {
		return n, nil
	}
	return nil, ErrNoNode
}

func (r *Registry) Get(id string) (*Node, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.nodes[id]
	if !ok {
		return nil, ErrNoNode
	}
	return n, nil
}

func (r *Registry) Remove(id string) {
	r.mu.Lock()
	n, ok := r.nodes[id]
	delete(r.nodes, id)
	r.mu.Unlock()
	if ok {
		n.Remove()
	}
}

func (r *Registry) List() []*Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Node, 0, len(r.nodes))
	for _, n := range r.nodes {
		out = append(out, n)
	}
	return out
}

// WSConn exposes the node websocket for stream hijacking (terminal).
func (n *Node) WSConn() *websocket.Conn { return n.ws }

// Done closes when the node connection ends.
func (n *Node) Done() <-chan struct{} { return n.nodeCtx.Done() }

// NodeForMachine: pick the node that hosts a machine. Single-node: AnyNode.
func (r *Registry) NodeForMachine(nodeID string) (*Node, error) {
	if nodeID == "" {
		return r.AnyNode()
	}
	return r.Get(nodeID)
}
