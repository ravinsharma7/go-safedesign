package pipeline_test

import (
	"testing"

	"github.com/ravinsharma7/go-safedesign/internal/analyzers/bridge"
	"github.com/ravinsharma7/go-safedesign/internal/analyzers/complexity"
	"github.com/ravinsharma7/go-safedesign/internal/analyzers/deppolicy"
	"github.com/ravinsharma7/go-safedesign/internal/analyzers/langzone"
	"github.com/ravinsharma7/go-safedesign/internal/analyzers/moddep"
	"github.com/ravinsharma7/go-safedesign/internal/analyzers/ubilang"
	"github.com/ravinsharma7/go-safedesign/internal/analyzers/vocab"
	"github.com/ravinsharma7/go-safedesign/internal/analyzers/vocabco"
	"github.com/ravinsharma7/go-safedesign/internal/pipeline"
)

func TestProductionAnalyzerMetadataSatisfiesContract(t *testing.T) {
	all := []pipeline.AnalyzerMetadata{
		bridge.Metadata(),
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
