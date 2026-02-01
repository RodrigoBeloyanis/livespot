12_DECISIONS_LOG (PROJECT DECISIONS LOG)

GOAL
Record important decisions and rule/contract changes.
It does not replace the rule files. It exists for history and traceability.

RECOMMENDED FORMAT (SIMPLE ADR)
- DATE (YYYY-MM-DD)
- TOPIC
- DECISION
- MOTIVATION
- IMPACT (affected files and tables)
- RISKS / MITIGATIONS
DATE: 2026-02-01
TOPIC: Go module path
DECISION: Use module path github.com/RodrigoBeloyanis/livespot
MOTIVATION: Align module path with the configured Git remote to avoid local-only module names.
IMPACT: go.mod
RISKS / MITIGATIONS: If the repository origin changes, update go.mod and record the change here.

DATE: 2026-02-01
TOPIC: Config defaults and validation baseline
DECISION: Implement config defaults per 00_SOURCE_OF_TRUTH and add audit_redacted_json_max_bytes=4096; set live_require_ok_file=false with live_ok_file_path=var/LIVE.ok.
MOTIVATION: Stage 02 requires strict config validation and 07_SECURITY mandates a fixed audit_redacted_json_max_bytes default.
IMPACT: internal\config\config.go, internal\config\validate.go, 00_SOURCE_OF_TRUTH.md
RISKS / MITIGATIONS: If the redacted JSON limit is too small or the LIVE.ok policy changes, adjust defaults and revalidate with a logged decision.

DATE: 2026-02-01
TOPIC: Audit sink defaults and correlation identifiers
DECISION: Add audit_sqlite_path=var/data/audit.sqlite, audit_jsonl_dir=var/logs, and audit_sqlite_busy_timeout_ms=5000; define correlation IDs using fixed prefixes and snapshot/decision/order_intent IDs derived from SHA-256 hashes, and run_id/cycle_id formatted as run_YYYYMMDD_HHMMSS / cyc_YYYYMMDD_HHMMSS.
MOTIVATION: Stage 04 requires SQLite+JSONL sinks and correlation identifiers; defaults are necessary for deterministic, fail-closed initialization.
IMPACT: internal\config\config.go, internal\config\validate.go, internal\audit\*.go, internal\infra\sqlite\*.go, internal\observability\correlation.go, 00_SOURCE_OF_TRUTH.md, migrations\0001_init.sql.
RISKS / MITIGATIONS: If ID formats or paths must change, update 00_SOURCE_OF_TRUTH.md and record the decision; ensure readers accept prefix+hash IDs.

