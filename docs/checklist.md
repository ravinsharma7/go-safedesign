# Project Checklist

This checklist is intentionally lightweight and arbitrary. It is a planning surface, not a proof contract.

Legend:

```text
[x] done
[~] in progress / partially implemented
[?] prototyping / discovery
[ ] projected / not started
```

## Done

- [x] Define proof-aware query result statuses.
- [x] Define trust levels from syntax observation to build validation.
- [x] Define out-of-order processing rule.
- [x] Define placeholder contract.
- [x] Define freshness states.
- [x] Define reusable core pipeline and analyzer architecture contract.
- [x] Refactor prototype into internal core, source, pipeline, indexer, and viewer packages.
- [x] Add analyzer metadata, stage constants, run record, and registry skeleton.
- [x] Add shared `safedesign.json` configuration loader.
- [x] Add generic internal analyzer runner that owns run records and config hashes.
- [x] Move viewer HTML into embedded static assets.
- [x] Split indexer implementation by pipeline stage responsibility.
- [x] Add graph stage run records for implemented stages.
- [x] Add first real analyzer: package dependency policy allow-list evaluation.
- [x] Emit structured dependency policy results with rule IDs, status, trust, config hash, and evidence.
- [x] Emit policy diagnostics with `fail` and `unknown` statuses.
- [x] Materialize package dependency policy query results from policy facts.
- [x] Add viewer policy status counts, filtering, and evidence-edge navigation.
- [x] Build a runnable placeholder-aware CLI prototype.
- [x] Add deterministic local fixture with at least 3 modules, 3 packages, and 5 Go files.
- [x] Emit canonical JSON graph output.
- [x] Add thin interactive browser viewer over the canonical JSON graph.
- [x] Create synthetic placeholder package nodes for missing imports.
- [x] Create unresolved call nodes instead of fake semantic targets.
- [x] Count runtime/concurrency markers as syntax facts.
- [x] Demonstrate negative query result as `unknown`.
- [x] Demonstrate paused path job for missing package.
- [x] Demonstrate simulated file freshness supersession.
- [x] Add regression tests and golden summary for the prototype.
- [x] Add generic observation/evidence facts for analyzer outputs that are not labels, metrics, warnings, or policies.
- [x] Add analyzer-result validation before merging facts into the canonical graph.
- [x] Add analyzer metadata contract validation before analyzer execution.
- [x] Add zero-config vocabulary extraction observations.
- [x] Add vocabulary co-occurrence observations.
- [x] Add internal convenience projections for vocabulary and co-occurrence observations.

## In Progress

- [~] Stable graph IR for modules, packages, files, imports, functions, placeholders, unresolved calls, and runtime markers.
- [~] Separation of core graph facts from analyzer-specific labels, metrics, warnings, and policies.
- [~] Package dependency policy analyzer; allow-list diagnostics exist, cycles/outdated modules/dependency loading policy are still projected.
- [~] Stage-oriented indexer implementation split across named pipeline responsibilities.
- [~] Pipeline run reporting; built-in stages and internal analyzers emit run records, external analyzer execution is still projected.
- [~] Package loading via `go/packages`.
- [~] Placeholder reconciliation when real package facts arrive.
- [~] Query model with explicit evidence and trust metadata.
- [~] Source provenance with file path, source hash, line range, extractor version, trust level, and freshness.
- [~] Path job model for paused/partial analysis.
- [~] Browser graph/code exploration UI with mode switching, pan, zoom, selection, neighbor focus, and source-block evidence without a duplicated schema.
- [~] Viewer policy navigation; policy results are clickable, evidence-backed, and status-filtered, richer drill-down is still projected.

## Prototyping / Discovery

- [?] Analyzer/internal stability roadmap captured in `docs/analyzer-stability-roadmap.md`.
- [?] DDD language-zone analysis design captured in `docs/ddd-language-zones.md`.
- [?] Separate viewer direction captured in `docs/viewer-direction.md`.
- [?] Final analyzer interface and package layout for core versus use-case analyzers.
- [?] Configuration schema for DDD, dependency policy, complexity weights, third-party labels, and framework extractors.
- [?] Best reconciliation model for placeholder symbols after type information arrives.
- [?] Whether call edges should attach first to enclosing function instead of package.
- [?] How to represent incomplete snippets/fragments that are not valid Go files.
- [?] How much `go/packages` failure detail should be normalized into structured diagnostics.
- [?] Whether graph output should be split into JSON and NDJSON modes.
- [?] How to keep the embedded debug viewer while adding a separately served dedicated viewer.
- [?] How rich the source-code UX should become before the viewer needs dedicated frontend assets.
- [?] Whether code-first navigation should become the default for call-path investigations.
- [?] How to integrate editor unsaved buffers through overlays.
- [?] How to represent build tags and ignored files in completeness proofs.
- [?] Whether path jobs should be persisted as separate state files.

## Projected

- [ ] External analyzer runner process/plugin model.
- [ ] Rich configuration provenance facts linked into the graph.
- [ ] DDD / ubiquitous language / bounded context analyzer.
- [ ] Tunable cyclomatic and cognitive complexity analyzers.
- [ ] Extend package dependency policy analyzer with cycles, missing dependency classes, allowed module imports, and outdated modules.
- [ ] Third-party module behavior catalog and IO behavior labeller.
- [ ] Framework extractor interface for routes, controllers, models, repositories, middleware, and jobs.
- [ ] Filesystem watcher with dynamic watched file set.
- [ ] Incremental reindexing keyed by file hash/version.
- [ ] Real stale/superseded/invalidated fact replacement store.
- [ ] Full package completeness tracking.
- [ ] Type-resolved symbol resolution using official type info.
- [ ] Build/test validation layer.
- [ ] Symbol query API.
- [ ] Dependency/import query API.
- [ ] Complexity query API.
- [ ] Runtime marker query API.
- [ ] Negative-answer proof scope evidence.
- [ ] Mermaid renderer generated from canonical graph data.
- [ ] DOT/Graphviz renderer generated from canonical graph data.
- [ ] Boundary rule checks such as domain-to-infra import violations.
- [ ] Path job resume triggers.
- [ ] Persistent cache/index format.
- [ ] CLI flags for trust level, scope, output mode, query type, and viewer filtering defaults.
- [ ] Documentation for fixture scenarios and expected graph behavior.
