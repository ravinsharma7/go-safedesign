# Core Pipeline and Analyzer Contract

This document is normative. A coding agent MUST treat `MUST`, `MUST NOT`, `SHOULD`, and `MAY` as implementation requirements.

The goal is to support many analysis use cases through one core pipeline:

```text
DDD / ubiquitous language / bounded context analysis
complexity metrics
package and dependency policy
third-party IO behavior labels
framework structure extraction
path and query workflows
viewer/rendering workflows
```

No use case MAY fork the canonical fact model.

---

## Reserved Terms

Use these words only with the meanings below.

### Source

`Source` means input material read by the tool.

Examples:

```text
Go file
go.mod
go.work
configuration file
generated in-memory overlay
```

`Source` does not mean a graph node, query result, or analyzer output.

### Fact

`Fact` means one canonical graph record emitted by the pipeline.

A fact is one of:

```text
node
edge
label
metric
warning
policy_result
query_result
path_job
diagnostic
freshness_record
```

Every fact MUST carry:

```text
id
kind
trust level
provenance
freshness
producer id
producer version
created time or run id
```

### Node

`Node` means an entity in the canonical graph.

Examples:

```text
module
package
file
function
method
type
interface
route
endpoint
placeholder
runtime_marker
```

Nodes MUST NOT encode policy pass/fail status. Use `policy_result` or `warning` for that.

### Edge

`Edge` means a directed relationship between two nodes.

Examples:

```text
contains
declares
imports
calls
references
implements
depends_on
routes_to
handles
reads
writes
publishes
subscribes
violates
```

Edges MUST NOT contain large use-case-specific payloads. Use labels, metrics, warnings, or policy results attached to edges when needed.

### Label

`Label` means classification metadata attached to a node or edge.

Examples:

```text
ddd.layer=domain
bounded_context=ordering
io.network=true
framework.role=controller
```

Labels MUST have a namespace. Valid examples:

```text
ddd.layer
ddd.context
io.network
framework.role
module.behavior
```

Labels MUST say whether they are:

```text
configured
observed
inferred
```

### Metric

`Metric` means a numeric measurement attached to a node, edge, package, context, or scope.

Examples:

```text
cyclomatic_complexity=8
cognitive_complexity=13
decision_count=5
runtime_marker_count=2
```

Metrics MUST include:

```text
name
value
unit or scale
configuration hash when configurable
source facts
```

### Warning

`Warning` means an actionable issue with evidence.

Warnings MUST include:

```text
reason
source facts
affected node or edge
trust level
suggested next action, if known
```

Warnings MUST NOT be emitted without evidence.

### Policy Result

`Policy result` means the result of evaluating a configured rule.

Examples:

```text
allowed_import: pass
allowed_import: fail
domain_must_not_import_infra: fail
max_complexity: fail
```

Policy results MUST include:

```text
rule id
status: pass | fail | unknown | analysis_error
scope
source facts
configuration hash
```

### Analyzer

`Analyzer` means a named pipeline stage that consumes canonical facts and emits canonical facts.

Analyzers MUST NOT:

```text
define private graph schemas
write viewer-specific JSON
delete facts emitted by other analyzers
upgrade trust without required evidence
```

### Extractor

`Extractor` means an analyzer that reads sources directly.

Examples:

```text
go.mod extractor
Go syntax extractor
configuration extractor
```

Most analyzers SHOULD consume facts, not raw files.

### Renderer

`Renderer` means a consumer that turns facts into a human-facing format.

Examples:

```text
JSON output
NDJSON output
browser viewer
Mermaid
DOT
```

Renderers MUST NOT become sources of truth. They MAY keep UI state only.

---

## Canonical Pipeline Stages

Implement these stage names exactly in code, configuration, logs, diagnostics, and analyzer run records.

Stages MAY run incrementally. Every emitted fact MUST preserve trust, provenance, and freshness.

### Stage 01: `source_discovery`

Purpose:

```text
discover source roots, modules, files, and configuration files
```

Inputs:

```text
user-provided path
config path, if provided
overlay map, if provided
```

Outputs:

```text
source_record
freshness_record
diagnostic
```

Must not:

```text
parse Go syntax
claim package completeness
claim module dependency correctness
```

### Stage 02: `module_extraction`

Purpose:

```text
parse go.mod and go.work data
```

Inputs:

```text
source_record for go.mod/go.work
```

Outputs:

```text
module node
dependency edge
replace edge or attribute
diagnostic
```

Required tooling:

```text
golang.org/x/mod/modfile
```

### Stage 03: `syntax_extraction`

Purpose:

```text
parse individual Go files and emit syntax_observed facts
```

Inputs:

```text
source_record for .go file
```

Outputs:

```text
file node
package node with syntax_observed trust
function/method/type/interface/import/runtime_marker nodes
contains/declares/imports/calls edges
unresolved_call placeholders
diagnostic
```

Required tooling:

```text
go/parser
go/ast
```

Must not:

```text
claim package_loaded
claim type_resolved
claim build_validated
answer does_not_exist
```

### Stage 04: `base_graph_assembly`

Purpose:

```text
merge syntax and module facts into the canonical graph
```

Inputs:

```text
facts from module_extraction
facts from syntax_extraction
```

Outputs:

```text
canonical graph nodes and edges
placeholder nodes
reconciled edges
freshness records
```

Rules:

```text
same fact id + same source version => same fact
same fact id + different source version => old fact becomes superseded or stale
placeholder + real fact for same identity => placeholder is reconciled, superseded, or retained with explicit reason
```

### Stage 05: `package_loading`

Purpose:

```text
load packages with Go build context and official tooling
```

Inputs:

```text
module facts
package syntax facts
build constraints
overlay map, if present
```

Outputs:

```text
package_loaded labels or node trust upgrades
package diagnostics
missing package diagnostics
import cycle diagnostics
```

Required tooling:

```text
golang.org/x/tools/go/packages
```

Must not:

```text
hide syntax_observed facts when loading fails
turn missing package into does_not_exist unless required scope is complete
```

### Stage 06: `type_resolution`

Purpose:

```text
resolve identifiers, methods, receivers, call targets, and type relationships
```

Inputs:

```text
package_loaded facts
go/packages type information
```

Outputs:

```text
type_resolved labels or trust upgrades
resolved call edges
references edges
implements edges
superseded unresolved_call placeholders
diagnostic
```

Must not:

```text
invent call targets
resolve dynamic/interface-heavy calls without evidence
```

### Stage 07: `module_dependency_enrichment`

Purpose:

```text
enrich module/package dependency facts
```

Inputs:

```text
module facts
dependency edges
package loading diagnostics
module metadata cache, if provided
```

Outputs:

```text
module labels
dependency diagnostics
outdated_module warnings
missing_dependency warnings
```

Network access policy MUST be explicit in configuration. If network is disabled, outdated checks MUST return `unknown`, not `pass`.

### Stage 08: `third_party_behavior_labeling`

Purpose:

```text
label external modules/imports/symbols with behavior categories
```

Inputs:

```text
module facts
import facts
call facts
configured behavior catalog
```

Outputs:

```text
module.behavior labels
io.* labels
framework.* labels
diagnostic
```

Behavior labels MUST be one of:

```text
configured
observed
inferred
```

Only configured or observed labels MAY support strict policy failure. Inferred labels MAY produce warnings but MUST NOT support `does_not_exist`.

### Stage 09: `framework_extraction`

Purpose:

```text
extract framework-specific structures
```

Inputs:

```text
syntax facts
type_resolved facts, required when extractor mode is semantic
framework behavior labels
framework configuration
```

Outputs:

```text
route nodes
endpoint nodes
framework.role labels
routes_to edges
handles edges
diagnostic
```

Framework extractors MAY be absent. Missing extractor support means `unknown`, not absence of framework structure.

### Stage 10: `ddd_classification`

Purpose:

```text
classify packages/symbols into bounded contexts and DDD roles
```

Inputs:

```text
module/package/file/symbol facts
import/call edges
configured bounded contexts
configured layer rules
naming dictionaries, if provided
```

Outputs:

```text
ddd.context labels
ddd.layer labels
ddd.role labels
shared_kernel labels
domain_event labels
diagnostic
```

Configured labels are stronger than inferred labels. Inferred labels MUST include confidence and evidence.

### Stage 11: `complexity_metrics`

Purpose:

```text
compute tunable metrics for code scopes
```

Inputs:

```text
syntax facts
decision markers
runtime markers
complexity configuration
```

Outputs:

```text
metric: cyclomatic_complexity
metric: cognitive_complexity
metric: decision_count
metric: runtime_marker_count
```

Metric configuration MUST define all non-default weights. If no configuration is provided, defaults MUST be recorded in the metric provenance.

Current prototype implementation emits function/method `cyclomatic_complexity`, `decision_count`, and `runtime_marker_count` metrics. It does not yet emit cognitive complexity or package/module aggregate metrics.

### Stage 12: `policy_evaluation`

Purpose:

```text
evaluate configured rules against canonical facts
```

Inputs:

```text
canonical graph
labels
metrics
configuration rules
required trust level
```

Outputs:

```text
policy_result
warning
violates edge
```

Policy status MUST be one of:

```text
pass
fail
unknown
analysis_error
```

Rules:

```text
missing required facts => unknown
analyzer failure => analysis_error
complete evidence and rule satisfied => pass
complete evidence and rule violated => fail
```

Current prototype implementation:

```text
dependency_policy evaluates configured package import allow lists
complete disallowed import edge => fail policy_result plus diagnostic
incomplete or placeholder import edge => unknown policy_result plus diagnostic
allowed complete import edge => pass policy_result
policy_result evidence references the evaluated import edge
```

### Stage 13: `query_materialization`

Purpose:

```text
answer user/API queries with explicit completeness status
```

Inputs:

```text
canonical graph
trust level requested by query
scope requested by query
policy results, required when query references policy state
```

Outputs:

```text
query_result
path_job
```

Current prototype materializes dependency-policy queries for every scope with emitted policy results. Failed policy results make the query `fail`; if no failures exist but any policy result is `unknown`, the query is `unknown`; the query is `pass` only when all scoped policy results pass.

Query status MUST be one of:

```text
exists
does_not_exist
unknown
analysis_error
```

No query MAY return a plain boolean.

### Stage 14: `rendering`

Purpose:

```text
render canonical facts for humans or tools
```

Inputs:

```text
canonical graph
query results
path jobs
warnings
diagnostics
```

Outputs:

```text
JSON
NDJSON
browser viewer
Mermaid
DOT
```

Renderers MUST read canonical facts. Renderers MUST NOT create facts that later stages consume.

---

## Analyzer Interface Contract

Every analyzer MUST declare:

```text
id
version
stage
input fact kinds
minimum required trust level
maximum emitted trust level
emitted fact kinds
failure mode
incomplete input policy
```

`configuration section name` is optional. An analyzer declares it only when it consumes
configuration.

Every analyzer run MUST emit an analyzer run record:

```text
analyzer id
analyzer version
stage
input graph version
configuration hash
started time
finished time
status: completed | partial | analysis_error
diagnostics
```

Analyzer output facts MUST reference the run record.

---

## Configuration Contract

Configuration is input, not truth.

Configuration MAY define:

```text
bounded contexts
ubiquitous language terms
package prefixes
layers
allowed imports
forbidden imports
module behavior labels
framework detectors
complexity thresholds
metric weights
query scopes
network access policy
```

Configuration-derived facts MUST include:

```text
source: config
config file
config hash
rule id
```

Current prototype configuration is loaded from `safedesign.json` at the resolved project root unless `--config` is provided. Each analyzer receives only its configured section, and analyzer run records include the section hash.

Observed facts and configured facts MUST be linked by explicit edges or source-fact references. They MUST NOT be silently merged.

---

## Use-Case Requirements

### DDD / Ubiquitous Language / Bounded Context

Required analyzers:

```text
ddd_classification
policy_evaluation
```

Required configuration:

```text
bounded contexts
context package prefixes or matching rules
layer rules
allowed cross-context dependencies
shared kernel rules
```

Required outputs:

```text
ddd.context labels
ddd.layer labels
ddd.role labels
policy_result for each configured boundary rule
warning for each failed rule
```

### Complexity Metrics

Required analyzers:

```text
complexity_metrics
policy_evaluation, when thresholds are configured
```

Required default metrics:

```text
cyclomatic_complexity
decision_count
runtime_marker_count
```

Current prototype metric contract:

```text
name: cyclomatic_complexity
unit: count
scope: function_or_method
baseline: 1
status: pass when value <= warningThreshold, warning when value > warningThreshold
evidence: measured function or method node ID
```

Current prototype counting definition:

```text
+1 for each if
+1 for each for
+1 for each range
+1 for each switch case clause
+1 for each switch default clause, configurable
+1 for each select case clause
+1 for each select default clause, configurable
+1 for each && and || operator, configurable
+1 for each goto statement, configurable and disabled by default
```

Current prototype limitations:

```text
unsaved editor overlays are not included
cognitive complexity is not emitted
package and module aggregate metrics are not emitted
domain complexity query scope means package plus direct imports only
```

Required configurable weights:

```text
if
for
range
switch
case
select
boolean_operator
go_statement
defer_statement
channel_send
channel_receive
```

### Package and Dependency Policy

Required analyzers:

```text
module_dependency_enrichment
policy_evaluation
```

Required checks:

```text
import_cycle
allowed_import
forbidden_import
missing_dependency
outdated_module, only when module metadata is available
```

### Third-Party Behavior Labels

Required analyzer:

```text
third_party_behavior_labeling
```

Required label namespaces:

```text
io.network
io.database
io.filesystem
io.queue
io.cache
telemetry
crypto
auth
framework
orm
```

### Framework Structure

Required analyzer:

```text
framework_extraction
```

Required output kinds when supported by a framework extractor:

```text
route
endpoint
framework.role=controller
framework.role=model
framework.role=repository
framework.role=middleware
framework.role=job
```

Unsupported framework means `unknown`.

---

## Reuse Rule

New use cases MUST be added by:

```text
new analyzer
new configuration section
new labels, metrics, warnings, policy results, or query views
```

New use cases MUST NOT be added by:

```text
forking the graph schema
creating private AST walkers with private output formats
creating one-off JSON shapes for each viewer
claiming semantic proof from convention-only inference
using "not found" as "does not exist"
```
