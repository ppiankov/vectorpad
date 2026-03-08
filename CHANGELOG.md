# Changelog

All notable changes to this project will be documented in this file.

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
