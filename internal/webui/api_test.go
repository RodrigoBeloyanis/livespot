package webui

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/sqlite"
)

func TestWebUIAllowlistLive(t *testing.T) {
	server := newTestServer(t)
	handler := server.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/unknown", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/dashboard", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	cfg := config.Default()
	cfg.WebuiPort = 0
	temp := t.TempDir()
	dbPath := filepath.Join(temp, "audit.sqlite")
	writer, err := audit.NewWriter(cfg, audit.WriterOptions{DBPath: dbPath, JSONLDir: temp, Now: time.Now})
	if err != nil {
		t.Fatalf("writer: %v", err)
	}
	t.Cleanup(func() {
		_ = writer.Close()
	})
	db, err := sqlite.Open(dbPath, cfg)
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	if err := sqlite.Migrate(db, time.Now()); err != nil {
		t.Fatalf("db migrate: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	server, err := NewServer(cfg, db, writer, time.Now)
	if err != nil {
		t.Fatalf("server: %v", err)
	}
	return server
}
