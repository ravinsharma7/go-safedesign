package moddep

import (
	"testing"

	"github.com/ravinsharma7/go-safedesign/internal/core"
	"github.com/ravinsharma7/go-safedesign/internal/pipeline"
)

func TestAnalyzerLabelsDirectModuleDependencies(t *testing.T) {
	edge := core.Edge{
		ID:         "edge:depends_on:module:example.com/shop->module:example.com/payments",
		From:       "module:example.com/shop",
		To:         "module:example.com/payments",
		Kind:       "depends_on",
		TrustLevel: core.TrustSyntaxObserved,
		Complete:   true,
	}
	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: core.Graph{Edges: []core.Edge{edge}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Labels) != 1 {
		t.Fatalf("labels = %#v, want one direct dependency label", result.Labels)
	}
	label := result.Labels[0]
	if label.Name != "module.dependency" || label.Value != "direct" || label.TargetID != edge.ID || label.TargetKind != "edge" || label.Source != "observed" || !contains(label.Evidence, edge.ID) {
		t.Fatalf("label = %#v, want observed direct dependency label on edge", label)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("warnings = %#v, want none without missing deps or outdated check", result.Warnings)
	}
}

func TestAnalyzerEmitsMissingDependencyWarnings(t *testing.T) {
	diagnostic := core.Diagnostic{
		Source:     "go/packages:example.com/shop/order",
		Reason:     "no required module provides package example.com/missing",
		Status:     "missing_dependency",
		SourceFile: "order/service.go",
		LineRange:  "3:2-3:22",
	}
	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: core.Graph{Diagnostics: []core.Diagnostic{diagnostic}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("warnings = %#v, want missing dependency warning", result.Warnings)
	}
	warning := result.Warnings[0]
	if warning.Kind != "missing_dependency" || warning.Reason != diagnostic.Reason || len(warning.Evidence) != 1 || warning.SourceFile != diagnostic.SourceFile || warning.LineRange != diagnostic.LineRange {
		t.Fatalf("warning = %#v, want diagnostic-backed missing dependency warning", warning)
	}
}

func TestAnalyzerDoesNotEmitOutdatedWarningsByDefault(t *testing.T) {
	edge := core.Edge{ID: "edge:depends_on:module:a->module:b", From: "module:a", To: "module:b", Kind: "depends_on", TrustLevel: core.TrustSyntaxObserved, Complete: true}
	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: core.Graph{Edges: []core.Edge{edge}}})
	if err != nil {
		t.Fatal(err)
	}
	if hasWarningKind(result.Warnings, "outdated_module") {
		t.Fatalf("warnings = %#v, outdated checks should be disabled by default", result.Warnings)
	}
}

func TestAnalyzerEmitsUnknownOutdatedWarningsWhenRequestedOffline(t *testing.T) {
	edge := core.Edge{ID: "edge:depends_on:module:a->module:b", From: "module:a", To: "module:b", Kind: "depends_on", TrustLevel: core.TrustSyntaxObserved, Complete: true}
	result, err := Analyzer{}.Run(pipeline.GraphContext{
		Graph:         core.Graph{Edges: []core.Edge{edge}},
		Configuration: []byte(`{"checkOutdatedModules":true,"networkAccess":"disabled"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("warnings = %#v, want one outdated warning", result.Warnings)
	}
	warning := result.Warnings[0]
	if warning.Kind != "outdated_module" || warning.AffectedEdgeID != edge.ID || !contains(warning.Evidence, edge.ID) {
		t.Fatalf("warning = %#v, want edge-backed outdated module warning", warning)
	}
}

func TestMetadataUsesModuleDependencyEnrichmentStage(t *testing.T) {
	metadata := Metadata()
	if metadata.ID != ID || metadata.Stage != pipeline.StageModuleDependencyEnrichment {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func hasWarningKind(warnings []core.Warning, kind string) bool {
	for _, warning := range warnings {
		if warning.Kind == kind {
			return true
		}
	}
	return false
}
