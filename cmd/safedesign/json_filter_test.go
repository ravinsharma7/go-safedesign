package main

import (
	"testing"

	"github.com/ravinsharma7/go-safedesign/internal/core"
)

func TestFilterGraphJSONNoOptionsReturnsFullGraph(t *testing.T) {
	graph := jsonFilterFixture()

	filtered, err := filterGraphJSON(graph, graphJSONFilterOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if len(filtered.Nodes) != len(graph.Nodes) || len(filtered.Edges) != len(graph.Edges) || len(filtered.Observations) != len(graph.Observations) {
		t.Fatalf("unexpected filtered graph: %#v", filtered)
	}
	if len(filtered.Diagnostics) != len(graph.Diagnostics) || len(filtered.Runs) != len(graph.Runs) {
		t.Fatalf("expected non-kind sections to pass through: %#v", filtered)
	}
}

func TestFilterGraphJSONSections(t *testing.T) {
	graph := jsonFilterFixture()

	filtered, err := filterGraphJSON(graph, graphJSONFilterOptions{Sections: []string{"nodes", "observations"}})
	if err != nil {
		t.Fatal(err)
	}

	if len(filtered.Nodes) != 2 || len(filtered.Observations) != 2 {
		t.Fatalf("expected selected sections to remain: %#v", filtered)
	}
	if len(filtered.Edges) != 0 || len(filtered.Diagnostics) != 0 || len(filtered.Runs) != 0 {
		t.Fatalf("expected unselected sections to be empty: %#v", filtered)
	}
}

func TestFilterGraphJSONNodeAndEdgeKinds(t *testing.T) {
	graph := jsonFilterFixture()

	filtered, err := filterGraphJSON(graph, graphJSONFilterOptions{
		NodeKinds: []string{core.NodeKindPackage},
		EdgeKinds: []string{core.EdgeKindImports},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(filtered.Nodes) != 1 || filtered.Nodes[0].Kind != core.NodeKindPackage {
		t.Fatalf("expected only package nodes: %#v", filtered.Nodes)
	}
	if len(filtered.Edges) != 1 || filtered.Edges[0].Kind != core.EdgeKindImports {
		t.Fatalf("expected only import edges: %#v", filtered.Edges)
	}
	if len(filtered.Observations) != 2 {
		t.Fatalf("observation section should remain unfiltered without observation-name filter: %#v", filtered.Observations)
	}
}

func TestFilterGraphJSONObservationNames(t *testing.T) {
	graph := jsonFilterFixture()

	filtered, err := filterGraphJSON(graph, graphJSONFilterOptions{
		Sections:         []string{"observations"},
		ObservationNames: []string{core.ObservationNameLanguageZoneCandidate},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(filtered.Observations) != 1 {
		t.Fatalf("expected one observation: %#v", filtered.Observations)
	}
	if filtered.Observations[0].Name != core.ObservationNameLanguageZoneCandidate {
		t.Fatalf("unexpected observation: %#v", filtered.Observations[0])
	}
	if len(filtered.Nodes) != 0 || len(filtered.Edges) != 0 {
		t.Fatalf("expected only observations section: %#v", filtered)
	}
}

func TestFilterGraphJSONRejectsUnknownSection(t *testing.T) {
	_, err := filterGraphJSON(jsonFilterFixture(), graphJSONFilterOptions{Sections: []string{"facts"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func jsonFilterFixture() core.Graph {
	return core.Graph{
		Nodes: []core.Node{
			{ID: "package:example.com/app", Kind: core.NodeKindPackage},
			{ID: "file:main.go", Kind: core.NodeKindFile},
		},
		Edges: []core.Edge{
			{ID: "edge:imports:one", Kind: core.EdgeKindImports},
			{ID: "edge:contains:one", Kind: core.EdgeKindContains},
		},
		SourceRecords: []core.SourceRecord{{ID: "source:one", Kind: "go_file"}},
		Observations: []core.Observation{
			{ID: "observation:term", Name: core.ObservationNameVocabularyTerm},
			{ID: "observation:candidate", Name: core.ObservationNameLanguageZoneCandidate},
		},
		Labels:        []core.Label{{ID: "label:one"}},
		Warnings:      []core.Warning{{ID: "warning:one"}},
		Queries:       []core.QueryResult{{ID: "query:one"}},
		PathJobs:      []core.PathJob{{ID: "path:one"}},
		PolicyResults: []core.PolicyResult{{ID: "policy:one"}},
		Metrics:       []core.Metric{{ID: "metric:one"}},
		Freshness:     []core.Freshness{{FactID: "fact:one"}},
		Diagnostics:   []core.Diagnostic{{Reason: "diagnostic"}},
		Runs:          []core.RunRecord{{AnalyzerID: "analyzer"}},
	}
}
