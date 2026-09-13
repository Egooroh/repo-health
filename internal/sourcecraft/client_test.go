package sourcecraft

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"repo-health/internal/domain"
	"repo-health/internal/scoring"
	"sync/atomic"
	"testing"
	"time"
)

func TestOfficialContractPaginationAndPartialAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("missing token")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/repos/org/repo":
			fmt.Fprint(w, `{"id":"123","slug":"repo","visibility":"public","description":"Test","default_branch":"main","language":{"name":"Go"}}`)
		case "/repos/org/repo/trees":
			if r.URL.Query().Get("path") == "docs" {
				fmt.Fprint(w, `{"trees":[{"name":"CONTRIBUTING.md","path":"docs/CONTRIBUTING.md","type":"file"}]}`)
			} else if r.URL.Query().Get("page_token") == "p2" {
				fmt.Fprint(w, `{"trees":[{"name":"LICENSE","type":"file"},{"name":"docs","type":"dir"}]}`)
			} else {
				fmt.Fprint(w, `{"trees":[{"name":"README.md","type":"file"}],"next_page_token":"p2"}`)
			}
		case "/repos/org/repo/issues":
			if r.URL.Query().Get("filter") != "visibility=public" {
				t.Error("public issue filter missing")
			}
			fmt.Fprint(w, `{"issues":[{"visibility":"public","status":{"status_type":"in_progress"},"updated_at":"2026-09-11T00:00:00Z"},{"visibility":"private","status":{"status_type":"initial"},"updated_at":"2020-01-01T00:00:00Z"}]}`)
		case "/repos/org/repo/pulls":
			fmt.Fprint(w, `{"pull_requests":[]}`)
		case "/repos/org/repo/cicd/runs":
			w.WriteHeader(403)
			fmt.Fprint(w, `{"message":"SECRET-RAW-ERROR"}`)
		default:
			t.Errorf("unexpected API path: %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	c := NewWithHTTP(server.URL, "test-token", server.Client(), 5)
	s, err := c.Collect(context.Background(), "org/repo", time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Files) != 3 || !s.Sources["files"].Available() || s.Sources["ci"].Available() {
		t.Fatalf("unexpected source data: %+v", s)
	}
	if len(s.Issues) != 1 {
		t.Fatal("private issue was collected")
	}
	r := scoring.Calculate(s)
	if r.Score == nil || *r.Score != 100 || r.Coverage != 60 {
		t.Fatalf("unexpected result: %+v", r)
	}
	if s.Sources["ci"].Detail == "SECRET-RAW-ERROR" {
		t.Fatal("raw upstream error exposed")
	}
}

func TestTruncationAndRetries(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/retry" && attempts.Add(1) < 3 {
			w.WriteHeader(503)
			return
		}
		fmt.Fprint(w, `{"issues":[],"next_page_token":"again"}`)
	}))
	defer server.Close()
	c := NewWithHTTP(server.URL, "", server.Client(), 2)
	_, state := c.list(context.Background(), "/issues", "issues", nil)
	if state.Status != "truncated" {
		t.Fatalf("expected truncation: %+v", state)
	}
	var dst map[string]any
	if err := c.get(context.Background(), "/retry", nil, &dst); err != nil || attempts.Load() != 3 {
		t.Fatalf("retry: %d %v", attempts.Load(), err)
	}
}

func TestPrivateAndInvalidMetadataRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"id":"secret","slug":"private","visibility":"private"}`)
	}))
	defer server.Close()
	_, err := NewWithHTTP(server.URL, "", server.Client(), 1).Collect(context.Background(), "org/private", time.Now())
	if !errors.Is(err, domain.ErrNotPublic) {
		t.Fatalf("private repo accepted: %v", err)
	}
}

func TestDeniedAccessDoesNotClaimPrivateVisibility(t *testing.T) {
	for _, status := range []int{401, 403} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
			defer server.Close()
			_, err := NewWithHTTP(server.URL, "", server.Client(), 1).Collect(context.Background(), "org/repo", time.Now())
			if !errors.Is(err, domain.ErrAccessDenied) || errors.Is(err, domain.ErrNotPublic) {
				t.Fatalf("wrong visibility inference: %v", err)
			}
		})
	}
}
