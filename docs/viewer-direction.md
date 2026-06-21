# Viewer Direction

This document tracks the viewer as a separate design surface from the analyzer pipeline.
The current embedded viewer is useful as a thin prototype, but DDD language-zone analysis
needs a dedicated frontend model with progressive disclosure.

## Decision

The production viewer SHOULD be served separately from the analyzer CLI.

The Go analyzer should own:

```text
source discovery
graph construction
analyzer execution
canonical fact output
query/result endpoints
source evidence endpoints
```

The viewer should own:

```text
layout
interaction state
filter state
progressive disclosure
visual comparison flows
human annotation workflows
```

The viewer MUST NOT own a second graph truth. It may derive view models from canonical
facts, but those view models must remain disposable UI state.

## Why Separate It

DDD analysis is not just a graph drawing problem. The user needs to switch between:

```text
dependency overview
candidate language zones
vocabulary terms
original spellings
symbols and packages
source evidence
annotations and review decisions
```

That interaction will need dedicated layout, state management, filtering, and visual
iteration. Keeping all of that inside an embedded HTML string will make the viewer hard
to test and hard to redesign.

## Boundary

The analyzer and server boundary SHOULD expose stable data, not frontend assumptions.

Initial endpoints can stay small:

```text
GET /graph.json
GET /source?file=...&range=...
GET /viewer/summary
GET /viewer/language-zones
GET /viewer/terms
GET /viewer/term/{id}
GET /viewer/node/{id}/evidence
```

The `/viewer/*` endpoints are allowed to be read-optimized projections over canonical
facts. They SHOULD be versioned or clearly marked as viewer API if they become stable.

## Frontend Shape

The first separate viewer does not need a heavy application framework unless the design
requires it. The important split is that the frontend is built, tested, and iterated as
frontend code instead of being maintained as an embedded string.

Possible structure:

```text
viewer/
  package.json
  src/
  public/
  dist/
```

The Go binary MAY serve built viewer assets from `viewer/dist` for convenience, but the
viewer development server SHOULD be able to run independently against a graph JSON file
or analyzer HTTP server.

## Progressive Disclosure Model

The default view SHOULD not render every node at once.

Recommended navigation levels:

1. Overview: summarized dependency clusters and candidate language zones.
2. Zone: packages, dominant terms, bridge symbols, and cross-zone edges.
3. Term: original spellings, co-occurring terms, symbol occurrences, and neighborhoods.
4. Symbol: dependencies, source evidence, labels, warnings, metrics, and query results.
5. Evidence: exact source location, producer, trust level, confidence, and provenance.

Each inferred claim in the viewer SHOULD have a path back to source evidence.

## Near-Term Plan

1. Keep the existing embedded viewer as the debug viewer.
2. Add a small viewer API layer over canonical graph facts.
3. Add language-zone summary projection endpoints before building the new frontend.
4. Create a separate `viewer/` frontend workspace.
5. Build the first screen as a progressive-disclosure overview, not a full graph canvas.
6. Move richer DDD visualization work into the separate viewer while preserving the
   existing CLI JSON output.

## Open Questions

1. Should the analyzer server be long-running, or should the viewer mostly load static
   graph snapshots?
2. Should annotations be written through the viewer API, or exported as patchable
   SQLite/DuckDB files first?
3. Should the first separate viewer use a framework, or start with plain TypeScript and
   a small rendering library?
4. Which graph layout library is appropriate for clustered language-zone exploration?
