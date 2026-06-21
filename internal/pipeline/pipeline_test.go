package pipeline

import (
	"errors"
	"strings"
	"testing"

	"go-safedesign/internal/core"
)

func TestStageConstants(t *testing.T) {
	stages := []Stage{
		StageSourceDiscovery,
		StageModuleExtraction,
		StageSyntaxExtraction,
		StageBaseGraphAssembly,
		StagePackageLoading,
		StageTypeResolution,
		StageModuleDependencyEnrichment,
		StageThirdPartyBehaviorLabeling,
		StageFrameworkExtraction,
		StageDDDClassification,
		StageComplexityMetrics,
		StagePolicyEvaluation,
		StageQueryMaterialization,
		StageRendering,
	}
	if len(stages) != 14 {
		t.Fatalf("stage count = %d, want 14", len(stages))
	}
	if StageModuleExtraction != "module_extraction" || StageRendering != "rendering" {
		t.Fatalf("unexpected stage constants: %q %q", StageModuleExtraction, StageRendering)
	}
}

func TestRegistryCopiesAnalyzerMetadata(t *testing.T) {
	registry := NewRegistry()
	registry.Register(AnalyzerMetadata{
		ID:                   "test",
		Version:              "v1",
		Stage:                StageSyntaxExtraction,
		MinimumRequiredTrust: core.TrustSyntaxObserved,
		MaximumEmittedTrust:  core.TrustSyntaxObserved,
	})
	items := registry.Analyzers()
	items[0].ID = "mutated"
	if registry.Analyzers()[0].ID != "test" {
		t.Fatal("registry returned mutable backing slice")
	}
}

func TestRunAnalyzerBuildsRunRecord(t *testing.T) {
	analyzer := fakeAnalyzer{
		metadata: AnalyzerMetadata{ID: "fake", Version: "v1", Stage: StagePolicyEvaluation},
		result: AnalyzerResult{Diagnostics: []core.Diagnostic{{
			Source: "policy:test",
			Reason: "failed",
		}}, PolicyResults: []core.PolicyResult{{ID: "policy_result:test"}}, Metrics: []core.Metric{{ID: "metric:test"}}, Warnings: []core.Warning{{ID: "warning:test"}}, Edges: []core.Edge{{ID: "edge:test"}}},
	}

	result, run, err := RunAnalyzer(analyzer, GraphContext{Configuration: []byte(`{"x":1}`)})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("result = %#v", result)
	}
	if run.AnalyzerID != "fake" || run.AnalyzerVersion != "v1" || run.Stage != string(StagePolicyEvaluation) {
		t.Fatalf("run metadata = %#v", run)
	}
	if run.Status != "partial" || run.EmittedFacts != 5 || run.ConfigurationHash == "" {
		t.Fatalf("run = %#v, want partial with config hash", run)
	}
	if run.RunID == "" || run.StartedAt == "" || run.FinishedAt == "" {
		t.Fatalf("run = %#v, want run id and timestamps", run)
	}
}

func TestRunAnalyzerRecordsAnalysisError(t *testing.T) {
	analyzer := fakeAnalyzer{
		metadata: AnalyzerMetadata{ID: "fake", Version: "v1", Stage: StagePolicyEvaluation},
		err:      errors.New("bad config"),
	}

	_, run, err := RunAnalyzer(analyzer, GraphContext{})
	if err == nil {
		t.Fatal("expected error")
	}
	if run.Status != "analysis_error" || run.Diagnostics[0] != "fake: bad config" {
		t.Fatalf("run = %#v", run)
	}
}

func TestValidateAnalyzerResultRejectsInvalidOutput(t *testing.T) {
	graph := core.Graph{Nodes: []core.Node{{ID: "package:example.com/app", Kind: "package", TrustLevel: core.TrustSyntaxObserved}}}
	metadata := AnalyzerMetadata{
		ID:                  "fake",
		Stage:               StageDDDClassification,
		MaximumEmittedTrust: core.TrustSyntaxObserved,
		EmittedFactKinds:    []string{"observation"},
	}
	result := AnalyzerResult{
		Observations: []core.Observation{{
			ID:         "observation:bad",
			Kind:       "observation",
			Name:       "vocabulary.term",
			Value:      "order",
			TargetID:   "package:example.com/missing",
			TargetKind: "node",
			Evidence:   []string{"package:example.com/missing"},
			Source:     "observed",
			TrustLevel: core.TrustTypeResolved,
		}},
		Labels: []core.Label{{ID: "label:undeclared", Kind: "label", TrustLevel: core.TrustSyntaxObserved}},
	}

	diagnostics := ValidateAnalyzerResult(graph, metadata, result)
	if len(diagnostics) < 4 {
		t.Fatalf("diagnostics = %#v, want validation failures", diagnostics)
	}
	if !hasValidationReason(diagnostics, "undeclared fact kind label") {
		t.Fatalf("diagnostics = %#v, missing undeclared kind failure", diagnostics)
	}
	if !hasValidationReason(diagnostics, "exceeds analyzer maximum") {
		t.Fatalf("diagnostics = %#v, missing trust failure", diagnostics)
	}
	if !hasValidationReason(diagnostics, "references missing target") {
		t.Fatalf("diagnostics = %#v, missing target failure", diagnostics)
	}
	if !hasValidationReason(diagnostics, "references missing evidence") {
		t.Fatalf("diagnostics = %#v, missing evidence failure", diagnostics)
	}
}

func TestValidateAnalyzerResultAcceptsObservation(t *testing.T) {
	graph := core.Graph{Nodes: []core.Node{{ID: "function:example.com/app.PlaceOrder", Kind: "function", TrustLevel: core.TrustSyntaxObserved}}}
	metadata := AnalyzerMetadata{
		ID:                  "vocabulary_extraction",
		Stage:               StageDDDClassification,
		MaximumEmittedTrust: core.TrustSyntaxObserved,
		EmittedFactKinds:    []string{"observation"},
	}
	result := AnalyzerResult{Observations: []core.Observation{{
		ID:         "observation:vocabulary:test",
		Kind:       "observation",
		Name:       "vocabulary.term",
		Value:      "order",
		TargetID:   "function:example.com/app.PlaceOrder",
		TargetKind: "node",
		Evidence:   []string{"function:example.com/app.PlaceOrder"},
		Source:     "observed",
		TrustLevel: core.TrustSyntaxObserved,
	}}}
	if diagnostics := ValidateAnalyzerResult(graph, metadata, result); len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
}

func TestValidateAnalyzerMetadataAcceptsValidMetadata(t *testing.T) {
	metadata := validAnalyzerMetadataForTest()
	if diagnostics := ValidateAnalyzerMetadata(metadata); len(diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", diagnostics)
	}
}

func TestValidateAnalyzerMetadataRejectsMissingRequiredFields(t *testing.T) {
	diagnostics := ValidateAnalyzerMetadata(AnalyzerMetadata{})
	wants := []string{
		"missing id",
		"missing version",
		"missing stage",
		"missing input fact kinds",
		"missing minimum required trust",
		"missing maximum emitted trust",
		"missing emitted fact kinds",
		"missing failure mode",
		"missing incomplete input policy",
	}
	for _, want := range wants {
		if !hasValidationReason(diagnostics, want) {
			t.Fatalf("diagnostics = %#v, missing %q", diagnostics, want)
		}
	}
}

func TestValidateAnalyzerMetadataRejectsUnsupportedValues(t *testing.T) {
	metadata := validAnalyzerMetadataForTest()
	metadata.Stage = "custom_stage"
	metadata.MinimumRequiredTrust = "guessed"
	metadata.MaximumEmittedTrust = "claimed"
	metadata.EmittedFactKinds = []string{"custom_fact"}
	metadata.FailureMode = "abort"
	metadata.IncompleteInputPolicy = "maybe"

	diagnostics := ValidateAnalyzerMetadata(metadata)
	wants := []string{
		"unsupported stage",
		"unsupported minimum required trust",
		"unsupported maximum emitted trust",
		"unsupported emitted fact kind",
		"unsupported failure mode",
		"unsupported incomplete input policy",
	}
	for _, want := range wants {
		if !hasValidationReason(diagnostics, want) {
			t.Fatalf("diagnostics = %#v, missing %q", diagnostics, want)
		}
	}
}

func TestValidateAnalyzerMetadataRejectsTrustRangeInversion(t *testing.T) {
	metadata := validAnalyzerMetadataForTest()
	metadata.MinimumRequiredTrust = core.TrustTypeResolved
	metadata.MaximumEmittedTrust = core.TrustSyntaxObserved

	diagnostics := ValidateAnalyzerMetadata(metadata)
	if !hasValidationReason(diagnostics, "minimum required trust type_resolved exceeds maximum emitted trust syntax_observed") {
		t.Fatalf("diagnostics = %#v, missing trust range failure", diagnostics)
	}
}

func validAnalyzerMetadataForTest() AnalyzerMetadata {
	return AnalyzerMetadata{
		ID:                    "valid",
		Version:               "test",
		Stage:                 StageDDDClassification,
		InputFactKinds:        []string{core.NodeKindPackage},
		MinimumRequiredTrust:  core.TrustSyntaxObserved,
		MaximumEmittedTrust:   core.TrustTypeResolved,
		EmittedFactKinds:      []string{core.FactKindObservation},
		FailureMode:           FailureModePartial,
		IncompleteInputPolicy: IncompleteInputRequireComplete,
	}
}

func hasValidationReason(diagnostics []core.Diagnostic, want string) bool {
	for _, diagnostic := range diagnostics {
		if strings.Contains(diagnostic.Reason, want) {
			return true
		}
	}
	return false
}

type fakeAnalyzer struct {
	metadata AnalyzerMetadata
	result   AnalyzerResult
	err      error
}

func (f fakeAnalyzer) Metadata() AnalyzerMetadata {
	return f.metadata
}

func (f fakeAnalyzer) Run(GraphContext) (AnalyzerResult, error) {
	return f.result, f.err
}
