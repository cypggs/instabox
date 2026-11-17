// Package agentserver implements the node agent: connects to the control
// plane over WebSocket and executes sandbox RPCs.
package agentserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/ibox/ibox/internal/agent/sandbox"
	"github.com/ibox/ibox/internal/model"
	"github.com/ibox/ibox/internal/nodes"
)

type Agent struct {
	ControlURL string
	NodeID     string
	Token      string
	Mgr        *sandbox.Manager
	Templates  *TemplateBuilder
	hostname   string

	ws *websocket.Conn
}

func New(controlURL, nodeID, token string, mgr *sandbox.Manager) *Agent {
	host, _ := os.Hostname()
	return &Agent{
		ControlURL: controlURL, NodeID: nodeID, Token: token,
		Mgr: mgr, Templates: NewTemplateBuilder(mgr), hostname: host,
	}
}

// Run connects and serves until ctx is cancelled.
func (a *Agent) Run(ctx context.Context) error {
	for {
		if err := a.runOnce(ctx); err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				log.Printf("agent: connection lost: %v; reconnecting in 3s", err)
				time.Sleep(3 * time.Second)
			}
		}
		select {
		case <-ctx.Done():
			return nil
		default:
		}
	}
}

func (a *Agent) runOnce(ctx context.Context) error {
	u, err := url.Parse(a.ControlURL)
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("node_id", a.NodeID)
	q.Set("hostname", a.hostname)
	u.RawQuery = q.Encode()
	hdr := http.Header{}
	if a.Token != "" {
		hdr.Set("authorization", "Bearer "+a.Token)
	}
	ws, _, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{
		HTTPHeader: hdr,
	})
	if err != nil {
		return err
	}
	defer ws.CloseNow()
	a.ws = ws
	log.Printf("agent: connected to control plane at %s", a.ControlURL)

	go a.metricsReporter(ctx)

	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			return err
		}
		var msg nodes.Request
		if json.Unmarshal(data, &msg) != nil {
			continue
		}
		if msg.Stream != "" {
			// stream frames: {"stream": id, "data": base64} or {"stream": id, "close": true}
			// (data lives at the top level of the frame, not in params)
			var sf struct {
				Data  string `json:"data"`
				Close bool   `json:"close"`
			}
			_ = json.Unmarshal(data, &sf)
			if msg.Method == "terminal.open" {
				a.openTerminal(&msg)
				continue
			}
			var data []byte
			if sf.Data != "" {
				data, _ = base64.StdEncoding.DecodeString(sf.Data)
			}
			a.HandleInboundData(msg.Stream, data, sf.Close)
			continue
		}
		if msg.Method == "" {
			continue
		}
		go a.handleRPC(ctx, &msg)
	}
}

func (a *Agent) metricsReporter(ctx context.Context) {
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			samples := a.Mgr.AllSamples()
			b, _ := json.Marshal(map[string]any{"event": "metrics.batch", "samples": samples})
			wctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			_ = a.ws.Write(wctx, websocket.MessageText, b)
			cancel()
		}
	}
}

func (a *Agent) reply(id string, result any, err *model.Error) {
	msg := nodes.Response{ID: id, Error: err}
	if result != nil {
		b, e := json.Marshal(result)
		if e != nil {
			msg.Error = &model.Error{Code: "internal", Message: e.Error()}
		} else {
			msg.Result = b
		}
	}
	b, _ := json.Marshal(msg)
	wctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = a.ws.Write(wctx, websocket.MessageText, b)
}

func (a *Agent) handleRPC(ctx context.Context, msg *nodes.Request) {
	var params map[string]any
	if msg.Params != nil {
		_ = json.Unmarshal(msg.Params, &params)
	}
	str := func(k string) string {
		if v, ok := params[k].(string); ok {
			return v
		}
		return ""
	}
	intv := func(k string) int {
		switch v := params[k].(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
		return 0
	}

	switch msg.Method {
	case "sandbox.create":
		a.rpcCreate(ctx, msg, params)
	case "sandbox.start":
		a.rpcStart(msg, str("machineId"), params)
	case "sandbox.stop":
		a.rpcSimple(msg, str("machineId"), func(sb *sandbox.Sandbox) error {
			return a.Mgr.Stop(sb)
		})
	case "sandbox.suspend":
		a.rpcSimple(msg, str("machineId"), func(sb *sandbox.Sandbox) error {
			return a.Mgr.Suspend(sb)
		})
	case "sandbox.resume":
		sb, ok := a.Mgr.Get(str("machineId"))
		if !ok {
			a.reply(msg.ID, nil, &model.Error{Code: "not_found", Message: "no such machine on this node"})
			return
		}
		restored, err := a.Mgr.Resume(sb, nil)
		if err != nil {
			a.reply(msg.ID, nil, &model.Error{Code: "internal", Message: err.Error()})
			return
		}
		a.reply(msg.ID, map[string]any{"restored": restored}, nil)
	case "sandbox.restart":
		sb, ok := a.Mgr.Get(str("machineId"))
		if !ok {
			a.reply(msg.ID, nil, &model.Error{Code: "not_found", Message: "no such machine on this node"})
			return
		}
		if err := a.Mgr.Restart(sb, nil); err != nil {
			a.reply(msg.ID, nil, &model.Error{Code: "internal", Message: err.Error()})
			return
		}
		a.reply(msg.ID, map[string]any{"ok": true}, nil)
	case "sandbox.delete":
		sb, ok := a.Mgr.Get(str("machineId"))
		if !ok {
			a.reply(msg.ID, map[string]any{"ok": true}, nil)
			return
		}
		if err := a.Mgr.Delete(sb); err != nil {
			a.reply(msg.ID, nil, &model.Error{Code: "internal", Message: err.Error()})
			return
		}
		a.reply(msg.ID, map[string]any{"ok": true}, nil)
	case "sandbox.fork":
		a.rpcFork(msg, str("machineId"), str("newMachineId"))
	case "sandbox.exec":
		a.rpcExec(ctx, msg, params)
	case "sandbox.ps":
		sb, ok := a.Mgr.Get(str("machineId"))
		if !ok {
			a.reply(msg.ID, nil, &model.Error{Code: "not_found", Message: "no such machine"})
			return
		}
		procs, err := a.Mgr.PS(sb)
		if err != nil {
			a.reply(msg.ID, nil, &model.Error{Code: "internal", Message: err.Error()})
			return
		}
		a.reply(msg.ID, map[string]any{"processes": procs}, nil)
	case "sandbox.logs":
		a.reply(msg.ID, map[string]any{"console": a.consoleLog(str("machineId"), intv("tail"))}, nil)
	case "sandbox.metrics":
		a.reply(msg.ID, a.Mgr.AllSamples(), nil)
	case "files.list":
		a.rpcFilesList(msg, str("machineId"), str("path"))
	case "files.read":
		a.rpcFilesRead(msg, str("machineId"), str("path"))
	case "files.write":
		a.rpcFilesWrite(msg, params)
	case "ports.publish":
		a.rpcPortsPublish(msg, str("machineId"), intv("port"))
	case "ports.unpublish":
		sb, ok := a.Mgr.Get(str("machineId"))
		if ok {
			a.Mgr.UnpublishPort(sb, intv("port"))
		}
		a.reply(msg.ID, map[string]any{"ok": true}, nil)
	case "ports.lookup":
		sb, ok := a.Mgr.Get(str("machineId"))
		if !ok {
			a.reply(msg.ID, map[string]any{"hostPort": 0}, nil)
			return
		}
		a.reply(msg.ID, map[string]any{"hostPort": a.Mgr.LookupForward(sb, intv("port"))}, nil)
	case "snapshot.save":
		a.rpcSnapshotSave(msg, params)
	case "snapshot.restore":
		a.rpcSnapshotRestore(msg, str("machineId"), str("snapshotId"))
	case "snapshot.list":
		a.rpcSnapshotList(msg, str("snapshotId"), str("path"))
	case "templates.ensure":
		a.rpcTemplatesEnsure(msg, params)
	default:
		a.reply(msg.ID, nil, &model.Error{Code: "not_found", Message: "unknown method " + msg.Method})
	}
}

// streamSend sends a binary frame on a stream.
func (a *Agent) streamSend(streamID string, data []byte) {
	msg, _ := json.Marshal(map[string]any{
		"stream": streamID,
		"data":   base64.StdEncoding.EncodeToString(data),
	})
	wctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = a.ws.Write(wctx, websocket.MessageText, msg)
}

func (a *Agent) streamClose(streamID string) {
	msg, _ := json.Marshal(map[string]any{"stream": streamID, "close": true})
	wctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = a.ws.Write(wctx, websocket.MessageText, msg)
}

func (a *Agent) rpcCreate(ctx context.Context, msg *nodes.Request, params map[string]any) {
	spec := sandbox.CreateSpec{}
	if v, ok := params["spec"].(map[string]any); ok {
		spec.MachineID, _ = v["machineId"].(string)
		spec.Name, _ = v["name"].(string)
		spec.VCPUs = intOf(v["vcpus"])
		spec.MemMiB = intOf(v["memMiB"])
		spec.RootDiskGiB = intOf(v["rootDiskGiB"])
		spec.DiskGiB = intOf(v["diskGiB"])
		spec.GuestUser, _ = v["guestUser"].(string)
		if env, ok := v["env"].(map[string]any); ok {
			spec.Env = map[string]string{}
			for k, val := range env {
				spec.Env[k], _ = val.(string)
			}
		}
		spec.Runtime, _ = v["runtime"].(string)
		spec.Network, _ = v["networkMode"].(string)
	}
	snapshotID, _ := params["snapshotId"].(string)

	// resolve template dir (bootstrap if needed)
	templateID := strOf(params, "templateId")
	if spec.MachineID == "" {
		spec.MachineID = strOf(params, "machineId")
	}
	if templateID == "" {
		templateID = "standard"
	}

	var sb *sandbox.Sandbox
	var err error
	log.Printf("rpc sandbox.create machine=%s template=%s snapshot=%s", spec.MachineID, templateID, snapshotID)
	if snapshotID != "" {
		// create from snapshot CAS
		mf, merr := a.Mgr.ReadManifest(snapshotID)
		if merr != nil {
			a.reply(msg.ID, nil, &model.Error{Code: "not_found", Message: "snapshot content not found on this node"})
			return
		}
		tmp := a.Mgr.MachineTmpDir(spec.MachineID + "-restore")
		os.RemoveAll(tmp)
		defer os.RemoveAll(tmp)
		if err := a.Mgr.ExtractSnapshot(mf.RootDigest, mf.DataDigest, tmp); err != nil {
			a.reply(msg.ID, nil, &model.Error{Code: "internal", Message: err.Error()})
			return
		}
		spec.SnapshotDir = joinPath(tmp, "root-extract")
		spec.DataSource = joinPath(tmp, "data-extract")
		sb, err = a.Mgr.Create(spec)
	} else {
		tmplDir, terr := a.Templates.Ensure(ctx, templateID)
		if terr != nil {
			a.reply(msg.ID, nil, &model.Error{Code: "no_capacity", Message: "template bootstrap failed: " + terr.Error(), Retryable: true})
			return
		}
		spec.TemplateDir = tmplDir
		sb, err = a.Mgr.Create(spec)
	}
	if err != nil {
		a.reply(msg.ID, nil, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	// start immediately for desired running
	if err := a.Mgr.Start(sb, &sandbox.StartSpec{
		VCPUs: spec.VCPUs, MemMiB: spec.MemMiB, GuestUser: spec.GuestUser, Env: spec.Env,
	}); err != nil {
		a.reply(msg.ID, nil, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	a.reply(msg.ID, map[string]any{
		"machineId": sb.MachineID, "state": string(sb.State),
		"rootDir": sb.RootDir, "dataDir": sb.DataDir,
	}, nil)
}

func (a *Agent) rpcStart(msg *nodes.Request, machineID string, params map[string]any) {
	sb, ok := a.Mgr.Get(machineID)
	if !ok {
		a.reply(msg.ID, nil, &model.Error{Code: "not_found", Message: "no such machine on this node"})
		return
	}
	spec := &sandbox.StartSpec{}
	if v, ok := params["spec"].(map[string]any); ok {
		spec.VCPUs = intOf(v["vcpus"])
		spec.MemMiB = intOf(v["memMiB"])
		spec.GuestUser, _ = v["guestUser"].(string)
	}
	if err := a.Mgr.Start(sb, spec); err != nil {
		a.reply(msg.ID, nil, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	a.reply(msg.ID, map[string]any{"machineId": machineID, "state": string(sb.State)}, nil)
}

func (a *Agent) rpcSimple(msg *nodes.Request, machineID string, fn func(*sandbox.Sandbox) error) {
	sb, ok := a.Mgr.Get(machineID)
	if !ok {
		a.reply(msg.ID, nil, &model.Error{Code: "not_found", Message: "no such machine on this node"})
		return
	}
	if err := fn(sb); err != nil {
		a.reply(msg.ID, nil, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	a.reply(msg.ID, map[string]any{"ok": true, "state": string(sb.State)}, nil)
}

func (a *Agent) rpcExec(ctx context.Context, msg *nodes.Request, params map[string]any) {
	machineID := strOf(params, "machineId")
	sb, ok := a.Mgr.Get(machineID)
	if !ok {
		a.reply(msg.ID, nil, &model.Error{Code: "not_found", Message: "no such machine"})
		return
	}
	var argv []string
	if v, ok := params["argv"].([]any); ok {
		for _, item := range v {
			if s, ok := item.(string); ok {
				argv = append(argv, s)
			}
		}
	}
	env := map[string]string{}
	if v, ok := params["env"].(map[string]any); ok {
		for k, val := range v {
			env[k], _ = val.(string)
		}
	}
	var stdin []byte
	if s, ok := params["stdin"].(string); ok && s != "" {
		stdin, _ = base64.StdEncoding.DecodeString(s)
	}
	res, err := a.Mgr.Exec(ctx, sb, argv, strOf(params, "cwd"), env, stdin, intOf(params["timeoutSeconds"]), params["shell"] == true)
	if err != nil {
		code := "internal"
		if err.Error() == "machine_not_running" {
			code = "machine_not_running"
		}
		a.reply(msg.ID, nil, &model.Error{Code: code, Message: err.Error()})
		return
	}
	a.reply(msg.ID, map[string]any{
		"exitCode": res.ExitCode, "stdout": base64.StdEncoding.EncodeToString(res.Stdout),
		"stderr": base64.StdEncoding.EncodeToString(res.Stderr), "truncated": res.Truncated,
	}, nil)
}

func (a *Agent) rpcFilesList(msg *nodes.Request, machineID, path string) {
	sb, ok := a.Mgr.Get(machineID)
	if !ok {
		a.reply(msg.ID, nil, &model.Error{Code: "not_found", Message: "no such machine"})
		return
	}
	entries, err := sb.List(path)
	if err != nil {
		a.reply(msg.ID, nil, fileErr(err))
		return
	}
	a.reply(msg.ID, map[string]any{"entries": entries}, nil)
}

func (a *Agent) rpcFilesRead(msg *nodes.Request, machineID, path string) {
	sb, ok := a.Mgr.Get(machineID)
	if !ok {
		a.reply(msg.ID, nil, &model.Error{Code: "not_found", Message: "no such machine"})
		return
	}
	b, err := sb.ReadFile(path)
	if err != nil {
		a.reply(msg.ID, nil, fileErr(err))
		return
	}
	a.reply(msg.ID, map[string]any{"content": base64.StdEncoding.EncodeToString(b)}, nil)
}

func (a *Agent) rpcFilesWrite(msg *nodes.Request, params map[string]any) {
	machineID := strOf(params, "machineId")
	sb, ok := a.Mgr.Get(machineID)
	if !ok {
		a.reply(msg.ID, nil, &model.Error{Code: "not_found", Message: "no such machine"})
		return
	}
	content, _ := base64.StdEncoding.DecodeString(strOf(params, "content"))
	err := sb.WriteFile(strOf(params, "path"), content, params["parents"] == true)
	if err != nil {
		a.reply(msg.ID, nil, fileErr(err))
		return
	}
	a.reply(msg.ID, map[string]any{"ok": true}, nil)
}

func fileErr(err error) *model.Error {
	switch err {
	case sandbox.ErrNotFound:
		return &model.Error{Code: "not_found", Message: "no such file or directory"}
	case sandbox.ErrMissingParent:
		return &model.Error{Code: "missing_parent", Message: "parent directory does not exist; retry with parents=true"}
	default:
		return &model.Error{Code: "internal", Message: err.Error()}
	}
}

func (a *Agent) rpcPortsPublish(msg *nodes.Request, machineID string, port int) {
	sb, ok := a.Mgr.Get(machineID)
	if !ok {
		a.reply(msg.ID, nil, &model.Error{Code: "not_found", Message: "no such machine"})
		return
	}
	hostPort, err := a.Mgr.PublishPort(sb, port)
	if err != nil {
		a.reply(msg.ID, nil, &model.Error{Code: "machine_not_running", Message: err.Error()})
		return
	}
	a.reply(msg.ID, map[string]any{"hostPort": hostPort}, nil)
}

func (a *Agent) rpcSnapshotSave(msg *nodes.Request, params map[string]any) {
	machineID := strOf(params, "machineId")
	rootOnly := params["rootOnly"] == true
	sb, ok := a.Mgr.Get(machineID)
	if !ok {
		a.reply(msg.ID, nil, &model.Error{Code: "not_found", Message: "no such machine"})
		return
	}
	rootDigest, dataDigest, rootBytes, dataBytes, err := a.Mgr.SaveSnapshot(sb, rootOnly)
	if err != nil {
		a.reply(msg.ID, nil, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	snapID := strOf(params, "snapshotId")
	if snapID != "" {
		_ = a.Mgr.WriteManifest(snapID, rootDigest, dataDigest)
	}
	a.reply(msg.ID, map[string]any{
		"rootDigest": rootDigest, "dataDigest": dataDigest,
		"rootBytes": rootBytes, "dataBytes": dataBytes,
	}, nil)
}

func (a *Agent) rpcSnapshotRestore(msg *nodes.Request, machineID, snapshotID string) {
	mf, err := a.Mgr.ReadManifest(snapshotID)
	if err != nil {
		a.reply(msg.ID, nil, &model.Error{Code: "not_found", Message: "snapshot content not found on this node"})
		return
	}
	sb, ok := a.Mgr.Get(machineID)
	if !ok {
		a.reply(msg.ID, nil, &model.Error{Code: "not_found", Message: "no such machine"})
		return
	}
	tmp := a.Mgr.MachineTmpDir(machineID + "-restore")
	os.RemoveAll(tmp)
	defer os.RemoveAll(tmp)
	if err := a.Mgr.ExtractSnapshot(mf.RootDigest, mf.DataDigest, tmp); err != nil {
		a.reply(msg.ID, nil, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	// replace machine dirs
	a.Mgr.Stop(sb)
	os.RemoveAll(sb.RootDir)
	os.RemoveAll(sb.DataDir)
	if err := sandbox.CopyDir(joinPath(tmp, "root-extract"), sb.RootDir); err != nil {
		a.reply(msg.ID, nil, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	if _, err := os.Stat(joinPath(tmp, "data-extract")); err == nil {
		if err := sandbox.CopyDir(joinPath(tmp, "data-extract"), sb.DataDir); err != nil {
			a.reply(msg.ID, nil, &model.Error{Code: "internal", Message: err.Error()})
			return
		}
	} else {
		os.MkdirAll(sb.DataDir, 0o755)
	}
	sandbox.ChownGuest(sb.DataDir)
	a.reply(msg.ID, map[string]any{"ok": true}, nil)
}

func (a *Agent) rpcSnapshotList(msg *nodes.Request, snapshotID, path string) {
	entries, err := a.Mgr.ListSnapshotDir(snapshotID, path)
	if err != nil {
		a.reply(msg.ID, nil, &model.Error{Code: "not_found", Message: err.Error()})
		return
	}
	a.reply(msg.ID, map[string]any{"entries": entries}, nil)
}

func (a *Agent) rpcFork(msg *nodes.Request, machineID, newMachineID string) {
	src, ok := a.Mgr.Get(machineID)
	if !ok {
		a.reply(msg.ID, nil, &model.Error{Code: "not_found", Message: "no such machine"})
		return
	}
	if src.State == sandbox.StatePaused || src.State == sandbox.StateSuspended {
		a.reply(msg.ID, nil, &model.Error{Code: "fork_unavailable", Message: "a paused machine cannot be forked; resume it first"})
		return
	}
	spec := sandbox.CreateSpec{
		MachineID:   newMachineID,
		SnapshotDir: src.RootDir,
		DataSource:  src.DataDir,
		Runtime:     src.Runtime,
		Network:     src.Network,
	}
	sb, err := a.Mgr.Create(spec)
	if err != nil {
		a.reply(msg.ID, nil, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	if err := a.Mgr.Start(sb, nil); err != nil {
		a.reply(msg.ID, nil, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	a.reply(msg.ID, map[string]any{"machineId": newMachineID, "state": string(sb.State)}, nil)
}

func (a *Agent) rpcTemplatesEnsure(msg *nodes.Request, params map[string]any) {
	id := strOf(params, "templateId")
	if id == "" {
		id = "standard"
	}
	dir, err := a.Templates.Ensure(context.Background(), id)
	if err != nil {
		a.reply(msg.ID, nil, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	a.reply(msg.ID, map[string]any{"templateId": id, "dir": dir}, nil)
}

func (a *Agent) consoleLog(machineID string, tail int) string {
	if tail <= 0 {
		tail = 200
	}
	sb, ok := a.Mgr.Get(machineID)
	if !ok {
		return ""
	}
	b, err := os.ReadFile(joinPath(sb.Dir, "runsc.log"))
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(lines) > tail {
		lines = lines[len(lines)-tail:]
	}
	return strings.Join(lines, "\n") + "\n"
}

func (a *Agent) openTerminal(msg *nodes.Request) {
	var params map[string]any
	_ = json.Unmarshal(msg.Params, &params)
	machineID := strOf(params, "machineId")
	streamID := msg.Stream
	// register the inbound-data channel BEFORE the ok reply goes out, so
	// keystrokes the client sends right after the open are queued, not dropped
	streamData.mu.Lock()
	streamData.ch[streamID] = make(chan []byte, 256)
	streamData.mu.Unlock()
	sb, ok := a.Mgr.Get(machineID)
	if !ok {
		streamData.mu.Lock()
		delete(streamData.ch, streamID)
		streamData.mu.Unlock()
		a.reply(msg.ID, nil, &model.Error{Code: "not_found", Message: "no such machine"})
		a.streamClose(streamID)
		return
	}
	if sb.State != sandbox.StateRunning {
		streamData.mu.Lock()
		delete(streamData.ch, streamID)
		streamData.mu.Unlock()
		a.reply(msg.ID, nil, &model.Error{Code: "machine_not_running", Message: "machine is not running"})
		a.streamClose(streamID)
		return
	}
	a.reply(msg.ID, map[string]any{"ok": true}, nil)
	pty, err := a.Mgr.ExecPTY(context.Background(), sb, "/bin/bash")
	if err != nil {
		streamData.mu.Lock()
		delete(streamData.ch, streamID)
		streamData.mu.Unlock()
		a.streamClose(streamID)
		return
	}
	// pty -> stream
	go func() {
		buf := make([]byte, 64*1024)
		for {
			n, err := pty.Read(buf)
			if n > 0 {
				a.streamSend(streamID, buf[:n])
			}
			if err != nil {
				a.streamClose(streamID)
				return
			}
		}
	}()
	// stream -> pty: handled by data frames in handleStreamData
	go a.terminalInputLoop(streamID, pty)
}

func (a *Agent) terminalInputLoop(streamID string, pty *os.File) {
	// poll for inbound data via a registry of pending channels
	for {
		data, ok := a.nextStreamData(streamID)
		if !ok {
			pty.Close()
			return
		}
		// resize messages are JSON {"cols":..,"rows":..}
		if len(data) > 0 && data[0] == '{' {
			var resize struct {
				Cols int `json:"cols"`
				Rows int `json:"rows"`
			}
			if json.Unmarshal(data, &resize) == nil && resize.Cols > 0 {
				setWinsize(pty, resize.Rows, resize.Cols)
				continue
			}
		}
		if _, err := pty.Write(data); err != nil {
			return
		}
	}
}

// streamDataRegistry bridges inbound stream frames to readers.
var streamData = struct {
	mu sync.Mutex
	ch map[string]chan []byte
}{ch: map[string]chan []byte{}}

func (a *Agent) nextStreamData(streamID string) ([]byte, bool) {
	streamData.mu.Lock()
	ch, ok := streamData.ch[streamID]
	if !ok {
		ch = make(chan []byte, 64)
		streamData.ch[streamID] = ch
	}
	streamData.mu.Unlock()
	d, ok := <-ch
	return d, ok
}

// HandleInboundData is called by the transport when a data frame arrives.
func (a *Agent) HandleInboundData(streamID string, data []byte, closed bool) {
	streamData.mu.Lock()
	ch, ok := streamData.ch[streamID]
	streamData.mu.Unlock()
	if closed {
		if ok {
			close(ch)
		}
		streamData.mu.Lock()
		delete(streamData.ch, streamID)
		streamData.mu.Unlock()
		return
	}
	if ok {
		select {
		case ch <- data:
		default:
		}
	}
}

func intOf(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	}
	return 0
}

func strOf(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

func joinPath(a, b string) string { return fmt.Sprintf("%s/%s", strings.TrimRight(a, "/"), strings.TrimLeft(b, "/")) }
