package httpserver

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestShutdownWaitsForAdmittedHandlerCompletion(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	started, cancelSeen, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(cancelSeen)
		<-release
		w.WriteHeader(204)
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, &http.Server{Handler: handler}, listener) }()
	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		resp, err := http.Get("http://" + listener.Addr().String())
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("handler never started")
	}
	cancel()
	select {
	case <-cancelSeen:
	case <-time.After(time.Second):
		t.Fatal("request context not cancelled")
	}
	select {
	case <-done:
		t.Fatal("server returned while handler still owned application state")
	case <-time.After(30 * time.Millisecond):
	}
	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown did not complete")
	}
	<-clientDone
}
