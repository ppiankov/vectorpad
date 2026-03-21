# Changelog

All notable changes to this project will be documented in this file.

## [0.22.0] - 2026-03-21

### Added
- Instant precedent recognition: 300ms debounced trigram search in risk panel while typing
- `InstantPrecedents()` client method for GET /v1/precedents
- Prediction debt health display in risk panel: debt ratio and band (healthy/accumulating/critical)
- `GetPredictionDebt()` client method for GET /v1/health/prediction-debt
- Live spar viewer: `vectorpad submit --to vectorcourt --live` streams deliberation rounds via SSE
- SSE stream client (`internal/vectorcourt/stream.go`) for GET /v1/submissions/{id}/stream
- ANSI spar renderer (`internal/tui/spar.go`) — plain terminal output, works over SSH
- Editor line indicator: `L3/18` in dashboard when text overflows visible area
- Adaptive classified view: collapses to 2 lines for long text, hidden for very long text

## [0.21.0] - 2026-03-19

### Changed
- Rebrand Oracul to VectorCourt: package `internal/oracul` renamed to `internal/vectorcourt`
- Auth header: `X-Oracul-Key` replaced by `X-VC-Key`
- Default endpoint: `https://oracul.app` replaced by `https://vectorcourt.com`
- Config keys: `oracul.api_key` / `oracul.endpoint` replaced by `vectorcourt.api_key` / `vectorcourt.endpoint`
- CLI flags: `--to oracul` / `--format oracul` replaced by `--to vectorcourt` / `--format vectorcourt`
- TUI: launch target "Oracul Council" renamed to "VectorCourt"
- Backward-compatible config migration: existing `oracul` config keys auto-migrate on load

## [0.20.0] - 2026-03-10

### Added
- `vectorpad outcome <claim-id> --result success|failure|partial --note "..."`: report decision outcomes to Oracul
- `ReportOutcome()` client method for POST /v1/cases/:id/outcome
- Outcome auto-annotates matching flight log record
- Shell completions updated with `outcome` subcommand

## [0.19.0] - 2026-03-10

### Added
- `vectorpad stash diff <id1> <id2>`: compare two verdict entries field-by-field
- Verdict diff logic in `internal/stash/diff.go`: JSON parsing, field comparison, rendered output
- `lookupStashItem` helper: find items by ID, claim ID, or prefix match

## [0.18.0] - 2026-03-10

### Added
- `vp` short alias: same binary built as `vp` via GoReleaser, included in Homebrew install and release tarballs

## [0.17.0] - 2026-03-10

### Added
- Oracul metadata in flight recorder: tier, filing quality, preflight result, warnings, precedent count
- `vectorpad log --stats` shows Oracul aggregates: submits, avg filing quality, rejection rate, top warnings
- `UpdateOracul()` method on Recorder for attaching async verdict data to flight records

## [0.16.0] - 2026-03-10

### Added
- Async Oracul submit: deliberation runs in background, TUI remains interactive
- Elapsed timer display in editor panel during deliberation
- Context cancellation on TUI quit cancels in-progress deliberation
- Rejects duplicate submissions while deliberation is active

## [0.15.0] - 2026-03-10

### Added
- Precedent search: live similar-case lookup in TUI risk panel after 3s idle
- `vectorpad precedent` CLI subcommand with `--limit` and `--json` flags
- PRECEDENTS section in risk panel: similarity scores, outcome dots, ref class summary
- `SearchPrecedents()` client method for GET /v1/precedents/search
- Debounced async precedent search with independent 3s tick track

## [0.14.0] - 2026-03-10

### Added
- Verdict history in stash: Oracul verdicts auto-stashed after successful submit
- Distinct `◆` symbol for verdict entries in TUI stash panel and CLI list
- `SourceVerdict` and `ItemTypeVerdict` types for verdict classification
- `stash list --type verdict` filters to verdict entries only
- `stash show <id>` renders full verdict JSON for verdict entries

## [0.13.0] - 2026-03-10

### Added
- Live preflight readiness indicator: READY/WARN/BLOCKED updates in risk panel after 2s idle
- Debounced async preflight check with Bubbletea tick/cmd pattern
- Cached results until text changes — no redundant API calls

## [0.12.0] - 2026-03-10

### Added
- Oracul Council as launch target 6 in ctrl+l overlay: classify, preflight gate, submit, verdict summary
- Target only visible when API key is configured; hidden otherwise

## [0.11.0] - 2026-03-10

### Added
- Oracul account status in TUI risk panel: tier, usage bar, reset time (only when API key configured)
- Account status refreshes on startup and after each launch
- Silent degradation: section hidden when no key or fetch fails

## [0.10.0] - 2026-03-10

### Added
- Oracul integration: `vectorpad submit --to oracul` sends classified cases for deliberation
- Oracul export: `vectorpad export --format oracul` emits CaseFiling JSON offline
- Config system: `vectorpad config set/get` for persistent settings (API key, endpoint)
- Preflight gate: validates cases with Oracul before submission, blocks rejected filings
- Sentence-to-CaseFiling mapper: DECISION, CONSTRAINT, TENTATIVE, SPECULATION, EXPLANATION tags map to filing fields
- Shell completions updated for config, submit, export commands

## [0.9.0] - 2026-03-08

### Added
- ContextSpectre feedback loop: risk panel shows session health grade, context pressure, turns remaining, model, and cost
- Decision economics display: actual CPD/TTC/CDR from contextspectre stats with per-epoch breakdown
- Feedback refreshes on TUI startup and after each launch event
- Graceful degradation: feedback sections hidden when contextspectre is not installed

## [0.8.0] - 2026-03-08

### Added
- Claim registry: SQLite stash with embedding similarity (`vectorpad stash`)
- SQLite backend replaces JSON for stash persistence (auto-migrates existing JSON)
- Ollama embeddings via nomic-embed-text for semantic similarity search
- Cosine similarity with thresholds: near-duplicate (>0.90), same idea (0.80-0.90), related (0.65-0.80)
- Claim IDs assigned at stash time for tracking idea evolution
- CLI: `stash add`, `stash list`, `stash compare`, `stash show`, `stash cluster`, `stash evolve`, `stash reindex`
- Graceful degradation: works without Ollama (no embeddings, Jaccard fallback for clustering)

## [0.7.0] - 2026-03-08

### Added
- Vector decomposition (`ctrl+b`): split high-blast-radius vectors into focused sub-vectors by target groups
- Sub-vectors include shared preamble (constraints, context) plus group-specific sentences
- Decompose suggestion in risk panel when 3+ distinct targets detected
- Decompose output in CLI pipe mode (`DECOMPOSE` section)

## [0.6.0] - 2026-03-08

### Added
- Pressure heat map: per-sentence risk scoring based on lock policy, classification tag, vague verb presence, and brevity
- Three pressure levels (LOW/MED/HIGH) with visual bar in risk panel
- Pressure output in CLI pipe mode (`PRESSURE` section)

## [0.4.1] - 2026-03-08

### Fixed
- Risk panel now displays detected scope markers ("all repos") and target count instead of showing `repos: 0 files: 0`

## [0.5.0] - 2026-03-08

### Added
- Scope declaration (`ctrl+d`): declare blast radius before writing, cross-reference against text
- Scope mismatches surface in risk panel (scope vs constraints, operation vs preservation, target mentions)

## [0.4.0] - 2026-03-08

### Added
- Negative space detection: 6 gap types flag missing constraint classes (preservation, success criteria, review, rollback, scope boundary, identity)
- Drift timeline in TUI: risk panel shows meaning drift as you edit (strengthened/weakened/flipped/added/removed)
- Flight recorder: append-only JSONL log of launched vectors with metrics, gaps, and outcome annotation (`vectorpad log`)
- Constraint pinning: risk panel warns when CONSTRAINT sentences are removed during editing

## [0.3.1] - 2026-03-08

### Fixed
- Pastewatch detection now checks both `pastewatch-cli` (Homebrew) and `pastewatch` binary names

## [0.3.0] - 2026-03-08

### Added
- Launch target picker (`ctrl+l`): clipboard, Claude for Mac, ChatGPT for Mac, Claude Code CLI, file
- Essence extraction (`ctrl+e`): collapse stash stack into launchable summary
- Stash uniqueness symbols: `●` bright (novel), `○` dim (overlaps), `◌` faint (duplicate)
- Shell completions for bash, zsh, and fish (`vectorpad completion <shell>`)
- Full README with badges, architecture, philosophy, bond, roadmap

## [0.2.1] - 2026-03-08

### Fixed
- `vectorpad` now launches TUI by default (no need for `vectorpad tui`)
- Added keybinding hint bar at bottom of editor panel
- Clipboard test skips on Linux CI when xsel/xclip unavailable

## [0.2.0] - 2026-03-07

### Added
- File attachment pipeline: detect paths, classify by extension, preview, serialize for copy-out
- Image protocol detection (iTerm2, kitty)
- Object card rendering in editor panel
- Paste interception: dragged files become attachment objects, not inline text
- Excerpt configuration per attachment
- Three serialization modes: path-only, excerpt, evidence

## [0.1.0] - 2026-03-07

### Added
- Three-panel TUI: stash roster, vector editor, risk panel
- Sentence classifier with 6 tags (CONSTRAINT, DECISION, TENTATIVE, QUESTION, SPECULATION, EXPLANATION)
- Meaning drift detection on 6 axes
- Pre-flight metrics: token weight, CPD, TTC, CDR projections
- Ambiguity detection with blast radius, brevity ratio, vague verb flagging
- Nudge protocol for ambiguous vectors
- Stash persistence with Jaccard similarity clustering
- Capability detection for pastewatch and contextspectre
- Pastewatch integration: scan outbound payload before clipboard copy
- CLI pipeline: stdin classification + preflight + ambiguity analysis
- `vectorpad tui` - interactive three-panel interface
- `vectorpad add` - quick-add ideas to stash from CLI
