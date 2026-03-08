# Bond: VectorPad ↔ ContextSpectre

## Identity

| Field | VectorPad | ContextSpectre |
|-------|-----------|----------------|
| Repo | ppiankov/vectorpad | ppiankov/contextspectre |
| Role | Pre-flight (intent authoring) | Runtime (session observation) |
| Tagline | Vector launch pad | Reasoning hygiene layer |
| Phase | Greenfield | Mature (v0.25.0, 87+ WOs) |

## Data flow

```
Operator → [VectorPad] → Context Window → [ContextSpectre] → Operator
            predicts                        measures
```

VectorPad predicts CPD/TTC/CDR before sending.
ContextSpectre measures actual CPD/TTC/CDR after execution.
The operator closes the loop by adjusting the vector.

## Shared metrics

| Metric | Full name | Direction |
|--------|-----------|-----------|
| CPD | Cost Per Decision | VP predicts → CS measures |
| TTC | Turns To Convergence | VP predicts → CS measures |
| CDR | Context Drift Rate | VP predicts → CS measures |

## Debugger mapping

| IDE concept | VectorPad | ContextSpectre |
|-------------|-----------|----------------|
| Breakpoints | Decision boundaries | Commit points |
| Variables | Vector state editor | Vector extraction |
| Step execution | - | Turn-by-turn observation |
| Stack trace | - | Compaction archaeology |

## Lineage

VectorPad WO-01..03 originated from ContextSpectre WO-087.
WO-087 status: `[→] moved to ppiankov/vectorpad`

## Philosophy (shared)

- Deterministic over ML
- Structural safety over probabilistic confidence
- Mirrors, not oracles
- Control over observability
