package main

import (
	"testing"

	"go-safedesign/internal/core"
)

func TestProblemsReportIncludesOnlyProblemFacts(t *testing.T) {
	graph := core.Graph{
		Diagnostics: []core.Diagnostic{{Reason: "parse failed"}},
		PolicyResults: []core.PolicyResult{
			{ID: "policy:pass", Status: "pass"},
			{ID: "policy:fail", Status: "fail"},
			{ID: "policy:unknown", Status: "unknown"},
		},
		Metrics: []core.Metric{
			{ID: "metric:pass", Status: "pass"},
			{ID: "metric:warning", Status: "warning"},
		},
		Queries: []core.QueryResult{
			{ID: "query:pass", Status: "pass"},
			{ID: "query:warning", Status: "warning"},
			{ID: "query:unknown", Status: "unknown"},
		},
		Runs: []core.RunRecord{
			{Stage: "completed", Status: "completed"},
			{Stage: "partial", Status: "partial"},
		},
	}

	report := buildProblemsReport(graph, 50)
	if len(report.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want 1", report.Diagnostics)
	}
	if got := ids(report.PolicyResults, func(item core.PolicyResult) string { return item.ID }); !same(got, []string{"policy:fail", "policy:unknown"}) {
		t.Fatalf("policy results = %#v", got)
	}
	if got := ids(report.Metrics, func(item core.Metric) string { return item.ID }); !same(got, []string{"metric:warning"}) {
		t.Fatalf("metrics = %#v", got)
	}
	if got := ids(report.Queries, func(item core.QueryResult) string { return item.ID }); !same(got, []string{"query:warning", "query:unknown"}) {
		t.Fatalf("queries = %#v", got)
	}
	if got := ids(report.Runs, func(item core.RunRecord) string { return item.Stage }); !same(got, []string{"partial"}) {
		t.Fatalf("runs = %#v", got)
	}
	if report.Summary.PolicyResults != 2 || report.Summary.Metrics != 1 || report.Summary.Queries != 2 || report.Summary.Runs != 1 {
		t.Fatalf("summary = %#v", report.Summary)
	}
}

func TestProblemsReportLimitIsPerListAndSummaryCountsArePreLimit(t *testing.T) {
	graph := core.Graph{
		Diagnostics: []core.Diagnostic{{Reason: "one"}, {Reason: "two"}},
		PolicyResults: []core.PolicyResult{
			{ID: "policy:one", Status: "fail"},
			{ID: "policy:two", Status: "unknown"},
		},
		Metrics: []core.Metric{
			{ID: "metric:one", Status: "warning"},
			{ID: "metric:two", Status: "warning"},
		},
	}

	report := buildProblemsReport(graph, 1)
	if len(report.Diagnostics) != 1 || len(report.PolicyResults) != 1 || len(report.Metrics) != 1 {
		t.Fatalf("limited report = %#v", report)
	}
	if report.Summary.Diagnostics != 2 || report.Summary.PolicyResults != 2 || report.Summary.Metrics != 2 {
		t.Fatalf("summary = %#v, want pre-limit counts", report.Summary)
	}
	if !report.Summary.Truncated {
		t.Fatalf("summary = %#v, want truncated", report.Summary)
	}
}

func TestProblemsReportTreatsNonPositiveLimitAsOne(t *testing.T) {
	graph := core.Graph{Diagnostics: []core.Diagnostic{{Reason: "one"}, {Reason: "two"}}}

	report := buildProblemsReport(graph, 0)
	if len(report.Diagnostics) != 1 || report.Summary.Diagnostics != 2 || !report.Summary.Truncated {
		t.Fatalf("report = %#v", report)
	}
}

func ids[T any](items []T, id func(T) string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, id(item))
	}
	return out
}

func same(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
