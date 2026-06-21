# introduction
I'm experimenting how to do drive software design in golang that is not bottlenecked by code review. Meaning code review I deem extremely important, but how to enable it so it more easier when the code is being produced at a very high  rate.

## status
- alpha and experimental

## dogfood
Run go-safedesign against this module with its own `safedesign.json`:

```sh
go run ./src --path . --json
```

By default, module discovery is bounded to the nearest `go.mod` root. Use `--workspace-root` when you intentionally want to scan a multi-module workspace.
For a compact agent-facing report of only diagnostics, non-pass policy/query facts, warning metrics, and non-completed runs:

```sh
go run ./src --path . --problems
```

# some goals: safe design tool
1. cyclomatic complexity analyzer that you can adjust
2. uses the language used by the codebase and API as baseline for quick judgement.
3. need to works on Linux with a large git codebase
4. common checks: cyclic import, allowed import, outdated modules
5. make zero assumption of project folder structure and friendly to live development workflow: 
   - can start from go.mod.
   - arbitrary file and folder entry.
   - live changes.  
   - does not block code editor or git operation
6. integrates with go fix and go ecosystem
7. co-tool with agents and humans
