# Changelog

All notable changes to this project will be documented in this file.

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
- `vectorpad tui` — interactive three-panel interface
- `vectorpad add` — quick-add ideas to stash from CLI
