package indexer

import (
	"fmt"
	"sort"
	"strings"

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

const (
	AnalyzerIDModuleDependencyEnrichment = moddep.ID
	AnalyzerIDVocabularyExtraction       = vocab.ID
	AnalyzerIDVocabularyCooccurrence     = vocabco.ID
	AnalyzerIDLanguageZoneCandidate      = langzone.ID
	AnalyzerIDBridgeSymbol               = bridge.ID
	AnalyzerIDUbiquitousLanguage         = ublang.ID
	AnalyzerIDComplexity                 = complexity.ID
	AnalyzerIDDependencyPolicy           = deppolicy.ID
)

type AnalyzerExecutionOptions struct {
	Include []string
	Skip    []string
}

type analyzerSpec struct {
	id           string
	dependencies []string
	factory      func() pipeline.Analyzer
}

func productionAnalyzerSpecs() []analyzerSpec {
	return []analyzerSpec{
		{id: AnalyzerIDModuleDependencyEnrichment, factory: func() pipeline.Analyzer { return moddep.Analyzer{} }},
		{id: AnalyzerIDVocabularyExtraction, factory: func() pipeline.Analyzer { return vocab.Analyzer{} }},
		{id: AnalyzerIDVocabularyCooccurrence, dependencies: []string{AnalyzerIDVocabularyExtraction}, factory: func() pipeline.Analyzer { return vocabco.Analyzer{} }},
		{id: AnalyzerIDLanguageZoneCandidate, dependencies: []string{AnalyzerIDVocabularyExtraction, AnalyzerIDVocabularyCooccurrence}, factory: func() pipeline.Analyzer { return langzone.Analyzer{} }},
		{id: AnalyzerIDBridgeSymbol, dependencies: []string{AnalyzerIDLanguageZoneCandidate}, factory: func() pipeline.Analyzer { return bridge.Analyzer{} }},
		{id: AnalyzerIDUbiquitousLanguage, factory: func() pipeline.Analyzer { return ublang.Analyzer{} }},
		{id: AnalyzerIDComplexity, factory: func() pipeline.Analyzer { return complexity.Analyzer{} }},
		{id: AnalyzerIDDependencyPolicy, factory: func() pipeline.Analyzer { return deppolicy.Analyzer{} }},
	}
}

func KnownAnalyzerIDs() []string {
	specByID := map[string]analyzerSpec{}
	for _, spec := range productionAnalyzerSpecs() {
		specByID[spec.id] = spec
	}
	return knownAnalyzerIDs(specByID)
}

func planAnalyzers(options AnalyzerExecutionOptions) ([]pipeline.Analyzer, error) {
	specs := productionAnalyzerSpecs()
	specByID := map[string]analyzerSpec{}
	for _, spec := range specs {
		specByID[spec.id] = spec
	}

	include, err := analyzerIDSet(options.Include, specByID)
	if err != nil {
		return nil, err
	}
	skip, err := analyzerIDSet(options.Skip, specByID)
	if err != nil {
		return nil, err
	}

	selected := map[string]bool{}
	if len(include) == 0 {
		for _, spec := range specs {
			selected[spec.id] = true
		}
	} else {
		var visit func(string)
		visit = func(id string) {
			if selected[id] {
				return
			}
			spec := specByID[id]
			for _, dependency := range spec.dependencies {
				visit(dependency)
			}
			selected[id] = true
		}
		for id := range include {
			visit(id)
		}
	}

	for id := range skip {
		delete(selected, id)
	}
	for {
		removed := false
		for id := range selected {
			for _, dependency := range specByID[id].dependencies {
				if !selected[dependency] {
					delete(selected, id)
					removed = true
					break
				}
			}
		}
		if !removed {
			break
		}
	}

	analyzers := make([]pipeline.Analyzer, 0, len(selected))
	for _, spec := range specs {
		if selected[spec.id] {
			analyzers = append(analyzers, spec.factory())
		}
	}
	return analyzers, nil
}

func analyzerIDSet(ids []string, specByID map[string]analyzerSpec) (map[string]bool, error) {
	out := map[string]bool{}
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := specByID[id]; !ok {
			return nil, fmt.Errorf("unknown analyzer %q; known analyzers: %s", id, strings.Join(knownAnalyzerIDs(specByID), ", "))
		}
		out[id] = true
	}
	return out, nil
}

func knownAnalyzerIDs(specByID map[string]analyzerSpec) []string {
	ids := make([]string, 0, len(specByID))
	for id := range specByID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
