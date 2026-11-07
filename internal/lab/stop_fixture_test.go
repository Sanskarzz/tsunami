package lab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStopFixtureCountsTargetCancellationSeparately(t *testing.T) {
	keys := map[string]string{"owner": strings.Repeat("a", 32), "peer": strings.Repeat("b", 32)}
	fixture, err := NewStopFixture(keys)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(fixture)
	defer server.Close()
	stats := func(id string) StopStats {
		t.Helper()
		response, err := server.Client().Get(server.URL + "/cases/" + id + "/stats")
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		var got StopStats
		if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		return got
	}
	wait := func(predicate func() bool) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if predicate() {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("target observation did not arrive")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/cases/run/objects/owner", nil)
	request.Header.Set("Authorization", "Bearer "+keys["peer"])
	done := make(chan struct{})
	go func() {
		defer close(done)
		response, err := server.Client().Do(request)
		if err == nil {
			response.Body.Close()
		}
	}()
	wait(func() bool { return stats("run").Active == 1 })
	before := stats("run")
	if before.Requests != 1 || before.ProbeStarted != 1 || before.ProbeCancelled != 0 || before.ProbeCompleted != 0 {
		t.Fatal("fixture confused an active request with cancellation")
	}
	cancel()
	<-done
	wait(func() bool { return stats("run").Active == 0 })
	after := stats("run")
	if after.ProbeCancelled != 1 || after.ProbeCompleted != 0 || after.ProbeCancelledAt == nil || stats("restart").Requests != 0 {
		t.Fatal("cancellation evidence is missing or leaked between cases")
	}
}
