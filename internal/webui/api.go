package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	auditdomain "github.com/RodrigoBeloyanis/livespot/internal/domain/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"
)

const (
	defaultOrdersLimit = 100
	maxOrdersLimit     = 500
)

type DashboardSnapshot struct {
	TsMs              int64               `json:"ts_ms"`
	Mode              string              `json:"mode"`
	SysMode           string              `json:"sys_mode"`
	SysModeSinceTsMs  int64               `json:"sys_mode_since_ts_ms"`
	ActiveReasonCodes []string            `json:"active_reason_codes"`
	UptimeS           int64               `json:"uptime_s"`
	RunID             string              `json:"run_id"`
	CycleID           string              `json:"cycle_id"`
	DecisionID        string              `json:"decision_id"`
	Stage             string              `json:"stage"`
	StageSinceTsMs    int64               `json:"stage_since_ts_ms"`
	AlertsActive      []AlertAggregateRow `json:"alerts_active"`
	StageHistory      []StageHistoryRow   `json:"stage_history"`
	Health            HealthSnapshot      `json:"health"`
	Intents           IntentsSnapshot     `json:"intents"`
	Reconcile         ReconcileSnapshot   `json:"reconcile"`
	Market            MarketSnapshot      `json:"market"`
	Risk              RiskSnapshot        `json:"risk"`
	AIGate            AIGateSnapshot      `json:"aigate"`
	TopK              TopKSnapshot        `json:"topk"`
}

type AlertAggregateRow struct {
	Severity      string `json:"severity"`
	ReasonCode    string `json:"reason_code"`
	Count         int64  `json:"count"`
	FirstTsMs     int64  `json:"first_ts_ms"`
	LastTsMs      int64  `json:"last_ts_ms"`
	Stage         string `json:"stage"`
	CycleID       string `json:"cycle_id"`
	DecisionID    string `json:"decision_id"`
	OrderIntentID string `json:"order_intent_id"`
}

type StageHistoryRow struct {
	TsMs       int64  `json:"ts_ms"`
	Stage      string `json:"stage"`
	CycleID    string `json:"cycle_id"`
	DecisionID string `json:"decision_id"`
	Symbol     string `json:"symbol"`
	Summary    string `json:"summary"`
}

type HealthSnapshot struct {
	DiskFreeBytes             int64  `json:"disk_free_bytes"`
	SqliteBytes               int64  `json:"sqlite_bytes"`
	WalBytes                  int64  `json:"wal_bytes"`
	ProcessRSSBytes           int64  `json:"process_rss_bytes"`
	ProcessCPUPctX10000       int32  `json:"process_cpu_pct_x10000"`
	AuditWriterQueuePctX10000 int32  `json:"audit_writer_queue_pct_x10000"`
	AuditWriterQueueLen       int64  `json:"audit_writer_queue_len"`
	AuditWriterLagMs          int64  `json:"audit_writer_lag_ms"`
	AuditWriterPressure       string `json:"audit_writer_pressure"`
	EventsPerMin              int64  `json:"events_per_min"`
	DropsPerMin               int64  `json:"drops_per_min"`
	LastCycleTsMs             int64  `json:"last_cycle_ts_ms"`
	LastWriterCommitTsMs      int64  `json:"last_writer_commit_ts_ms"`
}

type IntentsSnapshot struct {
	PendingByState map[string]int64 `json:"pending_by_state"`
	Recent         []IntentRow      `json:"recent"`
}

type IntentRow struct {
	OrderIntentID string `json:"order_intent_id"`
	Symbol        string `json:"symbol"`
	IntentKind    string `json:"intent_kind"`
	State         string `json:"state"`
	LastTsMs      int64  `json:"last_ts_ms"`
	CycleID       string `json:"cycle_id"`
	DecisionID    string `json:"decision_id"`
	Summary       string `json:"summary"`
}

type ReconcileSnapshot struct {
	LastReconcileRestTsMs  int64              `json:"last_reconcile_rest_ts_ms"`
	LastReconcileRestDurMs int64              `json:"last_reconcile_rest_dur_ms"`
	DriftScoreX10000       int32              `json:"drift_score_x10000"`
	DriftLabel             string             `json:"drift_label"`
	RecentDiffs            []ReconcileDiffRow `json:"recent_diffs"`
}

type ReconcileDiffRow struct {
	TsMs       int64  `json:"ts_ms"`
	Symbol     string `json:"symbol"`
	CycleID    string `json:"cycle_id"`
	DecisionID string `json:"decision_id"`
	Summary    string `json:"summary"`
}

type MarketSnapshot struct {
	Symbols []MarketSymbolRow `json:"symbols"`
}

type MarketSymbolRow struct {
	Symbol             string           `json:"symbol"`
	Regime             string           `json:"regime"`
	TrendScoreX10000   int32            `json:"trend_score_x10000"`
	RangeScoreX10000   int32            `json:"range_score_x10000"`
	SpreadCurrentBps   int32            `json:"spread_current_bps"`
	SpreadP50Bps       int32            `json:"spread_p50_bps"`
	SpreadP90Bps       int32            `json:"spread_p90_bps"`
	DeltaSpreadP90_10s int32            `json:"delta_spread_bps_p90_10s"`
	ImbalanceX10000    int32            `json:"imbalance_x10000"`
	EdgeBpsExpected    int32            `json:"edge_bps_expected"`
	CostsBps           map[string]int32 `json:"costs_bps"`
	BlockedReasonCode  string           `json:"blocked_reason_code"`
	SignalFlags        []string         `json:"signal_flags"`
}

type RiskSnapshot struct {
	ExposureTotalQuote           string              `json:"exposure_total_quote"`
	ExposureBySymbolQuote        []map[string]string `json:"exposure_by_symbol_quote"`
	LockedQuote                  string              `json:"locked_quote"`
	FreeQuote                    string              `json:"free_quote"`
	DailyPnLRealizedQuote        string              `json:"daily_pnl_realized_quote"`
	DailyPnLUnrealizedQuote      string              `json:"daily_pnl_unrealized_quote"`
	DailyLossLimitRemainingQuote string              `json:"daily_loss_limit_remaining_quote"`
	DrawdownQuote                string              `json:"drawdown_quote"`
	TradesToday                  int64               `json:"trades_today"`
	Cooldowns                    []map[string]any    `json:"cooldowns"`
}

type AIGateSnapshot struct {
	LastCallTsMs  int64    `json:"last_call_ts_ms"`
	LatencyMs     int      `json:"latency_ms"`
	Model         string   `json:"model"`
	Verdict       string   `json:"verdict"`
	ReasonCodes   []string `json:"reason_codes"`
	InputHash     string   `json:"input_hash"`
	SnapshotHash  string   `json:"snapshot_hash"`
	RawHash       string   `json:"raw_hash"`
	ModifyApplied bool     `json:"modify_applied"`
	ErrorKind     string   `json:"error_kind"`
}

type TopKSnapshot struct {
	CycleID               string           `json:"cycle_id"`
	Items                 []TopKItemRow    `json:"items"`
	MaxPairwiseCorrX10000 int32            `json:"max_pairwise_corr_x10000"`
	PairsOverLimit        []map[string]any `json:"pairs_over_limit"`
}

type TopKItemRow struct {
	Symbol            string             `json:"symbol"`
	ScoreX10000       int32              `json:"score_x10000"`
	FeaturesTop       []map[string]int32 `json:"features_top"`
	ChurnGuardApplied bool               `json:"churn_guard_applied"`
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/dashboard", s.wrap(s.dashboardHandler))
	s.mux.HandleFunc("/api/dashboard", s.wrap(s.dashboardAPIHandler))
	s.mux.HandleFunc("/api/orders", s.wrap(s.ordersHandler))
	s.mux.HandleFunc("/api/stream", s.wrap(s.streamHandler))
	s.mux.HandleFunc("/", s.wrap(s.notFoundHandler))
}

func (s *Server) wrap(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			s.writeAudit(r, http.StatusForbidden)
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if s.cfg.Mode == "LIVE" && !isAllowedPath(r.URL.Path) {
			s.writeAudit(r, http.StatusNotFound)
			http.NotFound(w, r)
			return
		}
		fn(w, r)
	}
}

func isAllowedPath(path string) bool {
	switch path {
	case "/dashboard", "/api/dashboard", "/api/orders", "/api/stream", "/":
		return true
	default:
		return false
	}
}

func (s *Server) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	s.writeAudit(r, http.StatusOK)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(dashboardHTML))
}

func (s *Server) dashboardAPIHandler(w http.ResponseWriter, r *http.Request) {
	snapshot, err := s.buildDashboardSnapshot(r.Context())
	if err != nil {
		s.writeAudit(r, http.StatusInternalServerError)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	s.writeAudit(r, http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(snapshot)
}

func (s *Server) ordersHandler(w http.ResponseWriter, r *http.Request) {
	limit := defaultOrdersLimit
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxOrdersLimit {
			s.writeAudit(r, http.StatusBadRequest)
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = n
	}
	rows, err := QueryRecentOrders(r.Context(), s.db, limit)
	if err != nil {
		s.writeAudit(r, http.StatusInternalServerError)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	s.writeAudit(r, http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(rows)
}

func (s *Server) streamHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		s.writeAudit(r, http.StatusInternalServerError)
		http.Error(w, "stream unsupported", http.StatusInternalServerError)
		return
	}
	s.writeAudit(r, http.StatusOK)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	snapshotTicker := time.NewTicker(time.Duration(s.cfg.WebuiStreamSnapshotIntervalMs) * time.Millisecond)
	pingTicker := time.NewTicker(15 * time.Second)
	defer snapshotTicker.Stop()
	defer pingTicker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-snapshotTicker.C:
			snapshot, err := s.buildDashboardSnapshot(ctx)
			if err != nil {
				continue
			}
			buf, _ := json.Marshal(snapshot)
			fmt.Fprintf(w, "event: snapshot\n")
			fmt.Fprintf(w, "data: %s\n\n", string(buf))
			flusher.Flush()
		case <-pingTicker.C:
			fmt.Fprintf(w, "event: ping\n")
			fmt.Fprintf(w, "data: {\"ts_ms\":%d}\n\n", s.now().UnixMilli())
			flusher.Flush()
		}
	}
}

func (s *Server) notFoundHandler(w http.ResponseWriter, r *http.Request) {
	s.writeAudit(r, http.StatusNotFound)
	http.NotFound(w, r)
}

func (s *Server) buildDashboardSnapshot(ctx context.Context) (DashboardSnapshot, error) {
	stage, err := QueryLatestStage(ctx, s.db)
	if err != nil {
		return DashboardSnapshot{}, err
	}
	alerts, _ := QueryAlerts(ctx, s.db, 20)
	history, _ := QueryStageHistory(ctx, s.db, 20)
	intents, _ := QueryIntents(ctx, s.db, s.cfg.WebuiIntentsRecentLimit)
	reconcile, _ := QueryReconcile(ctx, s.db, s.cfg.WebuiReconcileDiffsRecentLimit)
	aigate, _ := QueryAIGate(ctx, s.db)
	sysMode, sysSince, reasons := DeriveSysMode(alerts)

	health := BuildHealthSnapshot(s.cfg, s.start, s.writer)
	health.SqliteBytes, health.WalBytes, health.DiskFreeBytes = DiskStats()

	return DashboardSnapshot{
		TsMs:              s.now().UnixMilli(),
		Mode:              s.cfg.Mode,
		SysMode:           sysMode,
		SysModeSinceTsMs:  sysSince,
		ActiveReasonCodes: reasons,
		UptimeS:           int64(s.now().Sub(s.start).Seconds()),
		RunID:             stage.RunID,
		CycleID:           stage.CycleID,
		DecisionID:        stage.DecisionID,
		Stage:             stage.Stage,
		StageSinceTsMs:    stage.TsMs,
		AlertsActive:      alerts,
		StageHistory:      history,
		Health:            health,
		Intents:           intents,
		Reconcile:         reconcile,
		Market:            MarketSnapshot{Symbols: []MarketSymbolRow{}},
		Risk:              RiskSnapshot{ExposureTotalQuote: "0", ExposureBySymbolQuote: []map[string]string{}, LockedQuote: "0", FreeQuote: "0", DailyPnLRealizedQuote: "0", DailyPnLUnrealizedQuote: "0", DailyLossLimitRemainingQuote: "0", DrawdownQuote: "0", TradesToday: 0, Cooldowns: []map[string]any{}},
		AIGate:            aigate,
		TopK:              TopKSnapshot{Items: []TopKItemRow{}, PairsOverLimit: []map[string]any{}},
	}, nil
}

func (s *Server) writeAudit(r *http.Request, status int) {
	ctx := r.Context()
	runID := "unknown"
	cycleID := "unknown"
	stage := observability.STATE_UPDATE
	if s.db != nil {
		if latest, err := QueryLatestStage(ctx, s.db); err == nil {
			if latest.RunID != "" {
				runID = latest.RunID
			}
			if latest.CycleID != "" {
				cycleID = latest.CycleID
			}
			if latest.Stage != "" {
				stage = observability.StageName(latest.Stage)
			}
		}
	}
	record := audit.Record{
		Event: auditdomain.AuditEvent{
			TsMs:            s.now().UnixMilli(),
			RunID:           runID,
			CycleID:         cycleID,
			Mode:            s.cfg.Mode,
			Stage:           stage,
			EventType:       auditdomain.WEBUI_REQUEST,
			Reasons:         []reasoncodes.ReasonCode{},
			SnapshotID:      "",
			DecisionID:      "",
			OrderIntentID:   "",
			ExchangeTimeMs:  0,
			LocalReceivedMs: s.now().UnixMilli(),
		},
		Data: map[string]any{
			"method": r.Method,
			"path":   r.URL.Path,
			"status": status,
		},
	}
	if s.writer != nil {
		_ = s.writer.Write(record)
	}
}

const dashboardHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>LiveSpot Dashboard</title>
  <style>
    :root {
      --bg: #0f1c1f;
      --panel: #16262a;
      --accent: #e3b23c;
      --text: #e8f1f2;
      --muted: #8aa6a3;
    }
    body {
      margin: 0;
      font-family: "IBM Plex Sans", Arial, sans-serif;
      background: radial-gradient(circle at 20% 20%, #1f3a3a, var(--bg));
      color: var(--text);
    }
    header {
      padding: 16px 24px;
      border-bottom: 1px solid #213235;
      display: flex;
      align-items: center;
      justify-content: space-between;
    }
    .title {
      font-size: 20px;
      font-weight: 600;
      letter-spacing: 0.5px;
    }
    .status {
      background: var(--panel);
      padding: 6px 12px;
      border-radius: 999px;
      color: var(--accent);
      font-weight: 600;
    }
    main {
      padding: 16px 24px;
      display: grid;
      gap: 16px;
      grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
    }
    .card {
      background: var(--panel);
      border: 1px solid #213235;
      border-radius: 12px;
      padding: 12px;
      box-shadow: 0 10px 20px rgba(0,0,0,0.2);
    }
    .card h3 {
      margin: 0 0 8px 0;
      font-size: 14px;
      text-transform: uppercase;
      letter-spacing: 1px;
      color: var(--muted);
    }
    .value {
      font-size: 20px;
      font-weight: 600;
    }
    .list {
      list-style: none;
      padding: 0;
      margin: 0;
    }
    .list li {
      padding: 6px 0;
      border-bottom: 1px solid #1f2f32;
      font-size: 13px;
      color: var(--muted);
    }
    .list li:last-child {
      border-bottom: none;
    }
  </style>
</head>
<body>
  <header>
    <div class="title">LiveSpot WebUI</div>
    <div class="status" id="sysMode">NORMAL</div>
  </header>
  <main>
    <div class="card">
      <h3>Stage</h3>
      <div class="value" id="stage">BOOT</div>
      <div class="muted" id="cycle">cycle</div>
    </div>
    <div class="card">
      <h3>Alerts</h3>
      <ul class="list" id="alerts"></ul>
    </div>
    <div class="card">
      <h3>Intents</h3>
      <ul class="list" id="intents"></ul>
    </div>
    <div class="card">
      <h3>Reconcile</h3>
      <div class="value" id="reconcile">OK</div>
      <div class="muted" id="drift">drift</div>
    </div>
  </main>
  <script>
    const alertsEl = document.getElementById('alerts');
    const intentsEl = document.getElementById('intents');
    const sysModeEl = document.getElementById('sysMode');
    const stageEl = document.getElementById('stage');
    const cycleEl = document.getElementById('cycle');
    const reconcileEl = document.getElementById('reconcile');
    const driftEl = document.getElementById('drift');

    function render(snapshot) {
      sysModeEl.textContent = snapshot.sys_mode || 'NORMAL';
      stageEl.textContent = snapshot.stage || 'BOOT';
      cycleEl.textContent = snapshot.cycle_id || '';
      reconcileEl.textContent = snapshot.reconcile.drift_label || 'OK';
      driftEl.textContent = snapshot.reconcile.drift_score_x10000 || 0;
      alertsEl.innerHTML = '';
      (snapshot.alerts_active || []).forEach(a => {
        const li = document.createElement('li');
        li.textContent = a.reason_code + ' (' + a.count + ')';
        alertsEl.appendChild(li);
      });
      intentsEl.innerHTML = '';
      (snapshot.intents.recent || []).forEach(i => {
        const li = document.createElement('li');
        li.textContent = i.symbol + ' ' + i.state;
        intentsEl.appendChild(li);
      });
    }

    function connect() {
      const ev = new EventSource('/api/stream');
      ev.addEventListener('snapshot', e => {
        render(JSON.parse(e.data));
      });
      ev.onerror = () => {
        ev.close();
        setTimeout(poll, 1500);
      };
    }

    function poll() {
      fetch('/api/dashboard')
        .then(r => r.json())
        .then(render)
        .catch(() => {});
      setTimeout(poll, 3000);
    }

    connect();
  </script>
</body>
</html>`
