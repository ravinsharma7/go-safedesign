package indexer

import (
	"testing"

	"github.com/ravinsharma7/go-safedesign/internal/core"
	"github.com/ravinsharma7/go-safedesign/internal/pipeline"
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

func TestRunAnalyzerRejectsInvalidMetadataWithoutExecutingAnalyzer(t *testing.T) {
	ran := false
	b := &graphBuilder{
		nodes: map[string]Node{},
		edges: map[string]Edge{},
	}
	b.runAnalyzer(invalidMetadataAnalyzer{ran: &ran})

	if ran {
		t.Fatal("invalid metadata analyzer should not execute")
	}
	if len(b.observations) != 0 {
		t.Fatalf("observations = %#v, want no merged facts", b.observations)
	}
	if len(b.diagnostics) == 0 || b.diagnostics[0].Status != core.StatusAnalysisError {
		t.Fatalf("diagnostics = %#v, want metadata validation diagnostic", b.diagnostics)
	}
	run := runByAnalyzerForTest(b.runs, "invalid_metadata")
	if run == nil || run.Status != core.StatusAnalysisError || run.EmittedFacts != 0 {
		t.Fatalf("run = %#v, want analysis_error with no emitted facts", run)
	}
}

type invalidOutputAnalyzer struct{}

func (invalidOutputAnalyzer) Metadata() pipeline.AnalyzerMetadata {
	return pipeline.AnalyzerMetadata{
		ID:                    "invalid_output",
		Version:               "test",
		Stage:                 pipeline.StageDDDClassification,
		InputFactKinds:        []string{core.NodeKindPackage},
		MinimumRequiredTrust:  core.TrustSyntaxObserved,
		MaximumEmittedTrust:   core.TrustSyntaxObserved,
		EmittedFactKinds:      []string{"observation"},
		FailureMode:           pipeline.FailureModePartial,
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

type invalidMetadataAnalyzer struct {
	ran *bool
}

func (a invalidMetadataAnalyzer) Metadata() pipeline.AnalyzerMetadata {
	return pipeline.AnalyzerMetadata{
		ID:                    "invalid_metadata",
		Version:               "test",
		Stage:                 pipeline.StageDDDClassification,
		InputFactKinds:        []string{core.NodeKindPackage},
		MinimumRequiredTrust:  core.TrustSyntaxObserved,
		MaximumEmittedTrust:   core.TrustSyntaxObserved,
		EmittedFactKinds:      []string{"custom_fact"},
		FailureMode:           pipeline.FailureModePartial,
		IncompleteInputPolicy: pipeline.IncompleteInputAllow,
	}
}

func (a invalidMetadataAnalyzer) Run(pipeline.GraphContext) (pipeline.AnalyzerResult, error) {
	*a.ran = true
	return pipeline.AnalyzerResult{Observations: []core.Observation{{
		ID:         "observation:should-not-merge",
		Kind:       core.FactKindObservation,
		Name:       core.ObservationNameVocabularyTerm,
		Value:      "order",
		Source:     core.ObservationSourceObserved,
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
