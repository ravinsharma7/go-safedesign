# prior works
1. ast-grep behaves with prior knowledge about the codebase. our tool should not and cannot make that assumption. a "no" from the our tool should mean "it does not exists", not "not found".
2. go-ast supports this we should use this, the output is ast, i guess it is fine we don't need the concrete syntax. but i don't know if it possible to arbitrarily pass files and function to it and have it build it. if it does not we need to figure out how we can do it "incrementally" without hijacking a lot of things(we don't know to maintain a lot of things ourself.)'. ast structure might suddenly change, so this code should be treated as third party dependency.
3. goroutines, defer, channels, select and other runtimes behaviours we just ignore for now, but we record how many times it appears.
4. to prevent confusion we need to be able to say "project" diff "what is given to the tool".

# some initial design
1. watching fileset changes via linux event pool api, no naive pooling.
2. adjust those fileset changes being watched.
3. parsing a golang file.
4. parsing a golang file and then "passing" it at runtime.
5. read and parse the go.mod
6. choose a syntax or format to output the dependency graph. graphml?
7. ability to pause a "path" without breaking and can gracefully continue if required with new files or process a path without knowing what is preceded.
8. we cannot do validation on our own, not sure if we should be doing that or using golang official tool, but we are processing it in out of order, not sure how this will work.

# packages used
- single file AST        → go/parser
- real package/project   → go/packages
- custom analyzer        → go/analysis
- call graph / SSA later → x/tools/go/ssa, go/callgraph

## usable
- go/packages.Load

## deprecated
- parser.ParseDir



```
Rule 1:
domain must not import infra

Rule 2:
domain must not import HTTP, SQL, ORM, queue, framework code

Rule 3:
application may import domain

Rule 4:
infra may import application ports or domain interfaces

Rule 5:
bounded context A must not import bounded context B directly

Rule 6:
shared kernel must stay small

Rule 7:
domain events may cross context boundary, entities should not

∀ f ∈ Domain(ctx):
  imports(f) ⊆ Domain(ctx) ∪ SharedKernel ∪ StdLib

∀ f ∈ Domain(ctx):
  imports(f) ∩ {ORM, HTTP, DB, Queue, Framework} = ∅

∀ ctxA, ctxB:
  ctxA ≠ ctxB → direct_import(ctxA, ctxB) = forbidden
```
