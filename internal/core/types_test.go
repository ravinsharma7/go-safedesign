package core

import "testing"

func TestSortGraph(t *testing.T) {
	graph := SortGraph(Graph{
		Nodes:         []Node{{ID: "node:z"}, {ID: "node:a"}},
		Edges:         []Edge{{ID: "edge:z"}, {ID: "edge:a"}},
		SourceRecords: []SourceRecord{{ID: "source_record:z"}, {ID: "source_record:a"}},
		Observations:  []Observation{{ID: "observation:z"}, {ID: "observation:a"}},
		Labels:        []Label{{ID: "label:z"}, {ID: "label:a"}},
		Warnings:      []Warning{{ID: "warning:z"}, {ID: "warning:a"}},
		Diagnostics: []Diagnostic{
			{Source: "b", Reason: "2"},
			{Source: "a", Reason: "2"},
			{Source: "a", Reason: "1"},
		},
		PolicyResults: []PolicyResult{{ID: "policy_result:z"}, {ID: "policy_result:a"}},
		Metrics:       []Metric{{ID: "metric:z"}, {ID: "metric:a"}},
		Runs:          []RunRecord{{Stage: "module_extraction"}},
	})
	if graph.Nodes[0].ID != "node:a" || graph.Edges[0].ID != "edge:a" {
		t.Fatalf("graph not sorted: %#v", graph)
	}
	if graph.Diagnostics[0].Reason != "1" || graph.Diagnostics[1].Source != "a" {
		t.Fatalf("diagnostics not sorted: %#v", graph.Diagnostics)
	}
	if graph.PolicyResults[0].ID != "policy_result:a" {
		t.Fatalf("policy results not sorted: %#v", graph.PolicyResults)
	}
	if graph.Metrics[0].ID != "metric:a" {
		t.Fatalf("metrics not sorted: %#v", graph.Metrics)
	}
	if graph.SourceRecords[0].ID != "source_record:a" || graph.Labels[0].ID != "label:a" || graph.Warnings[0].ID != "warning:a" {
		t.Fatalf("new fact collections not sorted: %#v %#v %#v", graph.SourceRecords, graph.Labels, graph.Warnings)
	}
	if graph.Observations[0].ID != "observation:a" {
		t.Fatalf("observations not sorted: %#v", graph.Observations)
	}
	if len(graph.Runs) != 1 || graph.Runs[0].Stage != "module_extraction" {
		t.Fatalf("runs not preserved: %#v", graph.Runs)
	}
}
