package indexer

import (
	"testing"

	"go-safedesign/internal/core"
	"go-safedesign/internal/pipeline"
)

func TestRunAnalyzerRejectsInvalidOutputWithoutPartialMerge(t *testing.T) {
	b := &graphBuilder{
		nodes: map[string]Node{
			"package:example.com/app": {
				ID:         "package:example.com/app",
				Kind:       "package",
				TrustLevel: core.TrustSyntaxObserved,
			},
		},
		edges: map[string]Edge{},
	}
	b.runAnalyzer(invalidOutputAnalyzer{})

	if len(b.observations) != 0 {
		t.Fatalf("observations = %#v, want invalid facts rejected", b.observations)
	}
	if len(b.diagnostics) == 0 || b.diagnostics[0].Status != "analysis_error" {
		t.Fatalf("diagnostics = %#v, want validation diagnostic", b.diagnostics)
	}
	run := runByAnalyzerForTest(b.runs, "invalid_output")
	if run == nil || run.Status != "analysis_error" || run.EmittedFacts != 0 {
		t.Fatalf("run = %#v, want analysis_error with no merged facts", run)
	}
}

type invalidOutputAnalyzer struct{}

func (invalidOutputAnalyzer) Metadata() pipeline.AnalyzerMetadata {
	return pipeline.AnalyzerMetadata{
		ID:                    "invalid_output",
		Version:               "test",
		Stage:                 pipeline.StageDDDClassification,
		MaximumEmittedTrust:   core.TrustSyntaxObserved,
		EmittedFactKinds:      []string{"observation"},
		IncompleteInputPolicy: pipeline.IncompleteInputAllow,
	}
}

func (invalidOutputAnalyzer) Run(pipeline.GraphContext) (pipeline.AnalyzerResult, error) {
	return pipeline.AnalyzerResult{Observations: []core.Observation{{
		ID:         "observation:invalid",
		Kind:       "observation",
		Name:       "vocabulary.term",
		Value:      "missing",
		TargetID:   "package:example.com/missing",
		TargetKind: "node",
		Evidence:   []string{"package:example.com/missing"},
		Source:     "observed",
		TrustLevel: core.TrustSyntaxObserved,
	}}}, nil
}

func runByAnalyzerForTest(runs []core.RunRecord, analyzerID string) *core.RunRecord {
	for i := range runs {
		if runs[i].AnalyzerID == analyzerID {
			return &runs[i]
		}
	}
	return nil
}
