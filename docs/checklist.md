# Project Checklist

This checklist tracks active uncertainty and projected work. Stable completed milestones
are intentionally removed from this file once they are covered by tests, docs, and git
history.

Legend:

```text
[~] in progress / partially implemented
[?] prototyping / discovery
[ ] projected / not started
```

## In Progress

- [~] Stable graph IR for modules, packages, files, imports, functions, placeholders, unresolved calls, runtime markers, observations, labels, metrics, warnings, policies, and queries.
- [~] Package dependency policy analyzer; allow-list diagnostics exist, cycles/outdated modules/dependency loading policy are still projected.
- [~] Package loading via `go/packages`.
- [~] Placeholder reconciliation when real package facts arrive.
- [~] Source provenance with file path, source hash, line range, extractor version, trust level, and freshness.
- [~] Path job model for paused/partial analysis.
- [~] Browser graph/code exploration UI with mode switching, pan, zoom, selection, neighbor focus, and source-block evidence without a duplicated schema.
- [~] Viewer policy navigation; policy results are clickable, evidence-backed, and status-filtered, richer drill-down is still projected.
- [~] Language-zone candidate evidence; candidates are observations for human inspection, not bounded-context conclusions.
- [~] Bridge-symbol evidence; bridge observations explain candidate-zone coupling without declaring boundaries.

## Prototyping / Discovery

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
- [ ] Bounded-context inference from language-zone candidates and explicit user annotations.
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
