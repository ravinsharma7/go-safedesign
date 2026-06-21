package viewer

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"go-safedesign/internal/core"
)

func TestHandlerServesEmbeddedHTML(t *testing.T) {
	server := httptest.NewServer(Handler(core.Graph{}, filepath.Join("..", "..", "testdata", "workspace", "shop")))
	defer server.Close()

	body := getBody(t, server.URL+"/")
	if !strings.Contains(body, "Graph first") || !strings.Contains(body, "Code first") {
		t.Fatalf("viewer HTML missing navigation modes: %s", body[:min(len(body), 200)])
	}
	if !strings.Contains(body, "policyResults") || !strings.Contains(body, "Policy: ") || !strings.Contains(body, "Policy Status") {
		t.Fatalf("viewer HTML missing policy result hooks")
	}
	if !strings.Contains(body, "policy pass") || !strings.Contains(body, "graphSelectionID") {
		t.Fatalf("viewer HTML missing policy navigation helpers")
	}
	if !strings.Contains(body, "metrics") || !strings.Contains(body, "Metric Status") || !strings.Contains(body, "complexity warning") {
		t.Fatalf("viewer HTML missing metric hooks")
	}
}

func TestHandlerGraphJSON(t *testing.T) {
	graph := core.Graph{
		Nodes:         []core.Node{{ID: "node:a"}},
		SourceRecords: []core.SourceRecord{{ID: "source_record:a", Kind: "go_file"}},
		Labels:        []core.Label{{ID: "label:a"}},
		Warnings:      []core.Warning{{ID: "warning:a"}},
		PolicyResults: []core.PolicyResult{{ID: "policy_result:a", Status: "pass"}},
		Metrics:       []core.Metric{{ID: "metric:a", Status: "pass"}},
	}
	server := httptest.NewServer(Handler(graph, filepath.Join("..", "..", "testdata", "workspace", "shop")))
	defer server.Close()

	body := getBody(t, server.URL+"/graph.json")
	var got core.Graph
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Nodes) != 1 || got.Nodes[0].ID != "node:a" || len(got.SourceRecords) != 1 || len(got.Labels) != 1 || len(got.Warnings) != 1 || len(got.PolicyResults) != 1 || len(got.Metrics) != 1 {
		t.Fatalf("graph = %#v", got)
	}
}

func TestHandlerSource(t *testing.T) {
	server := httptest.NewServer(Handler(core.Graph{}, filepath.Join("..", "..", "testdata", "workspace", "shop")))
	defer server.Close()

	body := getBody(t, server.URL+"/source?file=order/service.go&range=9:1-17:2")
	if !strings.Contains(body, "PlaceOrder") {
		t.Fatalf("source response missing PlaceOrder: %s", body)
	}
}

func getBody(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
