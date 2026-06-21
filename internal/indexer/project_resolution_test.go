package indexer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ravinsharma7/go-safedesign/internal/core"
)

func TestResolveProjectDefaultsToNearestModuleRoot(t *testing.T) {
	project, err := ResolveProject(Options{Path: filepath.Join("..", "..", "testdata", "workspace", "shop", "order", "service.go")})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(project.Root) != "shop" {
		t.Fatalf("root = %s, want nearest shop module", project.Root)
	}
	if project.WorkspaceRoot != project.Root || project.SourceBase != project.Root {
		t.Fatalf("project = %#v, want default workspace/source base at root", project)
	}
}

func TestBuildGraphFromNestedDirectoryPath(t *testing.T) {
	graph, err := BuildGraph(Options{Path: filepath.Join("..", "..", "testdata", "workspace", "shop", "order")})
	if err != nil {
		t.Fatal(err)
	}
	if !hasNodeForTest(graph.Nodes, "module:example.com/shop") {
		t.Fatalf("nodes = %#v, missing shop module", graph.Nodes)
	}
	if !hasNodeForTest(graph.Nodes, "file:order/service.go") {
		t.Fatalf("nodes = %#v, missing source-base-relative order file", graph.Nodes)
	}
	if !hasNodeForTest(graph.Nodes, "module:example.com/payments") {
		t.Fatalf("nodes = %#v, local replace module should be included even without explicit workspace root", graph.Nodes)
	}
	if !hasSourceRecordForTest(graph.SourceRecords, "user_entry", "order") || !hasSourceRecordForTest(graph.SourceRecords, "project_root", ".") {
		t.Fatalf("source records = %#v, missing entry/root discovery", graph.SourceRecords)
	}
}

func TestWorkspaceRootScansSiblingModulesAndReconcilesModuleDependencies(t *testing.T) {
	graph, err := BuildGraph(Options{
		Path:          filepath.Join("..", "..", "testdata", "workspace", "shop"),
		WorkspaceRoot: filepath.Join("..", "..", "testdata", "workspace"),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{
		"module:example.com/shop",
		"module:example.com/payments",
		"module:example.com/missing-notification",
	} {
		if !hasNodeForTest(graph.Nodes, id) {
			t.Fatalf("nodes = %#v, missing %s", graph.Nodes, id)
		}
	}
	for _, id := range []string{
		"edge:depends_on:module:example.com/shop->module:example.com/payments",
		"edge:depends_on:module:example.com/shop->module:example.com/missing-notification",
	} {
		edge := edgeByIDForTest(graph.Edges, id)
		if edge == nil || !edge.Complete || edge.Synthetic || strings.HasPrefix(edge.To, core.IDPrefixPlaceholder) {
			t.Fatalf("edge %s = %#v, want real complete module dependency", id, edge)
		}
	}
}

func TestMissingRequiredModuleRemainsPlaceholderDependency(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, filepath.Join(dir, "app", "go.mod"), "module example.com/app\n\ngo 1.22\n\nrequire example.com/missing v0.0.0\n")
	writeFileForTest(t, filepath.Join(dir, "app", "main.go"), "package main\n\nfunc main() {}\n")

	graph, err := BuildGraph(Options{Path: filepath.Join(dir, "app"), WorkspaceRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !hasNodeForTest(graph.Nodes, core.PlaceholderModuleID("example.com/missing")) {
		t.Fatalf("nodes = %#v, missing placeholder module", graph.Nodes)
	}
	edge := edgeByIDForTest(graph.Edges, core.EdgeID(core.EdgeKindDependsOn, core.ModuleID("example.com/app"), core.PlaceholderModuleID("example.com/missing")))
	if edge == nil || edge.Complete || !edge.Synthetic || edge.Reason != "required_module_not_present_in_workspace" {
		t.Fatalf("edge = %#v, want incomplete placeholder module dependency", edge)
	}
}

func TestGoWorkUseDiscoversListedModules(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, filepath.Join(dir, "app", "go.mod"), "module example.com/app\n\ngo 1.22\n")
	writeFileForTest(t, filepath.Join(dir, "app", "main.go"), "package main\n\nfunc main() {}\n")
	writeFileForTest(t, filepath.Join(dir, "worker", "go.mod"), "module example.com/worker\n\ngo 1.22\n")
	writeFileForTest(t, filepath.Join(dir, "worker", "worker.go"), "package worker\n\nfunc Run() {}\n")
	writeFileForTest(t, filepath.Join(dir, "go.work"), "go 1.22\n\nuse (\n\t./app\n\t./worker\n)\n")

	graph, err := BuildGraph(Options{Path: filepath.Join(dir, "app"), WorkspaceRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	if !hasNodeForTest(graph.Nodes, core.ModuleID("example.com/worker")) {
		t.Fatalf("nodes = %#v, missing go.work use module", graph.Nodes)
	}
	worker := nodeByIDForTest(graph.Nodes, core.ModuleID("example.com/worker"))
	if worker == nil || !strings.Contains(worker.Reason, "go_work_use") {
		t.Fatalf("worker module = %#v, want go_work_use discovery reason", worker)
	}
}

func TestGoModReplaceDiscoversLocalModuleOutsideWorkspace(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, filepath.Join(dir, "app", "go.mod"), "module example.com/app\n\ngo 1.22\n\nrequire example.com/local v0.0.0\n\nreplace example.com/local => ../local\n")
	writeFileForTest(t, filepath.Join(dir, "app", "main.go"), "package main\n\nimport \"example.com/local/pkg\"\n\nfunc main() { pkg.Run() }\n")
	writeFileForTest(t, filepath.Join(dir, "local", "go.mod"), "module example.com/local\n\ngo 1.22\n")
	writeFileForTest(t, filepath.Join(dir, "local", "pkg", "pkg.go"), "package pkg\n\nfunc Run() {}\n")

	graph, err := BuildGraph(Options{Path: filepath.Join(dir, "app")})
	if err != nil {
		t.Fatal(err)
	}
	local := nodeByIDForTest(graph.Nodes, core.ModuleID("example.com/local"))
	if local == nil || !strings.Contains(local.Reason, "go_mod_replace") {
		t.Fatalf("local module = %#v, want replace-discovered module outside default root", local)
	}
	edge := edgeByIDForTest(graph.Edges, core.EdgeID(core.EdgeKindDependsOn, core.ModuleID("example.com/app"), core.ModuleID("example.com/local")))
	if edge == nil || !edge.Complete || edge.Synthetic {
		t.Fatalf("edge = %#v, want complete dependency to replace-discovered module", edge)
	}
}

func TestReplacePathWithoutGoModEmitsDiagnosticAndPlaceholder(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, filepath.Join(dir, "app", "go.mod"), "module example.com/app\n\ngo 1.22\n\nrequire example.com/bad v0.0.0\n\nreplace example.com/bad => ../bad\n")
	writeFileForTest(t, filepath.Join(dir, "app", "main.go"), "package main\n\nfunc main() {}\n")
	if err := os.MkdirAll(filepath.Join(dir, "bad"), 0o755); err != nil {
		t.Fatal(err)
	}

	graph, err := BuildGraph(Options{Path: filepath.Join(dir, "app")})
	if err != nil {
		t.Fatal(err)
	}
	if !hasNodeForTest(graph.Nodes, core.PlaceholderModuleID("example.com/bad")) {
		t.Fatalf("nodes = %#v, missing placeholder for bad replace module", graph.Nodes)
	}
	if !hasDiagnosticReasonForTest(graph.Diagnostics, "local replace path has no go.mod") {
		t.Fatalf("diagnostics = %#v, missing local replace diagnostic", graph.Diagnostics)
	}
}

func TestDuplicateModuleDiscoveryReasonsAreMerged(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, filepath.Join(dir, "app", "go.mod"), "module example.com/app\n\ngo 1.22\n\nrequire example.com/lib v0.0.0\n\nreplace example.com/lib => ../lib\n")
	writeFileForTest(t, filepath.Join(dir, "app", "main.go"), "package main\n\nfunc main() {}\n")
	writeFileForTest(t, filepath.Join(dir, "lib", "go.mod"), "module example.com/lib\n\ngo 1.22\n")
	writeFileForTest(t, filepath.Join(dir, "lib", "lib.go"), "package lib\n")
	writeFileForTest(t, filepath.Join(dir, "go.work"), "go 1.22\n\nuse (\n\t./app\n\t./lib\n)\n")

	graph, err := BuildGraph(Options{Path: filepath.Join(dir, "app"), WorkspaceRoot: dir})
	if err != nil {
		t.Fatal(err)
	}
	lib := nodeByIDForTest(graph.Nodes, core.ModuleID("example.com/lib"))
	if lib == nil || !strings.Contains(lib.Reason, "workspace_scan") || !strings.Contains(lib.Reason, "go_work_use") || !strings.Contains(lib.Reason, "go_mod_replace") {
		t.Fatalf("lib module = %#v, want merged discovery reasons", lib)
	}
	if countNodesForTest(graph.Nodes, core.ModuleID("example.com/lib")) != 1 {
		t.Fatalf("nodes = %#v, lib module should be indexed once", graph.Nodes)
	}
}

func TestSourceDiscoveryFromFileEntryWithConfigAndGoWork(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, filepath.Join(dir, "go.mod"), "module example.com/source-discovery\n\ngo 1.22\n")
	writeFileForTest(t, filepath.Join(dir, "go.work"), "go 1.22\n\nuse .\n")
	writeFileForTest(t, filepath.Join(dir, "safedesign.config.json"), `{"complexity":{"cyclomatic":{"warningThreshold":99}}}`)
	writeFileForTest(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n")

	graph, err := BuildGraph(Options{Path: filepath.Join(dir, "main.go")})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []struct {
		kind string
		path string
	}{
		{"user_entry", "main.go"},
		{"project_root", "."},
		{"workspace_root", "."},
		{"config", "safedesign.config.json"},
		{"go_work", "go.work"},
		{"go_mod", "go.mod"},
		{"go_file", "main.go"},
	} {
		if !hasSourceRecordForTest(graph.SourceRecords, want.kind, want.path) {
			t.Fatalf("source records = %#v, missing %s %s", graph.SourceRecords, want.kind, want.path)
		}
	}
	if run := runByStageForTest(graph.Runs, "source_discovery"); run == nil || run.RunID == "" || run.StartedAt == "" || run.FinishedAt == "" {
		t.Fatalf("source discovery run = %#v, want enriched run metadata", run)
	}
}

func TestSourceDiscoveryAllowsMissingExplicitConfig(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, filepath.Join(dir, "go.mod"), "module example.com/missing-config\n\ngo 1.22\n")
	writeFileForTest(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n")

	graph, err := BuildGraph(Options{Path: dir, ConfigPath: filepath.Join(dir, "missing-safedesign.config.json")})
	if err != nil {
		t.Fatal(err)
	}
	if hasSourceRecordForTest(graph.SourceRecords, "config", "missing-safedesign.config.json") {
		t.Fatalf("source records = %#v, missing config should not be recorded as present", graph.SourceRecords)
	}
}

func TestSyntaxExtractionEmitsTypeInterfaceStructAndFieldNodes(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, filepath.Join(dir, "go.mod"), "module example.com/syntax\n\ngo 1.22\n")
	writeFileForTest(t, filepath.Join(dir, "model.go"), `package syntax

type Alias string

type PaymentGateway interface {
	Charge(id string) bool
}

type OrderRequest struct {
	OrderID string
	Amount int
}
`)

	graph, err := BuildGraph(Options{Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{
		"type:example.com/syntax.Alias",
		"interface:example.com/syntax.PaymentGateway",
		"struct:example.com/syntax.OrderRequest",
		"field:example.com/syntax.OrderRequest.OrderID",
		"field:example.com/syntax.OrderRequest.Amount",
	} {
		if !hasNodeForTest(graph.Nodes, id) {
			t.Fatalf("nodes = %#v, missing %s", graph.Nodes, id)
		}
	}
	if !hasEdgeForTest(graph.Edges, "edge:declares:package:example.com/syntax->struct:example.com/syntax.OrderRequest") {
		t.Fatalf("edges = %#v, missing package declares struct edge", graph.Edges)
	}
	if !hasEdgeForTest(graph.Edges, "edge:contains:struct:example.com/syntax.OrderRequest->field:example.com/syntax.OrderRequest.OrderID") {
		t.Fatalf("edges = %#v, missing struct contains field edge", graph.Edges)
	}
}

func TestResolveProjectRequiresGoMod(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveProject(Options{Path: path})
	if err == nil || !strings.Contains(err.Error(), "no go.mod found") {
		t.Fatalf("error = %v, want no go.mod error", err)
	}
}

func hasNodeForTest(nodes []Node, id string) bool {
	for _, node := range nodes {
		if node.ID == id {
			return true
		}
	}
	return false
}

func nodeByIDForTest(nodes []Node, id string) *Node {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}

func countNodesForTest(nodes []Node, id string) int {
	count := 0
	for _, node := range nodes {
		if node.ID == id {
			count++
		}
	}
	return count
}

func hasEdgeForTest(edges []Edge, id string) bool {
	for _, edge := range edges {
		if edge.ID == id {
			return true
		}
	}
	return false
}

func edgeByIDForTest(edges []Edge, id string) *Edge {
	for i := range edges {
		if edges[i].ID == id {
			return &edges[i]
		}
	}
	return nil
}

func hasSourceRecordForTest(records []core.SourceRecord, kind, path string) bool {
	for _, record := range records {
		if record.Kind == kind && record.Path == path && record.RunID != "" {
			return true
		}
	}
	return false
}

func hasDiagnosticReasonForTest(diagnostics []Diagnostic, reason string) bool {
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Reason, reason) {
			return true
		}
	}
	return false
}

func runByStageForTest(runs []core.RunRecord, stage string) *core.RunRecord {
	for i := range runs {
		if runs[i].Stage == stage {
			return &runs[i]
		}
	}
	return nil
}

func writeFileForTest(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
