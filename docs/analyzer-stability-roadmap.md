# Analyzer and Internal Stability Roadmap

This document captures the near-term priority: make the analyzer pipeline and canonical
fact model stable before investing in a richer viewer.

The viewer can be replaced independently if the core emits durable, evidence-backed,
well-versioned facts. Therefore the next work should strengthen the internal model,
analyzer contracts, and regression tests.

## Priority

The next phase SHOULD focus on:

1. Stable analyzer output contracts.
2. Durable evidence facts for DDD language analysis.
3. Validation of analyzer output before facts enter the graph.
4. Deterministic graph output and regression tests.
5. Clear separation between observed facts, configured facts, inferred facts, warnings,
   metrics, policies, and projections.

Viewer work SHOULD be limited to debug inspection until these are stable.

## Current Gap

The existing `ubiquitous_language` analyzer is useful, but it is config-first:

```text
configured contexts
configured package prefixes
configured allowed terms
configured synonyms
configured discouraged terms
```

That mode is better treated as validation against known language decisions.

DDD discovery needs a separate zero-config evidence pass:

```text
observed original spelling
normalized tokens
source node
source location
package/module scope
trust level
producer metadata
```

The discovery pass should not require known contexts, known terms, or package-prefix
boundaries.

## Recommended Analyzer Split

Use separate analyzer responsibilities:

```text
vocabulary_extraction
  Emits observed vocabulary evidence from existing graph nodes and syntax snapshots.

vocabulary_cooccurrence
  Emits term co-occurrence and neighborhood evidence.

language_zone_candidate
  Emits inferred candidate language zones from vocabulary and dependency evidence.

ubiquitous_language_validation
  Compares observed vocabulary against configured DDD decisions.
```

The current `ubiquitous_language` analyzer MAY remain as the validation analyzer, but
zero-config discovery should not be forced through that configuration model.

## Evidence Facts

Before adding language-zone inference, add a fact type capable of representing
non-warning evidence.

Options:

1. Add a generic `Evidence` fact to `core.Graph`.
2. Add a narrower `Observation` fact to `core.Graph`.
3. Represent vocabulary observations as namespaced `Label` facts.

The preferred direction is a generic `Observation` fact because vocabulary terms,
co-occurrences, dependency clusters, and external annotations are not necessarily labels,
warnings, metrics, or policies.

Candidate shape:

```go
type Observation struct {
    ID          string     `json:"id"`
    Kind        string     `json:"kind"`
    Name        string     `json:"name"`
    Value       string     `json:"value,omitempty"`
    TargetID    string     `json:"targetId,omitempty"`
    TargetKind  string     `json:"targetKind,omitempty"`
    Attributes  map[string]string `json:"attributes,omitempty"`
    Evidence    []string   `json:"evidence,omitempty"`
    Source      string     `json:"source"` // observed | configured | inferred | imported
    Confidence  float64    `json:"confidence,omitempty"`
    TrustLevel  TrustLevel `json:"trustLevel"`
    Freshness   string     `json:"freshness,omitempty"`
    SourceFile  string     `json:"sourceFile,omitempty"`
    LineRange   string     `json:"lineRange,omitempty"`
    FactMetadata
}
```

Example vocabulary observation:

```json
{
  "kind": "observation",
  "name": "vocabulary.term",
  "value": "order",
  "targetId": "function:example.com/shop/order.PlaceOrder",
  "targetKind": "node",
  "attributes": {
    "original": "PlaceOrder",
    "tokenIndex": "1",
    "packagePath": "example.com/shop/order",
    "modulePath": "example.com/shop"
  },
  "source": "observed",
  "trustLevel": "syntax_observed"
}
```

## Analyzer Contract Hardening

The pipeline should validate analyzer results before merging them into the graph.

Validation SHOULD check:

1. Every fact has an ID.
2. Every fact has a kind.
3. Every fact has trust level when the fact type requires it.
4. Every evidence reference points to an existing fact or is explicitly marked external.
5. Every target ID exists when the fact claims to target a graph node or edge.
6. Analyzer-emitted trust does not exceed `MaximumEmittedTrust`.
7. Analyzer output kinds match `EmittedFactKinds`.
8. Fact IDs are deterministic for deterministic inputs.
9. Analyzer errors produce diagnostics and do not partially merge invalid facts.

This validation should happen in one internal place rather than inside every analyzer.

## Near-Term Implementation Plan

1. Add `Observation` to `core.Graph`, sorting, metadata assignment, and fact counting.
2. Extend `pipeline.AnalyzerResult` so analyzers can emit observations.
3. Add analyzer-result validation before merge.
4. Add `vocabulary_extraction` as a zero-config analyzer.
5. Move reusable identifier splitting out of `ubilang` into a shared internal package or
   keep it private until a second analyzer needs it.
6. Add tests proving original spelling preservation, acronym handling, deterministic IDs,
   source metadata, and package/module scoping.
7. Keep the current configured DDD analyzer separate as validation, not discovery.
8. Only after observations exist, add co-occurrence and candidate zone inference.

## What Not To Do Yet

Do not build bounded-context inference first.

Do not build a dedicated DDD viewer first.

Do not encode DDD concepts into folder or framework assumptions.

Do not treat capitalization inconsistency as a warning until there is enough evidence to
separate language inconsistency from serialization, SQL, generated code, or API boundary
differences.
