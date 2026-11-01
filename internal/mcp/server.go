package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mwirges/wx/internal/config"
)

const (
	// DefaultPort is the documented grok mcp add port.
	DefaultPort = 9742
	// BindHost is the only address wx mcp serve will listen on.
	BindHost = "127.0.0.1"
	// Path is the Streamable HTTP endpoint.
	Path = "/mcp"
)

// ErrDisabled means serve was asked to run while MCP is off.
var ErrDisabled = errors.New("mcp serve is disabled; set mcp.enabled in config or pass --enable")

// ErrPortInUse means the configured port is taken. The process must exit
// rather than bind a different port. The holder is not signaled.
var ErrPortInUse = errors.New("mcp port already in use; refusing to pick another port")

// ErrAlreadyRunning means a live wx mcp serve holds the pidfile or lock.
// The caller must not start a second listener and must not signal that pid
// unless it is the recorded child and MaySignal allows it.
var ErrAlreadyRunning = errors.New("another wx mcp serve is already running; refusing to start a second listener")

// Options configures one serve process.
type Options struct {
	Enabled         bool
	Port            int
	Version         string
	DefaultLocation string
	// Log receives the ready line. Nil writes to stderr.
	// The bearer token is never written here.
	Log io.Writer
	// RequireToken applies to /mcp and /healthz. There is no healthz bypass.
	RequireToken bool
	// Token is the expected bearer. It is compared in constant time and
	// never logged. Empty with RequireToken is a startup error.
	Token string
	// ParentPID, when > 1, cancels the server after that process exits.
	// The Mac app passes its own pid. 0 means do not watch.
	ParentPID int
	// ExitOnStdinClose cancels the server when Stdin returns EOF.
	// The Mac app holds the write end of that pipe. Off for a normal CLI
	// so a closed or /dev/null stdin does not kill a headless serve.
	ExitOnStdinClose bool
	// Stdin is the pipe to watch. Nil uses os.Stdin when ExitOnStdinClose.
	Stdin io.Reader
	// RuntimeDir holds mcp.pid and mcp.lock. Empty uses the directory of
	// the default config file (~/.config/wx). Tests pass a temp directory.
	RuntimeDir string
}

// ServeConfig is the CLI view of config plus flags. FlagPort is used only
// when FlagPortSet is true. CLI wins over config, then the built-in default.
type ServeConfig struct {
	ConfigEnabled   bool
	FlagEnable      bool
	ConfigPort      int
	FlagPort        int
	FlagPortSet     bool
	Version         string
	DefaultLocation string
}

// ServerInfo is the wx_server_info tool result.
type ServerInfo struct {
	Version         string `json:"version"`
	Port            int    `json:"port"`
	Enabled         bool   `json:"enabled"`
	DefaultLocation string `json:"default_location"`
	Bind            string `json:"bind"`
}

// Server is a loopback Streamable HTTP MCP server.
type Server struct {
	opts    Options
	mcp     *sdkmcp.Server
	hits    atomic.Int64
	slots   chan struct{}
	weather weatherFunc
	radar   radarFunc
	// sliceE replaces live climate, sounding, outlook, and SPC fetches.
	// Nil uses the CLI fetch path. The byte cap still applies.
	sliceE sliceEFunc
}

// ResolveListenPort picks the listen port. An explicit flag, including an
// invalid one, replaces the config value. Zero config port means DefaultPort.
func ResolveListenPort(cfgPort, flagPort int, flagSet bool) (int, error) {
	var p int
	switch {
	case flagSet:
		p = flagPort
	case cfgPort != 0:
		p = cfgPort
	default:
		return DefaultPort, nil
	}
	if p < 1 || p > 65535 {
		return 0, fmt.Errorf("mcp port %d is invalid (want 1-65535)", p)
	}
	return p, nil
}

// Prepare turns config and flags into serve options. It does not listen.
// Disabled is reported before an invalid port so a refused process never
// opens a socket.
func Prepare(in ServeConfig) (Options, error) {
	if !in.ConfigEnabled && !in.FlagEnable {
		return Options{}, ErrDisabled
	}
	port, err := ResolveListenPort(in.ConfigPort, in.FlagPort, in.FlagPortSet)
	if err != nil {
		return Options{}, err
	}
	return Options{
		Enabled:         true,
		Port:            port,
		Version:         in.Version,
		DefaultLocation: in.DefaultLocation,
	}, nil
}

// New builds a server. Port 0 becomes DefaultPort. The server does not listen
// until ListenAndServe or Serve.
func New(opts Options) (*Server, error) {
	if opts.Port == 0 {
		opts.Port = DefaultPort
	}
	if opts.Port < 1 || opts.Port > 65535 {
		return nil, fmt.Errorf("mcp port %d is invalid (want 1-65535)", opts.Port)
	}
	s := &Server{opts: opts}
	s.mcp = sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "wx",
		Version: opts.Version,
	}, nil)
	sdkmcp.AddTool(s.mcp, &sdkmcp.Tool{
		Name:        "wx_server_info",
		Description: "Return this wx MCP server's version, loopback bind, port, enabled flag, and configured default location.",
	}, s.serverInfo)
	s.addWeatherTools()
	return s, nil
}

type serverInfoIn struct{}

func (s *Server) serverInfo(context.Context, *sdkmcp.CallToolRequest, serverInfoIn) (*sdkmcp.CallToolResult, ServerInfo, error) {
	return nil, s.Info(), nil
}

// Info returns the tool payload for this process.
func (s *Server) Info() ServerInfo {
	return ServerInfo{
		Version:         s.opts.Version,
		Port:            s.opts.Port,
		Enabled:         s.opts.Enabled,
		DefaultLocation: s.opts.DefaultLocation,
		Bind:            BindHost,
	}
}

// Hits counts inner /mcp and /healthz dispatches. The Host/Origin guard
// returns before either handler, so a rejected request leaves Hits unchanged.
func (s *Server) Hits() int64 {
	return s.hits.Load()
}

// Handler is the guarded HTTP handler. Tests call it without listening.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/healthz", http.HandlerFunc(s.serveHealth))
	mux.Handle(Path, sdkmcp.NewStreamableHTTPHandler(func(*http.Request) *sdkmcp.Server {
		s.hits.Add(1)
		return s.mcp
	}, nil))
	// Host/Origin first (403), then bearer (401). A rejected bearer never
	// reaches a tool. /healthz uses the same bearer check when required.
	return guard(s.opts.Port, tokenGuard(s.opts.RequireToken, s.opts.Token, mux))
}

func (s *Server) serveHealth(w http.ResponseWriter, r *http.Request) {
	s.hits.Add(1)
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, s.opts.Version)
	_, _ = io.WriteString(w, "\n")
}

// ListenAndServe binds 127.0.0.1 only. A busy port returns ErrPortInUse and
// does not try another port or signal the holder. A disabled server returns
// ErrDisabled and does not open a socket. A second serve against the same
// runtime directory returns ErrAlreadyRunning.
//
// ParentPID and ExitOnStdinClose are how the Mac supervisor reaps an orphan:
// the app passes its pid and holds the child stdin pipe. Either watch
// cancels ctx, Serve shuts down, and the lock is released.
func ListenAndServe(ctx context.Context, opts Options) error {
	if !opts.Enabled {
		return ErrDisabled
	}
	if opts.ParentPID != 0 && opts.ParentPID <= 1 {
		return fmt.Errorf("mcp: refusing --parent-pid %d", opts.ParentPID)
	}
	if opts.ParentPID > 1 && !processAlive(opts.ParentPID) {
		return fmt.Errorf("mcp: parent pid %d is not running; exiting", opts.ParentPID)
	}
	if opts.RequireToken {
		if err := validateToken(opts.Token); err != nil {
			return err
		}
	}
	dir := opts.RuntimeDir
	if dir == "" {
		cfgPath, err := config.Path()
		if err != nil {
			return err
		}
		dir = filepath.Dir(cfgPath)
	}
	lock, err := Acquire(dir)
	if err != nil {
		return err
	}
	defer lock.Release()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if opts.ParentPID > 1 {
		go watchParent(ctx, opts.ParentPID, cancel)
	}
	if opts.ExitOnStdinClose {
		stdin := opts.Stdin
		if stdin == nil {
			stdin = os.Stdin
		}
		go watchStdin(ctx, stdin, cancel)
	}

	s, err := New(opts)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(BindHost, strconv.Itoa(s.opts.Port))
	ln, err := net.Listen("tcp4", addr)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			return fmt.Errorf("%w: %s: not killing an unverified process: %w", ErrPortInUse, addr, err)
		}
		return fmt.Errorf("mcp: listen %s: %w", addr, err)
	}
	return s.Serve(ctx, ln)
}

// Serve takes ownership of ln. The listener must already be 127.0.0.1 and
// the configured port. ctx cancel shuts the server down.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	if ln == nil {
		return errors.New("mcp: nil listener")
	}
	if !s.opts.Enabled {
		_ = ln.Close()
		return ErrDisabled
	}
	tcp, ok := ln.Addr().(*net.TCPAddr)
	if !ok || tcp == nil || tcp.IP == nil || !tcp.IP.Equal(net.ParseIP(BindHost)) {
		_ = ln.Close()
		return fmt.Errorf("mcp: refusing to serve on %s; bind is %s only", ln.Addr(), BindHost)
	}
	if tcp.Port != s.opts.Port {
		_ = ln.Close()
		return fmt.Errorf("mcp: listener port %d does not match configured port %d", tcp.Port, s.opts.Port)
	}

	fmt.Fprintf(s.log(), "wx mcp serve: listening on http://%s%s\n", ln.Addr().String(), Path)

	hs := &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}
	errCh := make(chan error, 1)
	go func() {
		err := hs.Serve(ln)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		errCh <- err
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := hs.Shutdown(shutCtx); err != nil {
			_ = hs.Close()
		}
		return <-errCh
	}
}

func (s *Server) log() io.Writer {
	if s.opts.Log != nil {
		return s.opts.Log
	}
	return os.Stderr
}
