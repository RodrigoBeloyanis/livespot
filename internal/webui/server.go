package webui

import (
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"
)

type Server struct {
	cfg    config.Config
	db     *sql.DB
	writer *audit.Writer
	now    func() time.Time
	start  time.Time
	mux    *http.ServeMux
	server *http.Server
}

func NewServer(cfg config.Config, db *sql.DB, writer *audit.Writer, now func() time.Time) (*Server, error) {
	if db == nil {
		return nil, fmt.Errorf("webui db missing")
	}
	if writer == nil {
		return nil, fmt.Errorf("webui audit writer missing")
	}
	if now == nil {
		now = time.Now
	}
	mux := http.NewServeMux()
	s := &Server{
		cfg:    cfg,
		db:     db,
		writer: writer,
		now:    now,
		start:  now(),
		mux:    mux,
	}
	s.registerRoutes()
	s.server = &http.Server{
		Handler:      s.mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return s, nil
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.cfg.WebuiPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	go func() {
		_ = s.server.Serve(ln)
	}()
	return nil
}

func (s *Server) Close() error {
	if s.server == nil {
		return nil
	}
	return s.server.Close()
}

func (s *Server) Handler() http.Handler {
	return s.mux
}
