07_SECURITY (SECURITY, SECRETS, AND SAFETY LOCKS)

GOAL
Prevent secret leakage and reduce operational risk in LIVE.
Security is fail-closed when there is uncertainty.

SCOPE
This document defines:
- secrets policy (what is allowed and how secrets are handled)
- mandatory redaction rules for any persistence (SQLite/JSONL/logs)
- LIVE safety locks and checklist requirements
- strict AI Gate schema requirements and local enforcement rules
- Git hygiene rules (never commit runtime/secret artifacts)
- local web panel security (bind + allowlist + audit)

NON-NEGOTIABLE PRINCIPLES
- Never persist secrets (keys, tokens, signatures, cookies, credentials).
- Redaction happens BEFORE any persistence (SQLite/JSONL/log/alerts).
- If redaction or validation is uncertain, persist only hashes + verdict + reasons + error_code.
- In MODE=LIVE with AI_DEC=2, AI Gate failures are FAIL-CLOSED (PAUSE + block the entry).

SECRETS (ENVIRONMENT VARIABLES)
Rules:
- Only these environment variables may exist for secrets:
  - BINANCE_API_KEY
  - BINANCE_API_SECRET
  - OPENAI_API_KEY
- Secrets must never be persisted in:
  - logs
  - JSONL
  - SQLite
  - snapshots
  - prompts
- .env must never be committed/versioned.

REDACTION (MANDATORY, BEFORE ANY PERSISTENCE)
Definitions:
- raw request payload: the unredacted payload that would be sent to AI Gate.
- redacted request payload: canonical JSON payload after redaction and volatile-field removal.
- raw model response body: the unredacted model output body (string/bytes) returned by the AI Gate.
- redacted model response: canonical JSON derived from the model output after redaction and volatile-field removal.

Rules:
- Any payload sent to the AI Gate MUST be redacted before any persistence (SQLite/JSONL/log).
- Any model response MUST be redacted before any persistence (SQLite/JSONL/log).
- Any header/token/signature MUST be removed before any persistence (including audit).
- ai_gate.log MUST be redacted; raw prompt/response must never be written.

Forbidden content in audit/log/alerts:
- environment variable names with values (BINANCE_API_KEY, BINANCE_API_SECRET, OPENAI_API_KEY)
- Authorization headers, X-MBX-APIKEY, cookies
- request signatures, querystring signatures, sensitive querystrings
- full URLs containing credentials
- full raw prompt/response body without redaction
- stack traces that include secrets, raw payloads, or credential-like substrings

When in doubt (privacy-first):
- Persist only:
  - input_hash
  - snapshot_hash
  - raw_hash
  - verdict
  - reasons
  - error_code
- Do not persist any redacted JSON payload.

HASHES (TAMPER-DETECTION ONLY)
Rules:
- input_hash:
  - SHA-256 hex of the canonical JSON request payload that is actually sent to AI Gate.
  - Canonical JSON must follow RFC 8785 (JSON Canonicalization Scheme).
- snapshot_hash:
  - SHA-256 hex of the canonical snapshot used for the decision payload.
- raw_hash:
  - SHA-256 hex of the raw model response body (unredacted).
  - Purpose: tamper detection only.
  - The raw model response body MUST NEVER be persisted in full.

SIZE LIMIT AND TRUNCATION (DETERMINISTIC)
Purpose:
- Avoid persisting large redacted JSON blobs and guarantee deterministic storage.

Config requirement:
- audit_redacted_json_max_bytes is the maximum allowed byte length (UTF-8) for any persisted redacted JSON string.
- This field MUST exist in internal\config\config.go and MUST be documented in 00_SOURCE_OF_TRUTH.md.

Truncation algorithm (deterministic):
1) Let len_bytes be the byte length of the UTF-8 encoded redacted JSON string.
2) If len_bytes <= audit_redacted_json_max_bytes: persist the full string.
3) Else: persist exactly the ASCII string "<TRUNCATED len_bytes={len_bytes}>" (no JSON content).

Rules:
- It is forbidden to persist partial JSON (do not cut/clip the JSON string).
- Never write the full raw payload.
- request_json_redacted and response_json_redacted may exist only as redacted canonical JSON.

VOLATILE FIELDS (MUST REMOVE BEFORE HASHING AND PERSISTENCE)
Definition:
- volatile fields are any fields that change between identical logical requests and would break stable hashing.

Examples (non-exhaustive, treat as volatile):
- timestamps and dates
- nonces
- sequence numbers
- temporary IDs
- random values

Rule:
- Redacted JSON persisted (request_json_redacted/response_json_redacted) MUST remove volatile fields.

LOCAL ALERTING (SECURITY AND PRIVACY)
Rules:
- local alerts (console, panel, sound/toast) MUST be redacted by default.
- alert text may contain only:
  - stage
  - event_type
  - reasons
  - correlation IDs: run_id, cycle_id, snapshot_id, decision_id, order_intent_id
  - safe numeric metrics (disk_free_bytes, sqlite_wal_bytes, audit_writer_queue_pct, etc.)
- it is forbidden to include:
  - headers, X-MBX-APIKEY, signatures, sensitive querystrings
  - non-redacted request/response payloads
  - raw model prompt/response content
- when an alert is raised, there MUST be a corresponding auditable ALERT_RAISED event.

CRITICAL FAILURES MUST ALERT (IN ADDITION TO FAIL-CLOSED)
Rules:
- If AI Gate is mandatory and there is a failure (timeout, invalid schema, network error), in addition to blocking/pausing per policy, emit ALERT_RAISED.
- If there is low disk, saturated DB writer queue, reconcile drift above limit, or DB not writable, emit ALERT_RAISED immediately.

LIVE SAFETY LOCKS (EXTERNAL LOCK + CHECKLIST)
Rules:
- MODE=LIVE requires an external lock:
  - scripts\live.ps1 requires the --live argument
  - optional: file var\LIVE.ok must exist
- Before operating LIVE, the checklist MUST pass:
  - MODE=LIVE
  - AI_DEC=2
  - DB OK and writable
  - filters loaded
  - clock OK
  - WS OK
  - initial reconcile OK

Failure behavior:
- If the checklist fails:
  - enter PAUSE (ReasonCode ENTER_PAUSE)
  - emit ALERT_RAISED (redacted details only)

Files:
- scripts\live.ps1 (must enforce the external lock and fail-closed)
- internal\app\live_checklist.go (validates checklist; on failure enters PAUSE and emits ALERT_RAISED)

AI GATE: MANDATORY SCHEMA (RETURN AND VALIDATION)
Rules:
- AI Gate return MUST be parsed only via strict schema validation.
- If schema is invalid, parsing fails, or required fields are missing:
  - the result is invalid and MUST be treated as an AI Gate failure.
- reasons must be non-empty and MUST be valid ReasonCode entries; unknown reasons invalidate the result.

Files:
- prompts\schemas\ai_gate_result.schema.json (schema)
- internal\domain\contracts\ai_gate.go (structure + validation: schema + invariants)

AI PAYLOAD: MINIMUM (NO SECRETS)
AI payload MUST always include (no secrets):
- estimated costs (fees/spread/slippage), edge_bps_expected, edge_score
- regime and microstructure (normal vs current spread, spread_delta, imbalance)
- ExitPlan (ATR TP/SL) and validations performed
- why Strategy wants to trade (reason codes and short explanations)
- correlation IDs and hashes:
  - snapshot_id, decision_id, input_hash, snapshot_hash

Rules:
- payload MUST be deterministic (canonical JSON) for stable hashing.
- payload MUST NOT include volatile fields outside the snapshot.

LOCAL ENFORCEMENT OF MODIFY (NON-NEGOTIABLE)
Rule:
- MODIFY may only reduce risk and/or aggressiveness.
- Every MODIFY suggestion MUST be validated locally.
- If it violates any rule: reject MODIFY and audit AIGATE_MODIFY_INVALID.

MODIFY may only:
- BLOCK
- reduce qty
- tighten stop / protection
- make entry less aggressive (require maker/limit, reduce churn)
- reduce or disable fallback

MODIFY must NEVER:
- increase qty
- loosen stop
- increase maximum risk
- change symbol or timeframe
- alter clientOrderId (IDs are owned by the system)
- remove protection
- introduce schema-unknown fields

Immutability rules for modified_decision:
- mode, symbol, side, intent, snapshot_id, cycle_id, constraints MUST NOT change.
- decision_id is recomputed by the system, never provided by the model.

Files:
- internal\engine\aigate\limits.go (validates/applies conservative MODIFY)
- internal\engine\aigate\client.go (call + strict parsing)
- prompts\ai_gate_system.txt (conservative rules + output format)

AI_DEC=2 IN LIVE AND FAIL-CLOSED
Rule:
- In MODE=LIVE, AI_DEC=2 is mandatory and implies fail-closed on the AI Gate.
- AI Gate failure in LIVE with AI_DEC=2 implies:
  - PAUSE the system (no new entries)
  - BLOCK the current decision (do not execute the entry)

Failures that trigger fail-closed:
- timeout
- network error
- empty response
- invalid schema
- parsing failed
- unknown reasons
- invalid MODIFY (attempted to increase risk)

Fail-closed semantics:
- stop generating new entries and enter PAUSE state.
- continue only management/reconcile operations per 05_EXECUTION_AND_FAILSAFE.md.
- emit SQLite + JSONL audit of AIGATE_CALL with error_code.

Minimum reason codes (AI Gate):
- AIGATE_MODIFY_INVALID
- AIGATE_PARSE_FAIL
- AIGATE_REASON_UNKNOWN
- AIGATE_SCHEMA_INVALID
- AIGATE_TIMEOUT

GIT POLICY (OFFLINE / LOCAL)
Goal:
- reproducible, auditable, clean repository
- no secrets in Git
- no runtime/build trash
- small and traceable commits

FORBIDDEN FILES (NEVER COMMIT)
A) Secrets and credentials
- .env
- files containing keys/tokens/cookies
- dumps containing authorization headers
- logs containing secrets (even accidentally)

B) Bot runtime artifacts
- var\ (entire folder)
- audit.sqlite and any generated DB
- *.log, *.jsonl from real execution
- runtime snapshots

C) Builds/binaries
- *.exe, *.dll, *.out
- bin\
- dist\
- build\
- out\

D) IDE/editor
- .vscode\
- .idea\
- *.swp\
- *.swo\
- .DS_Store

WHEN TO COMMIT
Mandatory commits:
1) When closing a unit of work (small complete feature).
2) When changing a contract/schema (Decision/Snapshot/Risk/AIGateResult).
3) When adding/changing migrations (new SQL file).
4) When adding/changing reasons.
5) When fixing a bug (ideally with a test or recorded reproduction).

Commit message pattern:
- <type>: <short summary>
Types:
- feat, fix, refactor, test, docs, chore, db

FILE SAFETY
Rules:
- Forbid var\ and any runtime files in Git.
- Ensure .gitignore covers all forbidden patterns.
- Rotate/retain logs and backup SQLite per 10_OPERATIONS_RULES.md.

INCIDENTS
- See 10_OPERATIONS_RULES.md for incident response runbook.

WEB PANEL (SECURITY)
Rules:
- Default bind: 127.0.0.1 only.
- Default port: defined in internal\config\config.go and documented in 08_SYSTEM_ARCHITECTURE.md.
- In MODE=LIVE the panel is strictly read-only and MUST NOT trigger any Binance REST/WS calls.
  - All data served by the panel MUST come from local persisted state (SQLite) and local process metrics.
  - The panel MUST NOT expose any endpoint that can submit/cancel/replace orders, change config, or mutate runtime state.

LIVE ALLOWLIST (MODE=LIVE)
The panel MUST allow only these endpoints (read-only):
- GET /dashboard
- GET /api/dashboard
- GET /api/orders?limit=N
- GET /api/stream

BLOCKED CLASSES (MODE=LIVE)
The panel MUST reject (fail-closed) any request that is not in the allowlist above, including:
- Any method other than GET (POST/PUT/PATCH/DELETE).
- Any route that could mutate execution, orders, config, or state.
- Any unknown route.

Rejection behavior (deterministic):
- For method != GET: return HTTP 403.
- For unknown paths: return HTTP 404.

Audit:
- Rejected requests in MODE=LIVE MUST emit an auditable WEBUI_REQUEST event (route, method, status).
- Allowed requests in MODE=LIVE MUST emit an auditable WEBUI_REQUEST event.
- WEBUI_REQUEST events MUST NOT be sampled or dropped.

LAN binding (non-loopback) is not supported:
- The panel MUST bind to 127.0.0.1 only.
- Any configuration that attempts to bind to 0.0.0.0 or any non-loopback interface MUST fail-closed at boot:
  - enter PAUSE
  - emit ALERT_RAISED (redacted details only)

MISSING INFORMATION (BLOCKER)
1) audit_redacted_json_max_bytes is referenced by this document but is not defined in 00_SOURCE_OF_TRUTH.md.
   Why this blocks determinism:
   - The truncation algorithm depends on an exact maximum byte limit.
   Resolution (required):
   - Add audit_redacted_json_max_bytes (int, bytes) to internal\config\config.go with a fixed default value.
   - Document the exact default value in 00_SOURCE_OF_TRUTH.md.
   - Record the decision (value + rationale) in 12_DECISIONS_LOG.md.
