# DDD Language Zone Analysis

This document is a design note for ubiquitous language and bounded-context analysis.
It is intentionally separate from the core pipeline contract because the model will need
multiple iterations before it becomes normative.

The analyzer MUST NOT assume framework structure, folder structure, or naming patterns
such as service, repository, controller, provider, model, or handler. These names MAY be
recorded as vocabulary evidence, but they MUST NOT be treated as architecture facts unless
another analyzer or user annotation supplies that meaning.

## Goal

Use the existing code graph, source language, and optional external annotation stores to
help humans discover language zones that may correspond to DDD bounded contexts.

The analyzer should first answer weaker, evidence-backed questions:

1. Which terms are used?
2. Where are those terms used?
3. Which terms co-occur?
4. Which symbols depend on each other?
5. Which terms appear with different meanings in different neighborhoods?
6. Which terms appear to be synonyms in the same neighborhood?
7. Which symbols or packages bridge otherwise separate language zones?

Only after those facts exist should the tool propose candidate bounded contexts.

## Reserved Terms

### Term

A `term` is a normalized vocabulary token or phrase extracted from source material.

Examples:

```text
invoice
invoice line
payment gateway
user id
```

Terms MUST preserve links to every original spelling that produced them.

### Original Spelling

An `original spelling` is the exact text observed in source material.

Examples:

```text
UserID
UserId
user_id
invoice-line
INVOICE_LINE
```

Original spelling matters because naming style can be evidence of language boundaries,
generated code, external API shape, or inconsistent terminology.

### Language Zone

A `language zone` is a cluster of symbols, files, packages, or graph nodes that share
vocabulary and relationship signals.

A language zone is not automatically a bounded context. It is a candidate area where a
human may inspect the language and decide whether a DDD boundary exists.

### Candidate Bounded Context

A `candidate bounded context` is a human-facing interpretation produced from one or more
language zones and their evidence.

The analyzer SHOULD phrase these as candidates, not conclusions, unless explicit user
configuration or annotation promotes them.

## Evidence Model

DDD analysis SHOULD emit evidence facts before emitting interpretation facts.

Useful evidence includes:

```text
identifier_term
file_path_term
package_term
comment_term
database_term
external_annotation_term
term_cooccurrence
term_spelling_variant
term_neighborhood
dependency_cluster
bridge_symbol
cross_zone_term
ambiguous_term
possible_synonym
```

Each evidence fact SHOULD include:

```text
id
kind
term or terms
source node ids
source edge ids, if applicable
original spelling, if applicable
normalization method
trust level
provenance
producer id
producer version
confidence, when inferred
```

Interpretation facts SHOULD reference evidence facts instead of recomputing private
state.

Example:

```json
{
  "kind": "candidate_language_zone",
  "label": "invoice/payment/refund cluster",
  "sourceFacts": [
    "term_cooccurrence:invoice:payment",
    "dependency_cluster:pkg-set-17",
    "bridge_symbol:payments.Gateway"
  ],
  "confidence": 0.68,
  "status": "candidate"
}
```

## Normalization

The first vocabulary pass SHOULD split common identifier styles:

```text
camelCase
PascalCase
snake_case
kebab-case
SCREAMING_SNAKE_CASE
mixed acronym forms such as UserID, UserId, HTTPServer, HttpServer
```

Normalization MUST NOT discard the original spelling.

The normalized representation SHOULD support:

```text
lowercase tokens
ordered token phrases
acronym handling
stop-word filtering
domain-specific stop-word overrides
stemming or lemmatization only as an optional pass
```

The analyzer SHOULD treat style inconsistency as evidence, not automatically as a
warning. For example, `UserID`, `UserId`, and `user_id` may be normal when crossing Go,
JSON, SQL, and external API boundaries.

## Inference Signals

Initial language-zone inference SHOULD combine weak signals instead of relying on one
strong assumption.

Useful signals:

1. Term frequency by package, file, symbol, and dependency cluster.
2. Term co-occurrence within symbols, files, packages, and graph neighborhoods.
3. Dependency graph clustering with internal and external edge density.
4. Vocabulary overlap between dependency clusters.
5. Same term appearing in different neighborhoods.
6. Different terms appearing in the same neighborhood.
7. Bridge nodes with high cross-cluster connectivity.
8. Cross-zone generic terms that should not dominate context naming.
9. Optional git change coupling when history is available.
10. Optional user or external database annotations.

The analyzer SHOULD surface confidence and evidence, not a single authoritative answer.

## External Annotation Store

SQLite or DuckDB can be used as an annotation and analysis store without changing the
canonical graph contract.

Useful annotation tables may include:

```text
canonical_terms
term_aliases
ignored_terms
term_domains
external_api_terms
database_terms
generated_source_marks
human_context_labels
review_decisions
```

Annotations SHOULD be imported as configured or observed facts with provenance. They
SHOULD NOT be hidden side channels required to understand analyzer output.

## Visualization Direction

The current graph viewer is useful for raw inspection, but DDD analysis needs progressive
disclosure. A better viewer should make it easy to move between overview, candidate zone,
term, symbol, and evidence views.

Required viewer capabilities:

1. Switch between dependency, vocabulary, and combined views.
2. Collapse dense graph areas into language-zone candidates.
3. Expand a zone into packages, files, symbols, terms, or evidence edges.
4. Filter by term, original spelling, trust level, producer, confidence, and annotation
   status.
5. Highlight ambiguous terms that appear in different neighborhoods.
6. Highlight possible synonyms that appear in the same neighborhood.
7. Show bridge symbols and cross-zone dependencies.
8. Compare two candidate zones by vocabulary overlap and dependency edges.
9. Keep source evidence one click away from every inferred claim.
10. Allow human decisions such as confirm, reject, rename, ignore, and annotate.

The viewer SHOULD avoid rendering every node at once by default. The default experience
should start with summarized clusters and let the user drill down.

## Suggested Iteration Plan

Completed foundation:

1. Analyzer/internal contracts are guarded by metadata and result validation.
2. `Observation` is the durable evidence fact for non-warning analyzer output.
3. Zero-config vocabulary extraction emits normalized term observations with source scope.
4. Vocabulary co-occurrence emits relationship evidence without bounded-context inference.
5. Internal observation projections provide optional convenience summaries over raw facts.
6. Language-zone candidate analysis emits package-scoped candidate observations for human
   inspection.
7. Bridge-symbol analysis emits dependency-backed evidence between candidate zones.

Next iterations:

1. Add ambiguity and synonym candidate evidence.
2. Add richer dependency-neighborhood summaries around bridge evidence.
3. Add an annotation import/export path using SQLite or DuckDB.
4. Redesign the viewer around progressive disclosure after the internal facts are stable.

## Prompting Guidance

Useful next prompts:

```text
Analyze the current graph and propose the smallest vocabulary evidence schema for DDD
language-zone analysis.
```

```text
Implement candidate language-zone observations from existing vocabulary, co-occurrence,
and dependency evidence without assuming service, repository, controller, or folder
conventions.
```

```text
Redesign the viewer interaction model for progressive disclosure of dependency clusters,
terms, symbols, and evidence.
```

```text
Review the existing analyzer code and identify where architecture-pattern assumptions
would leak into DDD analysis.
```
