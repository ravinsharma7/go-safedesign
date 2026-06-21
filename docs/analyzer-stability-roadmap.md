# Analyzer Stability Roadmap

The near-term priority is still the analyzer core, not the viewer. The core should emit
durable, evidence-backed facts that can be consumed directly by agents, tests, future
analyzers, or replaceable viewers.

## Current State

Implemented core guardrails:

1. `Observation` facts exist in the canonical graph for durable non-warning evidence.
2. Analyzer results are validated before facts merge into the canonical graph.
3. Analyzer metadata is validated before execution.
4. Placeholder-backed and incomplete inputs are handled through analyzer-declared
   incomplete-input policy.
5. Zero-config vocabulary extraction emits `vocabulary.term` observations.
6. Zero-config vocabulary co-occurrence emits `vocabulary.cooccurrence` observations.
7. `internal/observations` provides convenience projections over observations.
8. Language-zone candidate analysis emits package-scoped candidate observations for
   human inspection.

Projection helpers are not canonical semantics. They are read-only summaries over
`graph.Observations` for callers that want common groupings without duplicating traversal
logic. An analyzer or agent may consume raw observations directly when that is clearer.

## Analyzer Responsibility Split

Keep analyzer responsibilities separate:

```text
vocabulary_extraction
  Emits observed vocabulary evidence from graph nodes and source-backed symbols.

vocabulary_cooccurrence
  Emits inferred co-occurrence evidence from vocabulary observations.

ubiquitous_language
  Validates configured DDD language decisions. It is not the discovery analyzer.

language_zone_candidate
  Emits candidate language-zone evidence before any bounded-context conclusion.
```

## Next Steps

1. Add more observation projections only when they remove real duplication for callers.
2. Add ambiguity, synonym, and bridge-symbol evidence as observations.
3. Keep placeholder-backed evidence from supporting complete conclusions.
4. Keep configured validation separate from zero-config discovery.
5. Avoid viewer-specific assumptions in analyzer output.

## What Not To Do Yet

Do not build bounded-context conclusions before candidate evidence is stable.

Do not make projection helpers mandatory inputs for analyzers.

Do not encode DDD meaning from folder names, framework roles, or service/repository
patterns unless those assumptions are supplied as explicit external annotations.
