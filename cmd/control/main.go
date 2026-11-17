// Command control runs the ibox control plane.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ibox/ibox/internal/api"
	"github.com/ibox/ibox/internal/auth"
	"github.com/ibox/ibox/internal/controller"
	"github.com/ibox/ibox/internal/nodes"
	"github.com/ibox/ibox/internal/store"
)

func main() {
	var (
		addr       = flag.String("addr", ":8080", "control plane listen address")
		proxyAddr  = flag.String("proxy-addr", ":8081", "published-ports proxy listen address")
		dbPath     = flag.String("db", "ibox.db", "SQLite database path")
		secret     = flag.String("secret", env("IBOX_SECRET", "dev-secret-change-me"), "HMAC/encryption secret")
		nodeToken  = flag.String("node-token", env("IBOX_NODE_TOKEN", ""), "token agents use to connect")
		adminEmail = flag.String("admin-email", env("IBOX_ADMIN_EMAIL", "admin@ibox.local"), "bootstrap admin email")
		adminPass  = flag.String("admin-password", env("IBOX_ADMIN_PASSWORD", "admin1234"), "bootstrap admin password")
		corsOrigin = flag.String("cors-origin", env("IBOX_CORS_ORIGIN", "http://localhost:3000"), "allowed dashboard origin")
	)
	flag.Parse()

	st, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	am := auth.NewManager(st, *secret)
	hash, err := auth.NewPasswordHash(*adminPass)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}
	uid, err := st.UpsertUser(*adminEmail, "Admin", hash, true)
	if err != nil {
		log.Fatalf("upsert admin: %v", err)
	}
	if _, err := st.EnsurePersonalOrg(uid, *adminEmail); err != nil {
		log.Fatalf("ensure org: %v", err)
	}

	reg := nodes.NewRegistry(nil)
	ctl := controller.New(st, reg)
	reg.OnEvent = ctl.HandleEvent

	tmpl := api.NewTemplateRegistry()
	srv := api.NewServer(st, am, reg, ctl, tmpl)
	srv.NodeToken = *nodeToken
	if err := srv.InitCrypto([]byte(*secret)); err != nil {
		log.Fatalf("init crypto: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go ctl.Run(ctx)
	go runPortProxy(ctx, *proxyAddr, st, reg)

	handler := withCORS(srv, *corsOrigin)
	httpSrv := &http.Server{Addr: *addr, Handler: handler}
	go func() {
		log.Printf("ibox control plane listening on %s (port proxy on %s)", *addr, *proxyAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func withCORS(h http.Handler, origin string) http.Handler {
	// Allow multiple comma-separated origins for dev.
	origins := strings.Split(origin, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originHdr := r.Header.Get("origin")
		allowed := len(origins) == 1 && origins[0] == "*"
		for _, o := range origins {
			if o == originHdr {
				allowed = true
			}
		}
		if allowed {
			w.Header().Set("access-control-allow-origin", originHdr)
			w.Header().Set("access-control-allow-credentials", "true")
		}
		w.Header().Set("access-control-allow-headers", "content-type, authorization, x-api-key, x-ibox-csrf, x-confirm-delete, idempotency-key")
		w.Header().Set("access-control-allow-methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		h.ServeHTTP(w, r)
	})
}

// runPortProxy serves HTTP for published ports: {port}-{machineId}.ibox.localhost
func runPortProxy(ctx context.Context, addr string, st *store.Store, reg *nodes.Registry) {
	suffix := ".ibox.localhost"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if i := strings.Index(host, ":"); i >= 0 {
			host = host[:i]
		}
		name := strings.TrimSuffix(host, suffix)
		parts := strings.SplitN(name, "-", 2)
		if len(parts) != 2 {
			http.Error(w, "expected host: {port}-{machineId}"+suffix, 400)
			return
		}
		portNum, err := strconv.Atoi(parts[0])
		if err != nil {
			http.Error(w, "invalid port", 400)
			return
		}
		machineID := "m_" + parts[1]
		portRow, err := st.GetPort(machineID, portNum)
		if err != nil {
			http.Error(w, "no such published port", 404)
			return
		}
		// private port auth
		if portRow.Access == "private" {
			tok := r.URL.Query().Get("_token")
			if tok == "" {
				if c, cerr := r.Cookie("ibox_port_auth"); cerr == nil {
					tok = c.Value
				}
			}
			if tok == "" {
				tok = strings.TrimPrefix(r.Header.Get("authorization"), "Bearer ")
			}
			if tok == "" {
				w.Header().Set("www-authenticate", "Bearer")
				http.Error(w, "private port: a token is required", 401)
				return
			}
			want, _ := st.GetPortTokenHash(portRow.ID)
			sum := sha256.Sum256([]byte(tok))
			if want == "" || base64.RawURLEncoding.EncodeToString(sum[:]) != want {
				http.Error(w, "invalid token", 403)
				return
			}
			http.SetCookie(w, &http.Cookie{
				Name: "ibox_port_auth", Value: tok, Path: "/",
				SameSite: http.SameSiteLaxMode, MaxAge: 7 * 24 * 3600,
			})
			if r.URL.Query().Get("_token") != "" {
				q := r.URL.Query()
				q.Del("_token")
				r.URL.RawQuery = q.Encode()
				target := r.URL.String()
				if target == "" {
					target = "/"
				}
				http.Redirect(w, r, target, http.StatusFound)
				return
			}
		}
		rp := newPortReverseProxy(st, reg, machineID, portNum)
		rp.ServeHTTP(w, r)
	})

	srv := &http.Server{Addr: addr, Handler: handler}
	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()
	log.Printf("port proxy on %s", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("port proxy: %v", err)
	}
}

func newPortReverseProxy(st *store.Store, reg *nodes.Registry, machineID string, portNum int) *httputil.ReverseProxy {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			node, err := reg.AnyNode()
			if err != nil {
				return nil, err
			}
			res, err := node.Call(ctx, "ports.lookup", map[string]any{"machineId": machineID, "port": portNum}, machineID, 10*time.Second)
			if err != nil {
				return nil, err
			}
			var lr struct {
				HostPort int `json:"hostPort"`
			}
			if json.Unmarshal(res, &lr) != nil || lr.HostPort == 0 {
				return nil, errNoForward
			}
			var d net.Dialer
			return d.DialContext(ctx, "tcp", "127.0.0.1:"+strconv.Itoa(lr.HostPort))
		},
	}
	rp := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = "sandbox"
			req.Host = "sandbox"
			req.RequestURI = ""
		},
		Transport: transport,
		ErrorHandler: func(w http.ResponseWriter, req *http.Request, err error) {
			log.Printf("port proxy %s: %v", req.Host, err)
			http.Error(w, "sandbox port unreachable (is the machine running and the server listening?)", 502)
		},
	}
	return rp
}

var errNoForward = errorString("no port forward")

type errorString string

func (e errorString) Error() string { return string(e) }
