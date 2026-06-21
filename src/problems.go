package main

import "go-safedesign/internal/core"

type problemsReport struct {
	Summary       problemsSummary     `json:"summary"`
	Diagnostics   []core.Diagnostic   `json:"diagnostics"`
	PolicyResults []core.PolicyResult `json:"policyResults"`
	Metrics       []core.Metric       `json:"metrics"`
	Warnings      []core.Warning      `json:"warnings"`
	Queries       []core.QueryResult  `json:"queries"`
	Runs          []core.RunRecord    `json:"runs"`
}

type problemsSummary struct {
	Diagnostics   int  `json:"diagnostics"`
	PolicyResults int  `json:"policyResults"`
	Metrics       int  `json:"metrics"`
	Warnings      int  `json:"warnings"`
	Queries       int  `json:"queries"`
	Runs          int  `json:"runs"`
	Truncated     bool `json:"truncated"`
}

func buildProblemsReport(graph core.Graph, limit int) problemsReport {
	if limit < 1 {
		limit = 1
	}
	var report problemsReport
	report.Summary.Diagnostics = len(graph.Diagnostics)
	report.Diagnostics, report.Summary.Truncated = limited(graph.Diagnostics, limit, report.Summary.Truncated)

	var policyResults []core.PolicyResult
	for _, result := range graph.PolicyResults {
		if result.Status != "pass" {
			policyResults = append(policyResults, result)
		}
	}
	report.Summary.PolicyResults = len(policyResults)
	report.PolicyResults, report.Summary.Truncated = limited(policyResults, limit, report.Summary.Truncated)

	var metrics []core.Metric
	for _, metric := range graph.Metrics {
		if metric.Status == "warning" {
			metrics = append(metrics, metric)
		}
	}
	report.Summary.Metrics = len(metrics)
	report.Metrics, report.Summary.Truncated = limited(metrics, limit, report.Summary.Truncated)

	report.Summary.Warnings = len(graph.Warnings)
	report.Warnings, report.Summary.Truncated = limited(graph.Warnings, limit, report.Summary.Truncated)

	var queries []core.QueryResult
	for _, query := range graph.Queries {
		if query.Status != "pass" {
			queries = append(queries, query)
		}
	}
	report.Summary.Queries = len(queries)
	report.Queries, report.Summary.Truncated = limited(queries, limit, report.Summary.Truncated)

	var runs []core.RunRecord
	for _, run := range graph.Runs {
		if run.Status != "completed" {
			runs = append(runs, run)
		}
	}
	report.Summary.Runs = len(runs)
	report.Runs, report.Summary.Truncated = limited(runs, limit, report.Summary.Truncated)

	return report
}

func limited[T any](items []T, limit int, truncated bool) ([]T, bool) {
	if len(items) <= limit {
		return items, truncated
	}
	return items[:limit], true
}
