package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"repo-health/internal/app"
	"repo-health/internal/demo"
	"repo-health/internal/domain"
	"repo-health/internal/storage"
	"strings"
	"testing"
	"time"
)

func TestAnalysisEndToEndAndRequestBoundaries(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "health.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := app.New(store, demo.Collector{}, demo.Collector{}, log)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- service.Run(ctx) }()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	server := httptest.NewServer(New(service, log, false))
	defer server.Close()
	request := func(path, body, origin string) *http.Response {
		req, _ := http.NewRequest("POST", server.URL+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		res, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	res := request("/api/v1/analyses", `{"ref":"demo/atlas","demo":true}`, "")
	if res.StatusCode != 202 {
		t.Fatal("enqueue", res.StatusCode)
	}
	var job domain.Job
	if err = json.NewDecoder(res.Body).Decode(&job); err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	deadline := time.Now().Add(10 * time.Second)
	for {
		j, err := service.Job(ctx, job.ID)
		if err != nil {
			t.Fatal(err)
		}
		if j.Status == "completed" {
			break
		}
		if j.Status == "failed" || time.Now().After(deadline) {
			t.Fatalf("job failed/stuck: %+v", j)
		}
		time.Sleep(10 * time.Millisecond)
	}
	res, err = server.Client().Get(server.URL + "/api/v1/leaderboard?demo=true")
	if err != nil {
		t.Fatal(err)
	}
	var listing struct {
		Items []domain.Project `json:"items"`
	}
	json.NewDecoder(res.Body).Decode(&listing)
	res.Body.Close()
	if len(listing.Items) != 1 || listing.Items[0].Report.Score == nil {
		t.Fatal("completed report missing from demo ranking")
	}
	for _, tc := range []struct {
		body, origin string
		status       int
	}{
		{`{"ref":"https://evil.test/a/b"}`, "", 400},
		{`{"ref":"demo/atlas","demo":true}`, "https://evil.test", 403},
		{`{"ref":"demo/atlas","admin":true}`, "", 400},
		{`{"ref":"demo/atlas"} {}`, "", 400},
		{`{"ref":"demo/not-an-example","demo":true}`, "", 400},
	} {
		res = request("/api/v1/analyses", tc.body, tc.origin)
		res.Body.Close()
		if res.StatusCode != tc.status {
			t.Errorf("request %s got %d want %d", tc.body, res.StatusCode, tc.status)
		}
	}
	res, err = server.Client().Get(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 || res.Header.Get("Content-Security-Policy") == "" {
		t.Fatal("embedded UI or headers missing")
	}
}
