# introduction
I'm experimenting how to do drive software design in golang that is not bottlenecked by code review. I deem code review as important, but there seem to be some bad impression that code review is a "bottleneck" when using agents to create code at a very fast pace. 
This tool enables human(or agent) reviewer to get a better picture of code and intention quickly.

## status
- alpha and experimental. api, command, output may suddenly change.

## some goals: safe design tool
1. cyclomatic complexity analyzer that you can adjust
2. uses the language used by the codebase and API as baseline for quick judgement.
3. need to works on Linux with a large git codebase
4. common checks: cyclic import, allowed import, outdated modules, etc
5. make zero assumption of project folder structure and friendly to live development workflow: 
   - can start from arbitary go.mod definition.
   - arbitrary file and folder entry.
   - live changes during development.  
   - does not block code editor or git operation.
6. integrates with go fix and go ecosystem
7. co-tool with agents and humans. viewer to see codebase structure and architecture.
8. extension for more domain specific checks and analysis.

## go tool usage
Register go-safedesign as a module-defined Go tool from the default branch:

```sh
go get -tool github.com/ravinsharma7/go-safedesign/cmd/safedesign@main
```

Run the registered tool from the consuming module:

```sh
go tool safedesign --path . --problems
go tool safedesign --path . --ddd-report
go tool safedesign --list-values --stdout
```

Update to the latest default-branch commit by running the same registration command again:

```sh
go get -tool github.com/ravinsharma7/go-safedesign/cmd/safedesign@main
```

Check the currently selected version and compare it with the current default-branch pseudo-version:

```sh
go list -m -json github.com/ravinsharma7/go-safedesign
go list -m -json github.com/ravinsharma7/go-safedesign@main
```

If the `@main` pseudo-version is newer, run `go get -tool github.com/ravinsharma7/go-safedesign/cmd/safedesign@main` to update the selected tool version.

Remove the registered tool from the consuming module:

```sh
go get -tool github.com/ravinsharma7/go-safedesign/cmd/safedesign@none
go mod tidy
```

## alpha branch and tag release process
Use this process when publishing an alpha snapshot branch such as `alpha-0.1.0` from `dev`.

Commit the release state on `dev`, then point both the branch and annotated tag at that commit:

```sh
git switch dev
git status --short
git add -A
git commit -m "add go tool deployment support"
release_commit=$(git rev-parse HEAD)
git tag -f -a alpha-0.1.0 -m "alpha-0.1.0" "$release_commit"
git branch -f alpha-0.1.0 "$release_commit"
```

Push with fully qualified refs because the branch and tag intentionally share the same name:

```sh
git push -u origin refs/heads/dev:refs/heads/dev
git push --force-with-lease origin refs/heads/alpha-0.1.0:refs/heads/alpha-0.1.0
git push --force origin refs/tags/alpha-0.1.0:refs/tags/alpha-0.1.0
```

Verify the branch and tag resolve to the same commit:

```sh
git rev-parse refs/heads/alpha-0.1.0
git rev-parse 'refs/tags/alpha-0.1.0^{}'
git ls-remote --symref origin HEAD refs/heads/alpha-0.1.0 refs/tags/alpha-0.1.0 'refs/tags/alpha-0.1.0^{}'
```

Set the GitHub repository default branch to the latest `alpha-*` branch in repository settings, or with an owner/admin token:

```sh
curl -fsS -X PATCH \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  https://api.github.com/repos/ravinsharma7/go-safedesign \
  -d '{"default_branch":"alpha-0.1.0"}'
```

## quick commands
Run go-safedesign against this module with its own `safedesign.config.json`:

```sh
go run ./cmd/safedesign --list-values
go run ./cmd/safedesign --path . --json
go run ./cmd/safedesign --path . --json --json-sections nodes,edges
go run ./cmd/safedesign --path . --json --json-sections observations --json-observation-names vocabulary.language_zone_candidate
go run ./cmd/safedesign --path . --json --output-dir /tmp/go-safedesign-report
```

By default, JSON output modes write files under `tmp/report`: `values.json`, `graph.json`, `stats.json`, `ddd-report.json`, or `problems.json`. Use `--stdout` when piping to `jq` or another command. Use `--output-dir <path>` to choose a different report directory.

By default, module discovery is bounded to the nearest `go.mod` root. Use `--workspace-root` when you intentionally want to scan sibling modules, `go.work use` entries, and local `go.mod replace` targets.

For a compact DDD evidence report over language-zone candidates, bridge evidence, incomplete dependency evidence, and supporting stats:

```sh
go run ./cmd/safedesign --path . --ddd-report
go run ./cmd/safedesign --path . --ddd-report --scope-package github.com/ravinsharma7/go-safedesign/internal/indexer
```

For a compact agent-facing report of only diagnostics, non-pass policy/query facts, warning metrics, and non-completed runs:

```sh
go run ./cmd/safedesign --path . --problems
```

For scoped graph inventory counts:

```sh
go run ./cmd/safedesign --path . --stats
go run ./cmd/safedesign --path . --stats --stdout | jq '.workspace'
go run ./cmd/safedesign --path . --stats --stdout | jq '.modules[] | {modulePath, dir, discoveryReasons, packages, files, dependsOn, missingDependencies}'
go run ./cmd/safedesign --path . --stats --stdout | jq '.packages[] | {packagePath, modulePath}'
go run ./cmd/safedesign --path . --stats --scope-module github.com/ravinsharma7/go-safedesign
go run ./cmd/safedesign --path . --stats --scope-file internal/indexer/builder.go
```

Use `.modules[]` to inspect module inventory. Use `.packages[]` to inspect packages grouped by their `modulePath`; package output will not show modules that have no discovered packages. To scan sibling modules, pass `--workspace-root`; otherwise `--path` is bounded to the nearest `go.mod`.

For a multi-module workspace:

```sh
go run ./cmd/safedesign --path testdata/workspace/shop --workspace-root testdata/workspace --stats
go run ./cmd/safedesign --path testdata/workspace/shop --workspace-root testdata/workspace --stats --scope-module example.com/shop
go run ./cmd/safedesign --path testdata/workspace/shop --workspace-root testdata/workspace --stats --stdout | jq '.modules[] | {modulePath, dir, discoveryReasons, packages, files, dependsOn, dependedOnBy, missingDependencies}'
```

Analyzer execution can be narrowed for faster exploration. Positive selection expands required prerequisites; skip selection also removes dependent analyzers that cannot run safely.

```sh
go run ./cmd/safedesign --path . --ddd-report --analyzers language_zone_candidate
go run ./cmd/safedesign --path . --problems --skip-analyzers complexity
```

For other Go projects, point `--path` at the project root, module root, package directory, or a Go file. Use `--workspace-root` for monorepos and `--config` when the shared `safedesign.config.json` lives outside the analyzed root.

```sh
go run ./cmd/safedesign --path /path/to/project --ddd-report
go run ./cmd/safedesign --path /path/to/project --problems
go run ./cmd/safedesign --path /path/to/project --json
go run ./cmd/safedesign --path /path/to/project --json --stdout > safedesign-graph.json
go run ./cmd/safedesign --path /path/to/project --json --json-sections nodes --json-node-kinds module,package,file
go run ./cmd/safedesign --path /path/to/project/module-a --workspace-root /path/to/project --stats
go run ./cmd/safedesign --path /path/to/project/module-a --workspace-root /path/to/project --stats --stdout | jq '.modules[] | {modulePath, dir, discoveryReasons, packages, files, dependsOn, missingDependencies}'
go run ./cmd/safedesign --path /path/to/project/module-a --workspace-root /path/to/project --config /path/to/safedesign.config.json --ddd-report
```

The graph is one workspace graph, not one graph per module. Module boundaries are represented by module nodes, package/file `modulePath` metadata, `depends_on` edges, and module-aware report scopes. `go.work use` entries and local `go.mod replace` paths are followed when they point to a directory with `go.mod`, even outside `--workspace-root`. Required modules not discovered locally remain placeholder module nodes and appear in stats as `missingDependencies`.

Module discovery currently deduplicates discovered modules by module path and directory. If a workspace intentionally contains multiple checkouts of the same module path, such as different git commits of the same project, the report may collapse them under one module identity. That is a known identity limitation; future work should distinguish module instances by module path plus checkout directory and version or commit evidence.

## command reference
The CLI currently requires one output mode: `--json`, `--ddd-report`, `--stats`, `--problems`, `--list-values`, or `--serve`.

Input and project resolution:
- `--path <path>`: project root, module root, package directory, Go file, or nested path to analyze.
- `--fixture <path>`: deprecated alias for `--path`.
- `--workspace-root <path>`: module discovery boundary for multi-module workspaces; defaults to the resolved nearest `go.mod` root.
- `--config <path>`: shared `safedesign.config.json`; defaults to `safedesign.config.json` under the resolved project root when present.
- `--policy-config <path>`: compatibility override for dependency policy config path.

Output modes:
- `--json`: canonical graph JSON with nodes, edges, observations, labels, metrics, diagnostics, runs, and related facts.
- `--ddd-report`: compact DDD evidence report with language-zone candidates, bridge evidence, incomplete dependency evidence, and scoped stats.
- `--stats`: compact graph inventory report with overall, module, package, and file counts.
- `--problems`: compact agent-facing report containing diagnostics, non-pass policy/query facts, warning metrics, warnings, and non-completed runs.
- `--list-values`: compact JSON listing the current binary's supported JSON sections, node kinds, edge kinds, fact kinds, statuses, freshness values, observation names/sources, trust levels, and analyzer IDs. This does not require `--path`.
- `--serve <addr>`: browser viewer, for example `--serve :8080`.

Output destination:
- `--output-dir <path>`: directory for JSON output files; defaults to `tmp/report`.
- `--stdout`: write JSON to stdout instead of a report file. Use this for shell pipes and redirects.

Generated reports under `tmp/report` are ignored by Git and excluded from VS Code file watching/search by `.vscode/settings.json`. If you use a different long-lived report directory, add the same path to `files.watcherExclude` and `search.exclude`.

JSON shaping:
- Discover available filter values from the current command with `go run ./cmd/safedesign --list-values --stdout`.
- `--json-sections <sections>`: comma-separated top-level graph sections to emit, for example `nodes,edges,observations`.
- `--json-node-kinds <kinds>`: comma-separated node kinds to keep in `--json`, for example `module,package,file`.
- `--json-edge-kinds <kinds>`: comma-separated edge kinds to keep in `--json`, for example `imports,depends_on`.
- `--json-observation-names <names>`: comma-separated observation names to keep in `--json`, for example `vocabulary.term,vocabulary.language_zone_candidate`.

Report shaping:
- `--limit <n>`: maximum number of items per compact report list; applies to `--ddd-report`, `--stats` where lists are added later, and `--problems`.
- `--scope-module <module>`: limit compact report output to module-related facts, for example `--scope-module github.com/ravinsharma7/go-safedesign`.
- `--scope-package <package>`: limit compact report output to package-related facts, for example `--scope-package github.com/ravinsharma7/go-safedesign/internal/indexer`.
- `--scope-file <file>`: limit compact report output to source-file-backed facts, for example `--scope-file internal/indexer/builder.go`.

Analyzer execution:
- `--analyzers <ids>`: comma-separated analyzer IDs to run; required prerequisites are added automatically.
- `--skip-analyzers <ids>`: comma-separated analyzer IDs to skip; dependent analyzers that cannot run safely are also skipped.
- `--disable-policy`: compatibility shorthand for skipping `dependency_policy`.
- `--disable-complexity`: compatibility shorthand for skipping `complexity`.
- `--simulate-change`: include a simulated changed-file freshness record.
- Discover current analyzer IDs with `go run ./cmd/safedesign --list-values --stdout | jq '.analyzerIds'`.

Trust levels:
- Discover current trust levels, ranks, and descriptions with `go run ./cmd/safedesign --list-values --stdout | jq '.trustLevels'`.
- Higher ranks mean stronger evidence. An analyzer cannot emit facts above its declared maximum emitted trust.
- Placeholder-backed or incomplete facts should not be treated as complete conclusions, even when they carry syntax-level evidence.
