# livespot â€” AGENTS.md (Codex CLI Operating Contract)

This file defines non-negotiable operating rules for Codex CLI inside this repository.
It is authoritative for agent workflow, git discipline, validation gates, and stop conditions.
All behavior must remain consistent with the document hierarchy below.

---

## 0) Environment
- OS: Windows 11
- Repo root: `C:\go\livespot`
- Shell: PowerShell (preferred) or Windows Terminal
- Language:
  - All `.md` files MUST be technical English.
  - Code comments SHOULD be technical English.

---

## 1) Document hierarchy (conflicts)
If any documents conflict, the higher authority wins:

1) `00_SOURCE_OF_TRUTH.md`
2) `01_DECISION_CONTRACT.md`
3) `02_DATA_SNAPSHOT_SPEC.md`
4) `03_STRATEGY.md`
5) `04_RISK_ENGINE_RULES.md`
6) `05_EXECUTION_AND_FAILSAFE.md`
7) `06_AUDIT_RULES.md`
8) `07_SECURITY.md`
9) `08_SYSTEM_ARCHITECTURE.md`
10) `09_CODE_STRUCTURE.md`
11) `10_OPERATIONS_RULES.md`
12) `11_INVARIANTS_MAP.md` (DERIVED ONLY; must not create new rules)
13) `README.md` (MANUAL ONLY; must not create new rules)
14) `12_DECISIONS_LOG.md` (HISTORY; must record decisions when required)

---

## 2) Non-negotiable rules (fail-proof)
### 2.1 Fail-closed
If any of the following conditions occur, the system MUST stop or enter PAUSE:
- Missing or invalid configuration
- Audit sink unavailable or failing
- LIVE locks missing/invalid
- Contract validation failure
- Any I/O required for correctness fails (disk, DB, filesystem, permissions)
Never continue silently.

### 2.2 Audit-first
If audit sinks are not available and validated, the system MUST NOT operate.
Audit is required before any action that would change state.

### 2.3 No guessing / no invention
Do not invent or assume behavior not explicitly supported by:
- the doc hierarchy, or
- existing code (if already present).
If a required detail is not defined, treat it as missing information (see 2.4).

### 2.4 Missing information protocol (BLOCKER)
If critical information is undefined and blocks deterministic implementation:
1) STOP immediately.
2) Output a section titled: `MISSING_INFORMATION (BLOCKER)` including:
   - What is missing (precise)
   - Why it blocks determinism or safety
   - Where to obtain it (exact doc path or code path)
3) Do NOT implement around the missing piece with ad-hoc defaults.
4) Only add an entry to `12_DECISIONS_LOG.md` if the documents explicitly allow choosing a conservative default.
   - The entry MUST include: date, decision, reason, impact, affected files.

### 2.5 Security and secrets
- Never print secrets, API keys, tokens, private keys, or full environment dumps.
- Redact sensitive values in logs and error messages.
- Prefer placeholders like `REDACTED` and `EXAMPLE_ONLY`.

---

## 3) Workflow (strict, repeatable)
Codex MUST operate in stages. For each stage:

1) PLAN
   - List the exact files to create/modify (with paths).
   - State purpose for each change.
   - Identify any doc references that justify the change.

2) IMPLEMENT
   - Minimal conservative code only.
   - Follow existing architecture and folder structure.
   - Do not implement future stages early.

3) VALIDATE (mandatory)
   Run and show outputs for:
   - `go test ./...`
   - `go vet ./...`
   If any fail:
   - Fix until both PASS.
   - Do not commit or push while failing.

4) CHECKLIST
   - Print a PASS/FAIL checklist (see section 6).

5) GIT (mandatory discipline)
   - Exactly one commit and one push per stage (see section 4).

6) STOP
   - After a stage is complete and pushed, print: `STAGE XX COMPLETE` and STOP.
   - Do not start the next stage automatically.

---

## 4) Git discipline (one stage = one branch = one commit = one push)
### 4.1 Branch rules
- One stage per branch.
- Branch naming:
  - `feat/step-01-bootstrap`
  - `feat/step-02-config`
  - `feat/step-03-contracts-hashing`
  - etc.

### 4.2 Commit rules
- Exactly ONE commit per stage branch (remote history must show exactly one).
- Commit message format:
  - `feat(step-XX): <concise summary>`
  - `chore(step-XX): <concise summary>` when appropriate

Prohibited:
- WIP commits
- â€œfixâ€, â€œtempâ€, â€œupdateâ€, â€œmiscâ€ messages
- multiple commits per stage

### 4.3 Amend policy (allowed before push)
If additional changes are needed after the commit but before the push:
- Use amend (do not create extra commits):
  - `git add -A`
  - `git commit --amend --no-edit`

### 4.4 Push rules
- Exactly ONE push per stage branch.
- Never push before validations PASS.
- Always show push output.

### 4.5 Standard git command sequence (per stage)
Codex MUST run these commands (and show outputs), adapting stage number and branch name:

1) Start:
- `git checkout main`
- `git pull --ff-only`
- `git checkout -b feat/step-XX-<slug>`

2) Work + validate:
- `go test ./...`
- `go vet ./...`

3) Commit (single):
- `git status`
- `git add -A`
- `git commit -m "feat(step-XX): <summary>"`

4) If adjustments required before push:
- `git add -A`
- `git commit --amend --no-edit`
- Re-run validations.

5) Push once:
- `git push -u origin feat/step-XX-<slug>`

---

## 5) Stage plan (must execute strictly in order)
Stage 01 â€” Repo bootstrap + minimal build
- Create Go module and repository skeleton consistent with `09_CODE_STRUCTURE.md`.
- Provide minimal `cmd/*` entrypoints.
- Ensure `go test ./...` and `go vet ./...` PASS.

Stage 02 â€” Config + strict validation (fail-closed)
- Implement config loading and strict validation.
- Invalid config must stop/PAUSE with explicit reason.
- Add unit tests for validation.

Stage 03 â€” Contracts + deterministic hashing
- Implement core structs and validations for:
  - decision, snapshot, audit event, risk verdict, AI gate result (only as specified).
- Implement deterministic serialization (canonical JSON) and SHA-256 hashing as specified.
- Add determinism tests.

Stage 04 â€” Audit sinks (SQLite + JSONL) + correlation IDs
- Implement audit events, correlation identifiers, and sinks.
- Audit must be required; sink failure must stop/PAUSE.
- Add tests for write/read (SQLite) and JSONL line validity.

Stage 05 â€” LIVE locks + redaction
- Implement strict LIVE safety locks as specified by docs.
- Implement strict log redaction.
- Add tests.

Stage 06 â€” Pipeline skeleton + stages + observability
- Implement stage machine and stage reporting.
- Provide dry-run mode producing audit events.
- No real execution/trading unless explicitly specified and unlocked.


Stage 07 - Doctor checks + troubleshooting docs
- Implement doctor command to verify environment and readiness:
  - config, locks, audit sinks, filesystem permissions, DB availability.
- Add troubleshooting section to README.md (technical English).

Stage 08 - Decision + AI gate (no orders)
- Implement live data ingestion (REST + WS) for snapshots and selection inputs.
- Implement decision pipeline with strategy propose, AI gate evaluation, and risk verdict.
- Persist snapshots and decisions (canonical JSON) without executing orders.
- Reduce console noise with throttled summaries.

---

## 6) Required checklists (print every stage)
### 6.1 Pre-commit checklist
- [ ] Stage scope implemented only (no future-stage work)
- [ ] Fail-closed enforced in new code paths
- [ ] Audit-first enforced (if stage touches audit-related behavior)
- [ ] No secrets printed (redaction verified when relevant)
- [ ] `go test ./...` PASS
- [ ] `go vet ./...` PASS
- [ ] `git status` clean after commit/amend

### 6.2 Pre-push checklist
- [ ] Exactly one commit exists on the stage branch
- [ ] Commit message matches required format
- [ ] Validations re-run after last code change and PASS
- [ ] Push will be executed exactly once

### 6.3 Post-push checklist
- [ ] Push output shown and indicates success
- [ ] Stage completion marker printed: `STAGE XX COMPLETE`
- [ ] STOP (do not proceed to next stage)

### 6.4 Decision + AI (no orders) checklist
- [ ] Live startup checks display sqlite migrate, OpenAI call, Binance REST, Binance WS, and USDT balance
- [ ] Universe scan, ranking, deep scan, and top-k selection run without error
- [ ] Strategy decision and AI gate evaluation complete (no order placement)
- [ ] Decisions persisted to SQLite with canonical JSON payload
- [ ] Summary logging throttled (no per-tick WS spam)

---

## 7) Output format (must be used)
For each stage, output in this exact order:

1) `PLAN`
2) `COMMAND LOG (with outputs)`
3) `VALIDATION RESULTS`
4) `CHECKLIST (PASS/FAIL)`
5) `GIT LOG (branch, commit hash, push output)`
6) `STAGE XX COMPLETE`

If blocked:
- Output `MISSING_INFORMATION (BLOCKER)` and STOP.

---


