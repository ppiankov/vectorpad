# vectorpad

[![CI](https://github.com/ppiankov/vectorpad/actions/workflows/ci.yml/badge.svg)](https://github.com/ppiankov/vectorpad/actions/workflows/ci.yml)
[![Release](https://github.com/ppiankov/vectorpad/releases/latest/badge.svg)](https://github.com/ppiankov/vectorpad/releases/latest)
[![Go](https://img.shields.io/github/go-mod/go-version/ppiankov/vectorpad)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Vector launch pad for AI-assisted reasoning.

## What VectorPad is

A pre-flight staging area for operator intent. You paste your directive, VectorPad classifies every sentence, measures the pressure it will project into a context window, and shows you what's missing before you send it. Three panels: your stash of parked ideas on the left, the active vector in the center, risk analysis on the right.

Think of it as the pre-flight checklist before you push reasoning into a bounded context window. Structure the thought, see the pressure, then choose: launch, stash, or branch.

## Why this exists - the README Massacre

VectorPad was born from a simple directive:

    clean up READMEs for alignment

The operator meant:
- keep the project voice
- preserve "Why this exists" sections
- keep philosophy paragraphs
- standardize formatting and badges
- only fix structure

What the agent received was five words.

The cleanup touched 18 repositories and replaced detailed documentation with short templates. Architecture diagrams, usage examples, years of accumulated clarity - gone in one pass.

Nothing malicious happened. The operator simply transmitted only a fraction of their intent.

That's an [ambiguous vector](https://github.com/ppiankov/contextspectre/blob/main/docs/concepts.md#glossary) - operator intent compressed below safe execution resolution. Not a bad prompt. Not a bad model. A transmission failure: private clarity that didn't survive serialization to text.

VectorPad exists to catch that moment before execution. A smoke detector for operator intent - not a judge, not a blocker. Just enough friction to ask: did you say everything you meant?

![VectorPad TUI - sentence classification with locked constraints](assets/screen.png)

## What VectorPad is NOT

- **Not a prompt template engine.** No fill-in-the-blank scaffolding. You write raw thought; VectorPad shows you what it looks like under classification
- **Not an LLM wrapper or chat interface.** VectorPad never calls a model. It stages what you send, not where you send it
- **Not a text editor.** It's a staging area with intentional friction on input and zero friction on output. You don't compose here - you inspect and launch
- **Not a code editor plugin** (yet). The terminal is the interface because cognitive separation matters - preflight thinking happens outside the execution environment

## Philosophy

Bounded context + uncontrolled flow = reasoning collapse. That's the core equation from [tokendynamics](https://github.com/ppiankov/contextspectre).

VectorPad applies three principles:
- **Deterministic classification over ML.** Every sentence gets a tag via regex and token matching. No probabilities, no embeddings, no model calls
- **Structural safety over probabilistic confidence.** The smoke detector fires on measurable signals (brevity ratio, vague verbs, scope markers) - not on vibes
- **Meaning preservation over brevity.** Constraints get locked. Decisions get flagged. The vector leaves VectorPad with the same intent density it entered with

## Quick start

```bash
brew install ppiankov/tap/vectorpad
vectorpad
```

Or build from source:

```bash
git clone https://github.com/ppiankov/vectorpad.git
cd vectorpad
make build
./vectorpad
```

## Usage

### TUI (default)

Run `vectorpad` to open the three-panel interface. Paste or type your directive in the center editor. Classification, metrics, and risk analysis update live.

**Keybindings:**

| Key | Action |
|-----|--------|
| `ctrl+y` | Copy vector to clipboard |
| `ctrl+l` | Launch (copy + mark sent) |
| `ctrl+s` | Stash current vector |
| `ctrl+r` | Recall from stash into editor |
| `ctrl+e` | Extract essence from stash stack |
| `ctrl+x` | Prune stash entry |
| `Tab` | Next panel |
| `ctrl+h` | Help overlay |
| `ctrl+c` | Quit |

### CLI pipe mode

```bash
echo "update all repos to have readme" | vectorpad
```

Outputs classified vector block, pre-flight metrics (tokens, CPD, TTC, CDR), ambiguity analysis, and nudge prompts.

### Quick-add to stash

```bash
vectorpad add "context dilution attack vector"
```

Parks an idea in the stash from the command line. Clustering happens automatically.

### File attachments

Drag a file into the terminal - VectorPad intercepts the path, classifies the file type, and creates an attachment object. Files become named references in the vector, not raw content pasted inline. On copy-out, attachments serialize to text-safe excerpts.

## Architecture

```
┌──────────────┬────────────────────────────┬───────────────┐
│ STASH        │ VECTOR EDITOR              │ RISK PANEL    │
│              │                            │               │
│ stack list   │ [paste/type directive]     │ blast radius  │
│ by topic     │                            │ brevity ratio │
│ (Jaccard     │ ─── classified ───         │ vague verbs   │
│  clustering) │ [CONSTRAINT][LOCKED] ...   │ warning level │
│              │ [DECISION] ...             │               │
│              │ ─── dashboard ───          │ nudge prompts │
│              │ tokens | CPD | TTC | CDR   │               │
│              │                            │ pastewatch    │
│              │ ctrl+y copy  ctrl+h help   │ status        │
└──────────────┴────────────────────────────┴───────────────┘
```

**Internal packages:**

| Package | Responsibility |
|---------|---------------|
| `classifier` | 6-tag sentence classification (CONSTRAINT, DECISION, TENTATIVE, QUESTION, SPECULATION, EXPLANATION) |
| `drift` | Meaning drift detection on 6 axes (modality, negation, numeric, scope, conditional, commitment) |
| `vector` | Vector block rendering - grouped, classified output |
| `preflight` | Pre-flight metrics: token weight, vector integrity, CPD/TTC/CDR projections |
| `ambiguity` | Ambiguous vector detection: blast radius, brevity ratio, vague verbs, nudge protocol |
| `stash` | Idea persistence with Jaccard similarity clustering and uniqueness scoring |
| `detect` | Capability detection for pastewatch and contextspectre binaries |
| `attach` | File attachment pipeline: detect path, classify, preview, serialize |
| `tui` | Three-panel Bubbletea interface with responsive breakpoints |

**Responsive layout:** stash panel hides below 80 columns, risk panel collapses below 120 columns. Editor is always visible.

## Bond: the reasoning debugger

VectorPad is one half of a reasoning debugger. The other half is [ContextSpectre](https://github.com/ppiankov/contextspectre).

| Tool | Role | Analogy |
|------|------|---------|
| VectorPad | Pre-flight - structure intent before sending | Setting breakpoints and inspecting variables |
| ContextSpectre | Runtime - observe what happens inside the session | Watching the stack trace and stepping through execution |

The feedback loop: VectorPad predicts CPD/TTC/CDR → model executes → ContextSpectre measures actual metrics → operator adjusts the next vector.

Optional integration with [Pastewatch](https://github.com/ppiankov/pastewatch) scans outbound payloads for secrets before they enter a context window.

## Known limitations

- **Classifier is pattern-based.** Sentences without signal words ("must", "should", "will we", "maybe") classify as EXPLANATION by default. This is intentional - false negatives are safer than false positives
- **Blast radius counts text patterns, not actual repos.** It looks for numbers adjacent to scope words, not your filesystem
- **No persistent attachment content.** The stash stores path references, not file content. Stale paths are possible
- **macOS-first clipboard.** Uses `pbcopy` on macOS, `xsel`/`xclip` on Linux. No Windows support yet
- **No Cobra.** The CLI is hand-rolled. Works for 5 subcommands; will need Cobra if the CLI grows

## Roadmap

- [x] Phase 1 - classifier, drift, vector block, preflight, TUI shell
- [x] Phase 2 - ambiguity detection, stash persistence, capability detection
- [x] Phase 3 - file attachment pipeline (detect, classify, preview, serialize)
- [ ] Phase 4 - launch targets, essence extraction, stash uniqueness visualization, shell completions
- [ ] Phase 5 - contextspectre feedback loop (predicted vs actual metrics)

## License

[MIT](LICENSE)
