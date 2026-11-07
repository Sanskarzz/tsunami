// Package httpserver owns listener shutdown and handler lifetime.
package httpserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"
)

type handlerGate struct {
	mu      sync.Mutex
	closing bool
	active  sync.WaitGroup
	handler http.Handler
}

func (g *handlerGate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	if g.closing {
		g.mu.Unlock()
		http.Error(w, "service is shutting down", 503)
		return
	}
	g.active.Add(1)
	g.mu.Unlock()
	defer g.active.Done()
	g.handler.ServeHTTP(w, r)
}

// Serve returns only after shutdown and admitted handlers exit. Callers may
// close their store afterwards. Forced process termination remains external.
func Serve(ctx context.Context, server *http.Server, listener net.Listener) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	handler := server.Handler
	if handler == nil {
		handler = http.DefaultServeMux
	}
	gate := &handlerGate{handler: handler}
	server.Handler = gate
	server.BaseContext = func(net.Listener) context.Context { return ctx }
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		<-ctx.Done()
		gate.mu.Lock()
		gate.closing = true
		gate.mu.Unlock()
		if q, ok := handler.(interface{ Quiesce() }); ok {
			q.Quiesce()
		}
		shutdown, done := context.WithTimeout(context.Background(), 5*time.Second)
		err := server.Shutdown(shutdown)
		done()
		if err != nil {
			_ = server.Close()
		}
		gate.active.Wait()
	}()
	err := server.Serve(listener)
	cancel()
	<-drained
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
