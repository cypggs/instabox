package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ibox/ibox/internal/auth"
	"github.com/ibox/ibox/internal/model"
)

// Org Secrets: encrypted values with allowed-hosts metadata.
// Values are write-only (never read back), like the original product.

func (s *Server) handleSecrets(w http.ResponseWriter, r *http.Request, ac *auth.Context) {
	switch r.Method {
	case http.MethodGet:
		secrets, err := s.Store.ListSecrets(ac.OrgID)
		if err != nil {
			s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
			return
		}
		count, _ := s.Store.CountSecrets(ac.OrgID)
		writeJSON(w, 200, map[string]any{"secrets": secrets, "used": count, "max": 100})
	case http.MethodPost:
		var req struct {
			Name         string   `json:"name"`
			AllowedHosts []string `json:"allowedHosts"`
			Value        string   `json:"value"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.err(w, 400, badRequest("invalid JSON body"))
			return
		}
		if req.Name == "" || !nameRe.MatchString(strings.ToLower(req.Name)) {
			s.err(w, 400, badRequest("invalid secret name"))
			return
		}
		count, _ := s.Store.CountSecrets(ac.OrgID)
		if count >= 100 {
			s.err(w, 429, &model.Error{Code: "quota_exceeded", Message: "secret limit reached (100)"})
			return
		}
		ct, err := s.encryptJSON(map[string]string{"value": req.Value})
		if err != nil {
			s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
			return
		}
		if err := s.Store.PutSecret(ac.OrgID, req.Name, req.AllowedHosts, ct); err != nil {
			s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "name": req.Name, "updatedAt": time.Now().UTC().Format(time.RFC3339Nano)})
	default:
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
	}
}

func (s *Server) handleSecretByName(w http.ResponseWriter, r *http.Request, ac *auth.Context) {
	name := strings.TrimPrefix(r.URL.Path, "/v1/secrets/")
	if name == "" {
		s.err(w, 404, &model.Error{Code: "not_found", Message: "secret not found"})
		return
	}
	switch r.Method {
	case http.MethodDelete:
		if err := s.Store.DeleteSecret(ac.OrgID, name); err != nil {
			s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
			return
		}
		w.WriteHeader(204)
	case http.MethodPut, http.MethodPatch:
		var req struct {
			AllowedHosts []string `json:"allowedHosts"`
			Value        string   `json:"value"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Value != "" {
			ct, err := s.encryptJSON(map[string]string{"value": req.Value})
			if err != nil {
				s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
				return
			}
			if err := s.Store.PutSecret(ac.OrgID, name, req.AllowedHosts, ct); err != nil {
				s.err(w, 500, &model.Error{Code: "internal", Message: err.Error()})
				return
			}
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	default:
		s.err(w, 405, &model.Error{Code: "bad_request", Message: "method not allowed"})
	}
}
