package pipeline_test

import (
	"testing"

	"go-safedesign/internal/analyzers/complexity"
	"go-safedesign/internal/analyzers/deppolicy"
	"go-safedesign/internal/analyzers/langzone"
	"go-safedesign/internal/analyzers/moddep"
	"go-safedesign/internal/analyzers/ubilang"
	"go-safedesign/internal/analyzers/vocab"
	"go-safedesign/internal/analyzers/vocabco"
	"go-safedesign/internal/pipeline"
)

func TestProductionAnalyzerMetadataSatisfiesContract(t *testing.T) {
	all := []pipeline.AnalyzerMetadata{
		complexity.Metadata(),
		deppolicy.Metadata(),
		langzone.Metadata(),
		moddep.Metadata(),
		ublang.Metadata(),
		vocab.Metadata(),
		vocabco.Metadata(),
	}
	for _, metadata := range all {
		if diagnostics := pipeline.ValidateAnalyzerMetadata(metadata); len(diagnostics) != 0 {
			t.Fatalf("%s metadata diagnostics = %#v", metadata.ID, diagnostics)
		}
	}
}
