// Package api implements the control-plane REST API.
package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/coder/websocket"

	"github.com/ibox/ibox/internal/auth"
	"github.com/ibox/ibox/internal/controller"
	"github.com/ibox/ibox/internal/model"
	"github.com/ibox/ibox/internal/nodes"
	"github.com/ibox/ibox/internal/store"
)

type Server struct {
	Store      *store.Store
	Auth       *auth.Manager
	Registry   *nodes.Registry
	Controller *controller.Controller
	Templates  *TemplateRegistry
	Mux        *http.ServeMux

	// PortProxyBase is the host suffix for published ports, e.g. ".ibox.localhost"
	PortProxyHostSuffix string
	// NodeToken authenticates agents.
	NodeToken string
	// PortProxyListenerAddr is where the port proxy listens (separate port).
	Addr string
}

func NewServer(st *store.Store, am *auth.Manager, reg *nodes.Registry, ctl *controller.Controller, tr *TemplateRegistry) *Server {
	s := &Server{
		Store: st, Auth: am, Registry: reg, Controller: ctl, Templates: tr,
		Mux:                 http.NewServeMux(),
		PortProxyHostSuffix: ".ibox.localhost",
		NodeToken:           "",
	}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Mux.ServeHTTP(w, r)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.Encode(v)
}

func requestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "req_" + base64.RawURLEncoding.EncodeToString(b)
}

func (s *Server) err(w http.ResponseWriter, status int, e *model.Error) {
	if e.RequestID == "" {
		e.RequestID = requestID()
	}
	if e.Retryable {
		if e.RetryAfterSeconds > 0 {
			w.Header().Set("retry-after", fmt.Sprintf("%d", e.RetryAfterSeconds))
		}
	}
	writeJSON(w, status, map[string]any{"error": e})
}

func badRequest(msg string) *model.Error {
	return &model.Error{Code: "bad_request", Message: msg}
}

// ctxAuth returns the authenticated context or writes an error.
func (s *Server) ctxAuth(w http.ResponseWriter, r *http.Request) *auth.Context {
	ctx, err := s.authenticate(r)
	if err != nil {
		code := "unauthenticated"
		hint := `log in via POST /v1/auth/login or create an API key and send it as "Authorization: Bearer <key>" or "X-API-Key: <key>"`
		s.err(w, 401, &model.Error{Code: code, Reason: code, Message: "missing or invalid credentials", Hint: hint, DocsURL: "/docs/errors#unauthenticated"})
		return nil
	}
	return ctx
}

func (s *Server) authenticate(r *http.Request) (*auth.Context, error) {
	// API key first
	if k := r.Header.Get("x-api-key"); k != "" {
		return s.Auth.ValidateAPIKey(strings.TrimSpace(k))
	}
	if h := r.Header.Get("authorization"); strings.HasPrefix(h, "Bearer ") {
		tok := strings.TrimPrefix(h, "Bearer ")
		if strings.HasPrefix(tok, auth.APIKeyPrefix) {
			return s.Auth.ValidateAPIKey(tok)
		}
		return nil, auth.ErrUnauthenticated
	}
	if c, err := r.Cookie(auth.SessionCookie); err == nil {
		return s.Auth.ValidateSession(c.Value)
	}
	return nil, auth.ErrUnauthenticated
}

// requireScope guards a handler with a scope.
func (s *Server) requireScope(ac *auth.Context, scope string) *model.Error {
	if !ac.HasScope(scope) {
		return &model.Error{
			Code:    "missing_scope",
			Message: fmt.Sprintf("this API key lacks the %s scope this route needs", scope),
			Hint:    "create a key with the required scope",
		}
	}
	return nil
}

// csrfCheck applies to session-authenticated mutations.
func (s *Server) csrfCheck(w http.ResponseWriter, r *http.Request, ac *auth.Context) bool {
	if ac.IsAPIKey {
		return true
	}
	if r.Method == "GET" || r.Method == "HEAD" {
		return true
	}
	got := r.Header.Get(auth.CSRFHeader)
	if got == "" || got != ac.CSRFToken {
		s.err(w, 403, &model.Error{Code: "permission_denied", Message: "missing or invalid CSRF token"})
		return false
	}
	return true
}

func (s *Server) routes() {
	s.Mux.HandleFunc("/v1/auth/login", s.handleLogin)
	s.Mux.HandleFunc("/v1/auth/logout", s.withAuth(s.handleLogout))
	s.Mux.HandleFunc("/v1/me", s.withAuth(s.handleMe))
	s.Mux.HandleFunc("/v1/me/terms", s.withAuth(s.handleTerms))
	s.Mux.HandleFunc("/v1/keys", s.withAuth(s.handleKeys))
	s.Mux.HandleFunc("/v1/keys/", s.withAuth(s.handleKeyByID))
	s.Mux.HandleFunc("/v1/orgs", s.withAuth(s.handleOrgs))
	s.Mux.HandleFunc("/v1/orgs/", s.withAuth(s.handleOrgByID))
	s.Mux.HandleFunc("/v1/templates", s.withAuth(s.handleTemplates))
	s.Mux.HandleFunc("/v1/machines", s.withAuth(s.handleMachines))
	s.Mux.HandleFunc("/v1/machines/", s.withAuth(s.handleMachineSub))
	s.Mux.HandleFunc("/v1/operations/", s.withAuth(s.handleOperation))
	s.Mux.HandleFunc("/v1/ports", s.withAuth(s.handlePortsGlobal))
	s.Mux.HandleFunc("/v1/snapshots", s.withAuth(s.handleSnapshots))
	s.Mux.HandleFunc("/v1/snapshots/", s.withAuth(s.handleSnapshotByID))
	s.Mux.HandleFunc("/v1/usage", s.withAuth(s.handleUsage))
	s.Mux.HandleFunc("/v1/secrets", s.withAuth(s.handleSecrets))
	s.Mux.HandleFunc("/v1/secrets/", s.withAuth(s.handleSecretByName))
	s.Mux.HandleFunc("/internal/nodes/connect", s.handleNodeConnect)
	s.Mux.HandleFunc("/v1/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "deploy/openapi.json")
	})
	s.Mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
}

type authedHandler func(w http.ResponseWriter, r *http.Request, ac *auth.Context)

func (s *Server) withAuth(h authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ac := s.ctxAuth(w, r)
		if ac == nil {
			return
		}
		if !s.csrfCheck(w, r, ac) {
			return
		}
		h(w, r, ac)
	}
}

// ---- auth handlers ----

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.err(w, 400, badRequest("invalid JSON body"))
		return
	}
	u, err := s.Store.GetUserByEmail(req.Email)
	if err != nil || !auth.CheckPassword(u.PasswordHash, req.Password) {
		s.err(w, 401, &model.Error{Code: "invalid_key", Message: "invalid email or password", Reason: "invalid_credentials"})
		return
	}
	value, csrf, err := s.Auth.IssueSession(u.ID)
	if err != nil {
		s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: auth.SessionCookie, Value: value, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Expires: time.Now().Add(30 * 24 * time.Hour),
	})
	orgID, _ := s.Store.EnsurePersonalOrg(u.ID, u.Email)
	writeJSON(w, 200, map[string]any{"ok": true, "csrfToken": csrf, "orgId": orgID, "name": u.Name, "email": u.Email})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request, ac *auth.Context) {
	if r.Method != "POST" {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	if ac.SessionID != "" {
		_ = s.Store.DeleteSession(ac.SessionID)
	}
	http.SetCookie(w, &http.Cookie{Name: auth.SessionCookie, Value: "", Path: "/", MaxAge: -1})
	writeJSON(w, 200, map[string]any{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request, ac *auth.Context) {
	if r.Method != "GET" {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	org, err := s.Store.GetOrgByUser(ac.UserID)
	if err != nil {
		s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	me := model.User{
		UserID: ac.UserID, Login: ac.Email, Email: ac.Email,
		Admin: false, CSRFToken: ac.CSRFToken,
		AllowedSpecs: []model.Spec{{VCPUs: 4, MemMiB: 8192, PersistentDiskGiB: 20}},
		Org: *org,
		IdleChoicesSeconds: []int{300, 600, 900, 1200, 1500, 1800},
		Plans: []model.Plan{
			{Tier: "pro", MonthlyFeeUSD: 20, MaxMachines: 500, MaxRunning: 100, MaxVCPUs: 8, MaxDiskGiB: 200},
			{Tier: "builder", MonthlyFeeUSD: 100, MaxMachines: 1000, MaxRunning: 200, MaxVCPUs: 16, MaxDiskGiB: 500},
			{Tier: "team", MonthlyFeeUSD: 500, MaxMachines: 0, MaxRunning: 500, MaxVCPUs: 32, MaxDiskGiB: 1024},
		},
	}
	u, err := s.Store.GetUserByID(ac.UserID)
	if err == nil {
		me.Name = u.Name
		me.Admin = u.Admin
	}
	writeJSON(w, 200, me)
}

func (s *Server) handleTerms(w http.ResponseWriter, r *http.Request, ac *auth.Context) {
	writeJSON(w, 200, map[string]any{"ok": true})
}

// ---- api keys ----

func (s *Server) handleKeys(w http.ResponseWriter, r *http.Request, ac *auth.Context) {
	switch r.Method {
	case http.MethodGet:
		keys, err := s.Store.ListAPIKeys(ac.OrgID)
		if err != nil {
			s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"keys": keys})
	case http.MethodPost:
		var req struct {
			Name      string   `json:"name"`
			ExpiresAt string   `json:"expiresAt,omitempty"`
			Scopes    []string `json:"scopes,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.err(w, 400, badRequest("invalid JSON body"))
			return
		}
		if req.Name == "" {
			s.err(w, 400, badRequest("name is required"))
			return
		}
		// ibk_ + 52 chars
		raw := make([]byte, 40)
		_, _ = rand.Read(raw)
		key := auth.APIKeyPrefix + base64.RawURLEncoding.EncodeToString(raw)[:52]
		sum := sha256.Sum256([]byte(key))
		kid, err := s.Store.InsertAPIKey(ac.OrgID, req.Name, key[:12], base64.RawURLEncoding.EncodeToString(sum[:]), req.Scopes, req.ExpiresAt)
		if err != nil {
			s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
			return
		}
		writeJSON(w, 201, map[string]any{"id": kid, "key": key, "prefix": key[:12], "name": req.Name})
	default:
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
	}
}

func (s *Server) handleKeyByID(w http.ResponseWriter, r *http.Request, ac *auth.Context) {
	if r.Method != http.MethodDelete {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	keyID := strings.TrimPrefix(r.URL.Path, "/v1/keys/")
	if err := s.Store.RevokeAPIKey(keyID, ac.OrgID); err != nil {
		s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	w.WriteHeader(204)
}

// ---- orgs ----

func (s *Server) handleOrgs(w http.ResponseWriter, r *http.Request, ac *auth.Context) {
	if r.Method != http.MethodGet {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	org, err := s.Store.GetOrgByUser(ac.UserID)
	if err != nil {
		s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	org.AllowedSpecs = []model.Spec{{VCPUs: 4, MemMiB: 8192, PersistentDiskGiB: 20}}
	writeJSON(w, 200, map[string]any{"orgs": []model.Org{*org}})
}

func (s *Server) handleOrgByID(w http.ResponseWriter, r *http.Request, ac *auth.Context) {
	if r.Method != http.MethodGet {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	org, err := s.Store.GetOrgByUser(ac.UserID)
	if err != nil {
		s.err(w, 404, &model.Error{Code: "not_found", Message: "org not found"})
		return
	}
	writeJSON(w, 200, org)
}

// ---- templates ----

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request, ac *auth.Context) {
	if r.Method != http.MethodGet {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	writeJSON(w, 200, map[string]any{"templates": s.Templates.List()})
}

// ---- misc ----

func (s *Server) handleOperation(w http.ResponseWriter, r *http.Request, ac *auth.Context) {
	if r.Method != http.MethodGet {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	opID := strings.TrimPrefix(r.URL.Path, "/v1/operations/")
	op, err := s.Store.GetOperation(opID)
	if err != nil {
		s.err(w, 404, &model.Error{Code: "not_found", Message: "no such operation"})
		return
	}
	writeJSON(w, 200, op)
}

func (s *Server) handlePortsGlobal(w http.ResponseWriter, r *http.Request, ac *auth.Context) {
	if r.Method != http.MethodGet {
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
		return
	}
	ports, err := s.Store.ListPorts(ac.OrgID)
	if err != nil {
		s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
		return
	}
	for _, p := range ports {
		s.decoratePortURL(p)
	}
	writeJSON(w, 200, map[string]any{"enabled": true, "ports": ports})
}

var nameRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,62})$`)

// machineByIDOrName fetches a machine scoped to the org.
func (s *Server) machineByIDOrName(ref, orgID string) (*model.Machine, *model.Error) {
	m, err := s.Store.GetMachine(ref, orgID)
	if err != nil {
		return nil, &model.Error{Code: "not_found", Message: "no such machine, or it belongs to another workspace"}
	}
	return m, nil
}

// ensureRunning returns nil if the machine is usable now. If it auto-suspended,
// it wakes it and waits up to 45s.
func (s *Server) ensureRunning(ctx context.Context, m *model.Machine, wait time.Duration) *model.Error {
	if m.State == model.StateRunning {
		return nil
	}
	if m.State == model.StateSuspended && m.AutoSuspendedAt != "" {
		// wake: desired=running, let controller resume
		_ = s.Store.UpdateMachine(m.ID, func(mm *model.Machine) { mm.DesiredState = model.DesiredRunning })
		s.Controller.Enqueue(m.ID)
		deadline := time.Now().Add(wait)
		for time.Now().Before(deadline) {
			cur, err := s.Store.GetMachineByID(m.ID)
			if err != nil {
				return &model.Error{Code: "not_found", Message: "machine disappeared"}
			}
			if cur.State == model.StateRunning {
				return nil
			}
			select {
			case <-ctx.Done():
				return &model.Error{Code: "host_unavailable", Message: "wake cancelled", Retryable: true}
			case <-time.After(500 * time.Millisecond):
			}
		}
		return &model.Error{Code: "operation_conflict", Message: "machine is waking up; try again in a few seconds", Retryable: true}
	}
	return &model.Error{Code: "machine_not_running", Message: "machine is stopped or paused; start or resume it first"}
}

// decoratePortURL sets the public URL for a port.
func (s *Server) decoratePortURL(p *model.Port) {
	host := fmt.Sprintf("%d-%s%s", p.Port, strings.TrimPrefix(p.MachineID, "m_"), s.PortProxyHostSuffix)
	p.URL = "http://" + host
}

// nodeFor returns a node or writes an error.
func (s *Server) nodeFor(ctx context.Context) (*nodes.Node, *model.Error) {
	n, err := s.Registry.AnyNode()
	if err != nil {
		return nil, &model.Error{Code: "no_capacity", Message: "no execution node connected", Retryable: true, RetryAfterSeconds: 15}
	}
	return n, nil
}

// ---- node websocket connect ----

func (s *Server) handleNodeConnect(w http.ResponseWriter, r *http.Request) {
	if s.NodeToken != "" {
		tok := r.URL.Query().Get("token")
		if tok == "" {
			tok = r.Header.Get("authorization")
			tok = strings.TrimPrefix(tok, "Bearer ")
		}
		if tok != s.NodeToken {
			s.err(w, 401, &model.Error{Code: "missing_key", Message: "invalid node token"})
			return
		}
	}
	nodeID := r.URL.Query().Get("node_id")
	if nodeID == "" {
		nodeID = "node-1"
	}
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // token auth instead of origin
	})
	if err != nil {
		return
	}
	var cap nodes.NodeCapacity
	cap.VCPUs = atoiDefault(r.URL.Query().Get("vcpus"), 6)
	cap.MemMiB = atoiDefault(r.URL.Query().Get("mem_mib"), 10240)
	cap.DiskGiB = atoiDefault(r.URL.Query().Get("disk_gib"), 120)
	n, err := s.Registry.Connect(r.Context(), nodeID, r.URL.Query().Get("hostname"), cap, c)
	if err != nil {
		c.CloseNow()
		return
	}
	// block until connection dies
	<-n.Done()
	s.Registry.Remove(nodeID)
}

func atoiDefault(s string, d int) int {
	if s == "" {
		return d
	}
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return d
		}
		n = n*10 + int(ch-'0')
	}
	return n
}

// readBodyLimit reads a body with a max size.
func readBodyLimit(r *http.Request, max int64) ([]byte, error) {
	return io.ReadAll(http.MaxBytesReader(nil, r.Body, max))
}

var _ = log.Printf
var _ = context.Background
