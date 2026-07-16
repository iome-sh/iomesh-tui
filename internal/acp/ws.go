package acp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// ServeOptions configures the ACP WebSocket HTTP listener.
type ServeOptions struct {
	// Listen is host:port. Default 127.0.0.1:7400 (loopback-only for safety).
	Listen string
	// Path is the WebSocket upgrade path. Default /acp.
	Path string
	// Token if non-empty requires Authorization: Bearer <token> or ?token=.
	Token string
	// AllowAnyOrigin permits browser clients from any Origin (default true for local DX).
	// Prefer Token when exposing beyond loopback.
	AllowAnyOrigin bool
	// ACP options shared by each connection (each conn gets its own Server).
	Options Options
}

// DefaultListen is the safe default bind address.
const DefaultListen = "127.0.0.1:7400"

// DefaultWSPath is the WebSocket upgrade path.
const DefaultWSPath = "/acp"

// ListenAndServe starts an HTTP server that upgrades WebSocket connections to ACP JSON-RPC.
// Each connection runs an independent ACP session (same protocol as stdio).
// Blocks until ctx is cancelled or the listener fails.
func ListenAndServe(ctx context.Context, so ServeOptions) error {
	if so.Listen == "" {
		so.Listen = DefaultListen
	}
	if so.Path == "" {
		so.Path = DefaultWSPath
	}
	if !strings.HasPrefix(so.Path, "/") {
		so.Path = "/" + so.Path
	}
	logger := so.Options.Logger
	if logger == nil {
		logger = New(so.Options).logger
		so.Options.Logger = logger
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ready\n"))
	})
	mux.HandleFunc("GET "+so.Path, func(w http.ResponseWriter, r *http.Request) {
		if so.Token != "" && !authorize(r, so.Token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		opts := &websocket.AcceptOptions{}
		if so.AllowAnyOrigin {
			opts.OriginPatterns = []string{"*"}
		}
		conn, err := websocket.Accept(w, r, opts)
		if err != nil {
			logger.Error("acp ws accept", "err", err)
			return
		}
		// One ACP server per connection (isolated sessions map).
		srv := New(so.Options)
		runErr := srv.RunWebSocket(r.Context(), conn)
		_ = conn.Close(websocket.StatusNormalClosure, "")
		if runErr != nil && !errors.Is(runErr, context.Canceled) && !errors.Is(runErr, io.EOF) {
			logger.Debug("acp ws session end", "err", runErr)
		}
	})

	httpSrv := &http.Server{
		Addr:              so.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	ln, err := net.Listen("tcp", so.Listen)
	if err != nil {
		return fmt.Errorf("acp serve listen %s: %w", so.Listen, err)
	}
	actual := ln.Addr().String()
	logger.Info("acp websocket listening",
		"addr", actual,
		"path", so.Path,
		"token_required", so.Token != "",
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- httpSrv.Serve(ln)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
		err := ctx.Err()
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func authorize(r *http.Request, token string) bool {
	if token == "" {
		return true
	}
	if r.URL.Query().Get("token") == token {
		return true
	}
	h := r.Header.Get("Authorization")
	const p = "Bearer "
	if strings.HasPrefix(h, p) && strings.TrimSpace(h[len(p):]) == token {
		return true
	}
	return false
}

// wsWriter adapts a WebSocket connection to io.Writer for Server.write (JSON text frames).
type wsWriter struct {
	ctx  context.Context
	conn *websocket.Conn
	mu   sync.Mutex
}

func (w *wsWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	// Server.write appends '\n' for stdio; strip for cleaner WS frames.
	msg := bytesTrimRightNewline(p)
	err := w.conn.Write(w.ctx, websocket.MessageText, msg)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func bytesTrimRightNewline(p []byte) []byte {
	for len(p) > 0 && (p[len(p)-1] == '\n' || p[len(p)-1] == '\r') {
		p = p[:len(p)-1]
	}
	return p
}

// RunWebSocket serves ACP JSON-RPC over a WebSocket connection.
// Each text message is one JSON-RPC request (optional trailing newline).
// Responses and session/update notifications are sent as text frames.
func (s *Server) RunWebSocket(ctx context.Context, conn *websocket.Conn) error {
	cfg, err := loadConfig(s.opts.ConfigPath)
	if err != nil {
		return err
	}
	s.cfg = cfg
	if s.opts.Yolo {
		s.cfg.Agent.Yolo = true
	}
	if s.opts.Workspace != "" {
		s.cfg.Agent.Workspace = s.opts.Workspace
	}
	s.out = &wsWriter{ctx: ctx, conn: conn}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		_, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		line := strings.TrimSpace(string(data))
		if line == "" {
			continue
		}
		if err := s.handleLine(ctx, line); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				return nil
			}
			s.logger.Error("acp ws handle", "err", err)
		}
	}
}
