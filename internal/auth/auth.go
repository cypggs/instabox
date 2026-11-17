// Package auth implements session cookies and API key authentication.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/ibox/ibox/internal/store"
)

const (
	SessionCookie = "ibox_session"
	CSRFHeader    = "x-ibox-csrf"
	APIKeyPrefix  = "ibk_"
)

type Context struct {
	UserID    int
	OrgID     string
	Email     string
	Scopes    []string // empty = full access
	SessionID string
	CSRFToken string
	// IsAPIKey distinguishes key auth from session auth.
	IsAPIKey bool
	// KeyID identifies the API key for last-used tracking.
	KeyID string
}

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrMissingScope    = errors.New("missing_scope")
)

type Manager struct {
	Store      *store.Store
	SessionTTL time.Duration
	secret     []byte // HMAC secret for session tokens
}

func NewManager(st *store.Store, secret string) *Manager {
	sec := []byte(secret)
	if len(sec) == 0 {
		sec = make([]byte, 32)
		_, _ = rand.Read(sec)
	}
	return &Manager{Store: st, SessionTTL: 30 * 24 * time.Hour, secret: sec}
}

// NewPasswordHash hashes a password.
func NewPasswordHash(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// IssueSession creates a session and returns (cookieValue, csrfToken).
func (m *Manager) IssueSession(userID int) (string, string, error) {
	tok := make([]byte, 32)
	if _, err := rand.Read(tok); err != nil {
		return "", "", err
	}
	raw := base64.RawURLEncoding.EncodeToString(tok)
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(raw))
	value := raw + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	csrf := base64.RawURLEncoding.EncodeToString(mac.Sum(nil)[:16])
	hash := hashToken(value)
	_, err := m.Store.CreateSession(userID, hash, csrf, m.SessionTTL)
	return value, csrf, err
}

// ValidateSession verifies a cookie value and returns the auth context.
func (m *Manager) ValidateSession(cookieValue string) (*Context, error) {
	parts := strings.Split(cookieValue, ".")
	if len(parts) != 2 {
		return nil, ErrUnauthenticated
	}
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(parts[0]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return nil, ErrUnauthenticated
	}
	sess, err := m.Store.GetSessionByTokenHash(hashToken(cookieValue))
	if err != nil {
		return nil, ErrUnauthenticated
	}
	u, err := m.Store.GetUserByID(sess.UserID)
	if err != nil {
		return nil, ErrUnauthenticated
	}
	orgID, err := m.Store.EnsurePersonalOrg(sess.UserID, u.Email)
	if err != nil {
		return nil, err
	}
	return &Context{
		UserID: sess.UserID, OrgID: orgID, Email: u.Email,
		SessionID: sess.ID, CSRFToken: sess.CSRFToken,
	}, nil
}

func hashToken(v string) string {
	sum := sha256.Sum256([]byte(v))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// ValidateAPIKey verifies an ibk_ key.
func (m *Manager) ValidateAPIKey(key string) (*Context, error) {
	sum := sha256.Sum256([]byte(key))
	kr, err := m.Store.GetAPIKeyByHash(base64.RawURLEncoding.EncodeToString(sum[:]))
	if err != nil {
		return nil, ErrUnauthenticated
	}
	if kr.RevokedAt != "" {
		return nil, ErrUnauthenticated
	}
	if kr.ExpiresAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, kr.ExpiresAt); err == nil && time.Now().UTC().After(t) {
			return nil, ErrUnauthenticated
		}
	}
	m.Store.TouchAPIKey(kr.ID)
	return &Context{
		UserID: 0, OrgID: kr.OrgID, Email: "api-key",
		Scopes: kr.Scopes, IsAPIKey: true, KeyID: kr.ID,
	}, nil
}

// HasScope reports whether the context holds a scope (empty list = full access).
func (c *Context) HasScope(scope string) bool {
	if len(c.Scopes) == 0 {
		return true
	}
	for _, s := range c.Scopes {
		if s == scope || s == "*" {
			return true
		}
		// machines:write covers machines:read
		if strings.HasSuffix(s, ":write") && scope == strings.TrimSuffix(s, "write")+"read" {
			return true
		}
	}
	return false
}

// RequireScope returns an error model if the scope is missing.
func (c *Context) RequireScope(scope string) *ScopeError {
	if !c.HasScope(scope) {
		return &ScopeError{Required: scope}
	}
	return nil
}

type ScopeError struct{ Required string }

func (e *ScopeError) Error() string { return "missing_scope: " + e.Required }
