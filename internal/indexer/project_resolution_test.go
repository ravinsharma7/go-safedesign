package indexer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-safedesign/internal/core"
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
	if hasNodeForTest(graph.Nodes, "module:example.com/payments") {
		t.Fatalf("nodes = %#v, default project should not scan sibling modules", graph.Nodes)
	}
	if !hasSourceRecordForTest(graph.SourceRecords, "user_entry", "order") || !hasSourceRecordForTest(graph.SourceRecords, "project_root", ".") {
		t.Fatalf("source records = %#v, missing entry/root discovery", graph.SourceRecords)
	}
}

func TestSourceDiscoveryFromFileEntryWithConfigAndGoWork(t *testing.T) {
	dir := t.TempDir()
	writeFileForTest(t, filepath.Join(dir, "go.mod"), "module example.com/source-discovery\n\ngo 1.22\n")
	writeFileForTest(t, filepath.Join(dir, "go.work"), "go 1.22\n\nuse .\n")
	writeFileForTest(t, filepath.Join(dir, "safedesign.json"), `{"complexity":{"cyclomatic":{"warningThreshold":99}}}`)
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
		{"config", "safedesign.json"},
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

	graph, err := BuildGraph(Options{Path: dir, ConfigPath: filepath.Join(dir, "missing-safedesign.json")})
	if err != nil {
		t.Fatal(err)
	}
	if hasSourceRecordForTest(graph.SourceRecords, "config", "missing-safedesign.json") {
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

func hasEdgeForTest(edges []Edge, id string) bool {
	for _, edge := range edges {
		if edge.ID == id {
			return true
		}
	}
	return false
}

func hasSourceRecordForTest(records []core.SourceRecord, kind, path string) bool {
	for _, record := range records {
		if record.Kind == kind && record.Path == path && record.RunID != "" {
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
