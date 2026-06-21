package indexer

import (
	"testing"

	"github.com/ravinsharma7/go-safedesign/internal/pipeline"
)

func TestAnalyzerPlanIncludeExpandsLanguageZoneDependencies(t *testing.T) {
	analyzers, err := planAnalyzers(AnalyzerExecutionOptions{Include: []string{AnalyzerIDLanguageZoneCandidate}})
	if err != nil {
		t.Fatal(err)
	}
	if got := plannedAnalyzerIDs(analyzers); !sameStrings(got, []string{AnalyzerIDVocabularyExtraction, AnalyzerIDVocabularyCooccurrence, AnalyzerIDLanguageZoneCandidate}) {
		t.Fatalf("planned analyzers = %#v", got)
	}
}

func TestAnalyzerPlanIncludeExpandsBridgeDependencies(t *testing.T) {
	analyzers, err := planAnalyzers(AnalyzerExecutionOptions{Include: []string{AnalyzerIDBridgeSymbol}})
	if err != nil {
		t.Fatal(err)
	}
	if got := plannedAnalyzerIDs(analyzers); !sameStrings(got, []string{AnalyzerIDVocabularyExtraction, AnalyzerIDVocabularyCooccurrence, AnalyzerIDLanguageZoneCandidate, AnalyzerIDBridgeSymbol}) {
		t.Fatalf("planned analyzers = %#v", got)
	}
}

func TestAnalyzerPlanSkipVocabularySkipsDependentDDDEvidenceAnalyzers(t *testing.T) {
	analyzers, err := planAnalyzers(AnalyzerExecutionOptions{Skip: []string{AnalyzerIDVocabularyExtraction}})
	if err != nil {
		t.Fatal(err)
	}
	got := plannedAnalyzerIDs(analyzers)
	for _, skipped := range []string{AnalyzerIDVocabularyExtraction, AnalyzerIDVocabularyCooccurrence, AnalyzerIDLanguageZoneCandidate, AnalyzerIDBridgeSymbol} {
		if containsString(got, skipped) {
			t.Fatalf("planned analyzers = %#v, should skip %s", got, skipped)
		}
	}
	for _, retained := range []string{AnalyzerIDModuleDependencyEnrichment, AnalyzerIDUbiquitousLanguage, AnalyzerIDComplexity, AnalyzerIDDependencyPolicy} {
		if !containsString(got, retained) {
			t.Fatalf("planned analyzers = %#v, should retain %s", got, retained)
		}
	}
}

func TestAnalyzerPlanSkipComplexityAndDependencyPolicy(t *testing.T) {
	analyzers, err := planAnalyzers(AnalyzerExecutionOptions{Skip: []string{AnalyzerIDComplexity, AnalyzerIDDependencyPolicy}})
	if err != nil {
		t.Fatal(err)
	}
	got := plannedAnalyzerIDs(analyzers)
	if containsString(got, AnalyzerIDComplexity) || containsString(got, AnalyzerIDDependencyPolicy) {
		t.Fatalf("planned analyzers = %#v, should skip complexity and dependency policy", got)
	}
	if !containsString(got, AnalyzerIDVocabularyExtraction) {
		t.Fatalf("planned analyzers = %#v, should preserve unrelated analyzers", got)
	}
}

func TestAnalyzerPlanRejectsUnknownAnalyzerIDs(t *testing.T) {
	_, err := planAnalyzers(AnalyzerExecutionOptions{Include: []string{"missing"}})
	if err == nil {
		t.Fatal("expected unknown analyzer error")
	}
}

func plannedAnalyzerIDs(analyzers []pipeline.Analyzer) []string {
	ids := make([]string, 0, len(analyzers))
	for _, analyzer := range analyzers {
		ids = append(ids, analyzer.Metadata().ID)
	}
	return ids
}

func sameStrings(left, right []string) bool {
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

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
