# Proof-Aware Go Code Indexer Design Contract

## Goal

Build a proof-aware Go code indexer that can answer symbol, dependency, complexity, and path queries with explicit completeness status.

The tool must distinguish between:

1. a fact that exists,
2. a fact that does not exist within a fully analyzed scope,
3. a fact that is currently unknown because analysis is incomplete,
4. a fact that could not be validated because analysis failed.

The tool must not behave like grep. A negative answer must mean proven absence within the requested scope and validation level.

The first implemented analyzer beyond indexing is `dependency_policy`. It evaluates package import allow lists from `safedesign.json` against canonical import edges. Each evaluated import emits a structured `policy_result`; complete disallowed imports are `fail`, allowed imports are `pass`, and incomplete placeholder-backed imports are `unknown`.

---

## Non-Goals for MVP

The MVP does not:

1. prove runtime behavior,
2. fully model goroutine scheduling,
3. prove concurrency safety,
4. build a custom Go parser,
5. replace official Go validation tools,
6. depend on grep-style prior search assumptions,
7. guarantee complete call graph precision for dynamic/interface-heavy code,
8. treat syntax extraction as semantic correctness.

Concurrency-related syntax may be counted, but not semantically proven.

---

## Core Invariant

A negative answer is only allowed when the requested project scope has been fully analyzed at the required validation level.

Otherwise, the answer must be `unknown`, not `false`.

Formal rule:

```text
CanAnswerNo(query, scope, level) ⇔
  ScopeComplete(scope, level)
  ∧ AnalysisSucceeded(scope, level)
  ∧ CheckedAllRelevantFacts(query, scope, level)
```

Therefore:

```text
¬found ∧ complete_at_level   ⇒ does_not_exist
¬found ∧ incomplete_at_level ⇒ unknown
```

---

## Trust Levels

All extracted facts must carry a trust level.

```text
syntax_observed
  → package_loaded
  → type_resolved
  → build_validated
```

### `syntax_observed`

The fact was found from parsing Go syntax.

Example:

```text
function declaration found
if statement counted
go statement counted
import text observed
```

This level does not prove semantic correctness.

---

### `package_loaded`

The fact belongs to a package successfully loaded by official Go package tooling.

Example:

```text
file belongs to package
package imports another package
package files were discovered with build constraints applied
```

---

### `type_resolved`

The fact was resolved using Go type information.

Example:

```text
method receiver resolved
function call target resolved
identifier mapped to declaration
interface method partially resolved
```

---

### `build_validated`

The package/module passed official build or test validation.

Example:

```text
go build passed
go test compilation passed
```

This is the highest MVP trust level.

---

## Query Result Contract

No query may return a plain boolean.

Bad:

```go
func HasSymbol(name string) bool
```

Good:

```go
func FindSymbol(name string, scope Scope, level TrustLevel) QueryResult[Symbol]
```

Result model:

```go
type QueryStatus string

const (
    Exists          QueryStatus = "exists"
    DoesNotExist    QueryStatus = "does_not_exist"
    Unknown         QueryStatus = "unknown"
    AnalysisError   QueryStatus = "analysis_error"
)
```

A query result must include:

```text
status
scope
required trust level
actual trust level
evidence
errors, if any
```

Example:

```json
{
  "status": "unknown",
  "reason": "package_not_type_resolved",
  "scope": "github.com/acme/shop/internal/order",
  "requiredTrustLevel": "type_resolved",
  "actualTrustLevel": "syntax_observed"
}
```

---

## Pipeline

1. Watch project files using filesystem notifications.
2. Maintain a dynamic watched file set.
3. Parse changed Go files using `go/parser` and `go/ast`.
4. Extract syntax-level facts from changed files.
5. Parse `go.mod` using `x/mod/modfile`.
6. Load full packages with `go/packages` when package-level correctness is required.
7. Use official Go type/build tooling for semantic validation.
8. Convert Go AST/package/type data into a stable internal graph schema.
9. Store facts with source provenance, trust level, and freshness status.
10. Output canonical JSON, with Mermaid/DOT as visualization formats.
11. Count concurrency/runtime markers but do not reason about runtime behavior.
12. Resume paused path jobs when missing facts, files, packages, or validation results become available.

---

## Out-of-Order Processing Contract

The indexer may process files out of order.

However:

```text
Out-of-order processing may create syntax facts.
Out-of-order processing must not claim semantic validity.
```

Piecewise rule:

```text
single file parsed
  ⇒ syntax facts allowed

package loaded
  ⇒ package facts allowed

type info available
  ⇒ type-resolved facts allowed

build/test passed
  ⇒ build-validated facts allowed
```

A fact extracted out of order must be marked as partial unless its required context is complete.

---

## Placeholder Contract

The indexer may create synthetic placeholder nodes and edges for missing, unparsed, incomplete, or unresolved code.

Placeholder facts:

1. may participate in partial dependency graphs,
2. may preserve partial path fragments,
3. must carry `synthetic=true`,
4. must carry a reason,
5. must not satisfy negative-answer proof,
6. must not upgrade beyond `syntax_observed` without official validation,
7. must be reconciled, superseded, or invalidated when real facts arrive.

Valid placeholder examples:

```text
placeholder:package:github.com/acme/notify
unresolved_call:internal/order/service.go:42:Notify
```

A placeholder may support this statement:

```text
OrderService.PlaceOrder has a partial call edge to an unresolved notification target.
```

A placeholder must not support this statement:

```text
OrderService.PlaceOrder definitely calls github.com/acme/notify.Notify.
```

When a later file parse or package load discovers the real package or symbol, the graph must reconcile the placeholder edge to the real node or keep an explicit superseded/invalidated record.

---

## Fact Freshness Contract

Every fact must carry source provenance.

A fact must include:

```text
source file
source hash or version
line range
extractor version
trust level
created time
freshness status
```

Freshness states:

```text
fresh
stale
superseded
invalidated
```

When a file changes, facts from the old file version must not be silently reused as current facts.

---

## Path Processing Contract

A path analysis must be resumable.

A path may be:

```text
running
paused_missing_file
paused_missing_package
paused_validation_pending
partial
completed
failed
```

A path must not fail merely because some required node is not yet indexed.

Instead, it must preserve:

```text
path id
start node
target node, if any
known path segments
frontier
visited nodes
missing dependencies
blocking reason
required trust level
current trust level
```

Example:

```json
{
  "status": "paused_missing_package",
  "pathSoFar": [
    "OrderController.Create",
    "OrderService.Create"
  ],
  "missing": {
    "kind": "package",
    "importPath": "github.com/acme/shop/internal/payment"
  }
}
```

---

## Path Fragment Contract

The tool may process a path fragment without knowing what preceded it.

Example:

```text
PaymentService.Charge → StripeClient.CreatePayment
```

Even if this predecessor is unknown:

```text
OrderController.Create → ?
```

The fragment is valid as a partial path.

Formal rule:

```text
PathFragment ⊆ Path
CompletePath = PathFragment with all required predecessor and successor edges resolved
```

A partial path must never be presented as a complete path.

---

## Validation Contract

The tool must not implement Go semantic validation by itself.

The tool may extract syntax facts independently, but semantic validation must be delegated to official Go tooling.

Use:

```text
go/parser    → single-file syntax parsing
go/ast       → syntax tree traversal
go/packages  → package loading and build-context-aware package structure
go/types     → type checking and symbol resolution
go build     → build validation
go test      → test compilation/build validation
```

The tool’s own validation may check internal consistency of its graph, but not claim Go language correctness without official validation.

---

## Concurrency Marker Contract

The MVP does not prove runtime behavior.

The MVP may count and locate concurrency/runtime markers:

```text
go statement
defer statement
select statement
channel send
channel receive
mutex lock
mutex unlock
context usage
```

These markers must be reported as syntax/runtime markers, not as behavioral proof.

Example valid statement:

```text
Function SubmitOrder contains 2 go statements and 1 defer statement.
```

Example invalid statement:

```text
Function SubmitOrder is concurrency-safe.
```

---

## Graph Schema Contract

The internal graph schema must be stable and independent of Go AST internals.

Go AST nodes are input data, not the public model.

The system should convert:

```text
go/ast node
  → stable internal IR
  → graph facts
  → query result
  → visualization
```

Core node kinds:

```text
module
package
file
symbol
function
method
type
interface
import
decision
runtime_marker
path_fragment
```

Core edge kinds:

```text
contains
declares
imports
calls
references
implements
contains_decision
contains_runtime_marker
depends_on
violates_boundary
```

Use-case-specific concepts such as DDD layers, bounded contexts, framework roles, complexity metrics, and third-party IO behavior must be represented as labels, metrics, warnings, policy facts, or typed edges on the canonical graph.

They must not create separate graph schemas.

The detailed reusable pipeline contract lives in [`docs/pipeline.md`](./pipeline.md).

---

## Output Contract

Canonical output must be JSON or NDJSON.

Visualization formats are renderers, not the source of truth.

Supported MVP outputs:

```text
JSON
NDJSON
Mermaid
DOT / Graphviz
```

Mermaid and DOT must be generated from canonical graph data.

---

## Warning Contract

Every warning must point to evidence.

A warning must include:

```text
file
line range
symbol
reason
trust level
path evidence, if relevant
suggested next action, if possible
```

A warning without evidence is not actionable.

---

## Negative Answer Contract

A negative answer must include proof scope.

Example:

```json
{
  "status": "does_not_exist",
  "query": "function SubmitOrder",
  "scope": {
    "kind": "package",
    "value": "github.com/acme/shop/internal/order"
  },
  "proofLevel": "type_resolved",
  "evidence": {
    "filesAnalyzed": 12,
    "packagesLoaded": 1,
    "errors": []
  }
}
```

If any required file/package/type validation is missing, return:

```json
{
  "status": "unknown",
  "reason": "scope_incomplete"
}
```

---

## MVP Implementation Order

The running project checklist lives in [`docs/checklist.md`](./checklist.md).

### Prototype: Placeholder-Aware CLI Demo

The checked-in prototype demonstrates the contract with a deterministic local workspace:

```text
go run ./src --path testdata/workspace/shop --json
go run ./src --path testdata/workspace/shop --serve :8080
```

The fixture contains:

```text
3 modules:
  example.com/shop
  example.com/payments
  example.com/missing-notification

at least 3 packages:
  example.com/shop/cmd/app
  example.com/shop/order
  example.com/shop/paymentclient
  example.com/payments/gateway

at least 5 Go files
```

The prototype intentionally demonstrates:

1. out-of-order file parsing with later placeholder reconciliation,
2. a missing package represented as a synthetic placeholder,
3. unresolved calls represented as `unresolved_call` nodes,
4. runtime markers counted as syntax facts,
5. a negative symbol query returning `unknown`,
6. a paused path job for a missing package,
7. a simulated changed-file freshness record.

The browser viewer is intentionally thin. It serves the same canonical graph through `/graph.json` and derives the browser UI from that JSON instead of maintaining a second graph schema. It may add interaction state such as navigation mode, pan, zoom, selected node, neighbor-only filtering, selected source block, and related source blocks, but that state must not become a separate source of graph truth.

The viewer may expose source-code UX for facts that carry `sourceFile` and `lineRange`. Source blocks are fetched from the fixture/source tree and are treated as evidence display, not as a separate parser or graph model.

The viewer should support at least two navigation modes:

1. graph-first navigation, where the node graph is primary and source blocks explain the selected graph fact,
2. code-first navigation, where source-backed functions/calls are primary and the graph acts as surrounding context.

The prototype is not the full daemon, watcher, or production query engine. It is a small executable proof that the IR can preserve partial knowledge without pretending it is semantically validated.

### Phase 1: Syntax Indexer

Implement:

```text
read go.mod
discover Go files
watch file changes
parse changed Go files
extract packages/imports/functions/methods/types
count if/switch/for/range/select
count go/defer/channel/select markers
emit JSON facts
```

### Phase 2: Stable Graph

Implement:

```text
module/package/file/function nodes
contains/declares/imports edges
canonical JSON graph output
Mermaid/DOT renderers
```

### Phase 3: Proof-Aware Queries

Implement:

```text
find symbol
list package imports
find functions above complexity threshold
find files containing concurrency markers
return exists/does_not_exist/unknown/analysis_error
```

### Phase 4: Package and Type Validation

Implement:

```text
go/packages loading
go/types resolution
type-resolved symbol facts
package completeness status
validation error reporting
```

### Phase 5: Resumable Path Jobs

Implement:

```text
path job state
partial path fragments
paused paths
resume triggers
missing dependency tracking
```

---

## Core Summary

The tool must parse early, validate later, answer carefully, and resume automatically.

It must never confuse:

```text
not found
```

with:

```text
does not exist
```

It must never confuse:

```text
syntax observed
```

with:

```text
semantically valid
```
