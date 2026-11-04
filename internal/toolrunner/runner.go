// Package toolrunner runs one fixed HTTP adapter. It is not a shell API.
package toolrunner

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/Sanskarzz/tsunami/internal/toolcontract"
)

//go:embed http-read.sh
var script []byte

const ScriptPath = "/opt/tsunami/http-read.sh"
const outputLimit = 128 << 10

func ScriptDigest() string { sum := sha256.Sum256(script); return hex.EncodeToString(sum[:]) }

type Forward func(context.Context, toolcontract.HTTPIntent) (json.RawMessage, error)

type Runner struct {
	ctx                     context.Context
	cancel                  context.CancelFunc
	server                  *http.Server
	dir, socket, scriptPath string
	mu                      sync.Mutex
	calls                   int
	busy                    chan struct{}
}

func New(ctx context.Context, forward Forward) (*Runner, error) {
	return newRunner(ctx, forward, ScriptPath)
}

func newRunner(ctx context.Context, forward Forward, scriptPath string) (*Runner, error) {
	data, err := os.ReadFile(scriptPath)
	if err != nil || !bytes.Equal(data, script) || forward == nil {
		return nil, errors.New("reviewed HTTP adapter is unavailable or changed")
	}
	dir, err := os.MkdirTemp("", "tsunami-http-")
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	r := &Runner{ctx: ctx, cancel: cancel, dir: dir, socket: filepath.Join(dir, "bridge.sock"), scriptPath: scriptPath, busy: make(chan struct{}, 1)}
	listener, err := net.Listen("unix", r.socket)
	if err != nil {
		cancel()
		_ = os.RemoveAll(dir)
		return nil, err
	}
	if err = os.Chmod(r.socket, 0600); err != nil {
		_ = listener.Close()
		cancel()
		_ = os.RemoveAll(dir)
		return nil, err
	}
	r.server = &http.Server{ReadHeaderTimeout: time.Second, ReadTimeout: 2 * time.Second, WriteTimeout: 13 * time.Second, IdleTimeout: time.Second, MaxHeaderBytes: 4096,
		BaseContext: func(net.Listener) context.Context { return ctx },
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Method != "POST" || req.URL.Path != "/request" || req.URL.RawQuery != "" || req.URL.RawPath != "" {
				http.Error(w, "unsupported adapter operation", 400)
				return
			}
			select {
			case r.busy <- struct{}{}:
				defer func() { <-r.busy }()
			default:
				http.Error(w, "adapter busy", 429)
				return
			}
			data, readErr := io.ReadAll(http.MaxBytesReader(w, req.Body, 4096))
			intent, decodeErr := toolcontract.DecodeHTTPIntent(data)
			if readErr != nil || decodeErr != nil {
				http.Error(w, "invalid HTTP intent", 400)
				return
			}
			opctx, done := context.WithTimeout(req.Context(), 12*time.Second)
			defer done()
			out, err := forward(opctx, intent)
			if err != nil {
				http.Error(w, "broker denied or unavailable", 403)
				return
			}
			if len(out) > outputLimit || !json.Valid(out) {
				http.Error(w, "invalid broker result", 502)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(out)
		}),
	}
	go func() { _ = r.server.Serve(listener) }()
	return r, nil
}

func (r *Runner) Close() { r.cancel(); _ = r.server.Close(); _ = os.RemoveAll(r.dir) }

func (r *Runner) Execute(ctx context.Context, in toolcontract.HTTPIntent) (json.RawMessage, error) {
	r.mu.Lock()
	if r.calls >= 8 {
		r.mu.Unlock()
		return nil, errors.New("adapter invocation limit reached")
	}
	r.calls++
	r.mu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	stop := context.AfterFunc(r.ctx, cancel)
	defer stop()
	data, err := json.Marshal(in)
	if err != nil || len(data) > 4096 {
		return nil, errors.New("invalid HTTP intent")
	}
	out, err := command(ctx, "/bin/sh", []string{r.scriptPath, r.socket}, data)
	if err != nil {
		return nil, errors.New("HTTP adapter failed or was cancelled; request is not retried")
	}
	if !json.Valid(out) {
		return nil, errors.New("invalid HTTP adapter result")
	}
	return json.RawMessage(out), nil
}

type limitedBuffer struct {
	b        bytes.Buffer
	limit    int
	overflow bool
	kill     func()
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if b.b.Len()+n > b.limit {
		remaining := b.limit - b.b.Len()
		_, _ = b.b.Write(p[:remaining])
		b.overflow = true
		b.kill()
		return n, nil
	}
	return b.b.Write(p)
}

// Linux process groups cover this fixed script and its ordinary descendants.
// A hostile process can create a new session; pod/cgroup teardown is still needed.
func command(ctx context.Context, path string, args []string, input []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C"}
	cmd.Stdin = bytes.NewReader(input)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	kill := func() {
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
	}
	cmd.Cancel = func() error { kill(); return nil }
	cmd.WaitDelay = time.Second
	stdout := &limitedBuffer{limit: outputLimit, kill: kill}
	stderr := &limitedBuffer{limit: 4096, kill: kill}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	err := cmd.Wait()
	kill()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if stdout.overflow || stderr.overflow {
		return nil, errors.New("adapter output limit exceeded")
	}
	if err != nil {
		return nil, errors.New("adapter process failed")
	}
	return stdout.b.Bytes(), nil
}
