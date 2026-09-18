package sourcecraft

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"repo-health/internal/domain"
	"sync/atomic"
	"testing"
	"time"
)

func TestPersonalCollectorAndPermissionRecheck(t *testing.T) {
	var denied atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer personal-pat" {
			t.Error("missing credential")
		}
		if denied.Load() {
			w.WriteHeader(403)
			return
		}
		switch r.URL.Path {
		case "/repos":
			fmt.Fprint(w, `{"repositories":[]}`)
		case "/repos/org/private":
			fmt.Fprint(w, `{"id":"1","slug":"private","visibility":"private","is_empty":true}`)
		case "/repos/org/private/issues":
			if r.URL.Query().Get("filter") != "" {
				t.Error("personal issues filtered as public")
			}
			fmt.Fprint(w, `{"issues":[{"visibility":"private","status":{"status_type":"initial"}}]}`)
		default:
			fmt.Fprint(w, `{}`)
		}
	}))
	defer server.Close()
	c := NewWithHTTP(server.URL, "personal-pat", server.Client(), 1)
	ctx := context.Background()
	if _, err := c.Collect(ctx, "org/private", time.Now()); !errors.Is(err, domain.ErrNotPublic) {
		t.Fatal("public collector accepted private data")
	}
	c.personal = true
	if err := c.ValidateToken(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.CheckAccess(ctx, "org/private"); err != nil {
		t.Fatal(err)
	}
	s, err := c.Collect(ctx, "org/private", time.Now())
	if err != nil || len(s.Issues) != 1 || s.Sources["appsec"].Status != "not_connected" {
		t.Fatalf("personal collection: %+v %v", s, err)
	}
	denied.Store(true)
	if c.CheckAccess(ctx, "org/private") == nil {
		t.Fatal("access recheck reused previous success")
	}
}
