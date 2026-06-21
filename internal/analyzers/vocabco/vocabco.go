package vocabco

import (
	"sort"
	"strconv"
	"strings"

	"go-safedesign/internal/core"
	"go-safedesign/internal/pipeline"
)

const (
	ID      = "vocabulary_cooccurrence"
	Version = "prototype-1"
)

type Analyzer struct{}

type pairKey struct {
	PackagePath string
	TermA       string
	TermB       string
}

type pairAccumulator struct {
	key      pairKey
	count    int
	evidence map[string]bool
	trust    core.TrustLevel
}

func (Analyzer) Metadata() pipeline.AnalyzerMetadata {
	return pipeline.AnalyzerMetadata{
		ID:                    ID,
		Version:               Version,
		Stage:                 pipeline.StageDDDClassification,
		InputFactKinds:        []string{core.FactKindObservation},
		MinimumRequiredTrust:  core.TrustSyntaxObserved,
		MaximumEmittedTrust:   core.TrustTypeResolved,
		EmittedFactKinds:      []string{core.FactKindObservation},
		ConfigurationSection:  "vocabularyCooccurrence",
		FailureMode:           pipeline.FailureModePartial,
		IncompleteInputPolicy: pipeline.IncompleteInputAllow,
	}
}

func Metadata() pipeline.AnalyzerMetadata {
	return Analyzer{}.Metadata()
}

func (a Analyzer) Run(context pipeline.GraphContext) (pipeline.AnalyzerResult, error) {
	observations := a.evaluate(context.Graph)
	return pipeline.AnalyzerResult{Observations: observations}, nil
}

func (Analyzer) evaluate(graph core.Graph) []core.Observation {
	byTarget := vocabularyTermsByTarget(graph.Observations)
	pairs := map[pairKey]*pairAccumulator{}
	for _, targetTerms := range byTarget {
		for _, pair := range pairsForTarget(targetTerms) {
			acc := pairs[pair.key]
			if acc == nil {
				acc = &pairAccumulator{key: pair.key, evidence: map[string]bool{}, trust: pair.trust}
				pairs[pair.key] = acc
			}
			acc.count++
			if core.TrustRank(pair.trust) < core.TrustRank(acc.trust) {
				acc.trust = pair.trust
			}
			for _, id := range pair.evidence {
				acc.evidence[id] = true
			}
		}
	}
	return observationsForPairs(pairs)
}

type targetTerm struct {
	Term        string
	PackagePath string
	ID          string
	TrustLevel  core.TrustLevel
}

type targetPair struct {
	key      pairKey
	evidence []string
	trust    core.TrustLevel
}

func vocabularyTermsByTarget(observations []core.Observation) map[string][]targetTerm {
	out := map[string][]targetTerm{}
	for _, observation := range observations {
		if observation.Name != core.ObservationNameVocabularyTerm || observation.Value == "" || observation.TargetID == "" {
			continue
		}
		pkg := observation.Attributes["packagePath"]
		if pkg == "" {
			continue
		}
		out[observation.TargetID] = append(out[observation.TargetID], targetTerm{
			Term:        observation.Value,
			PackagePath: pkg,
			ID:          observation.ID,
			TrustLevel:  observation.TrustLevel,
		})
	}
	return out
}

func pairsForTarget(terms []targetTerm) []targetPair {
	byTerm := map[string][]targetTerm{}
	for _, term := range terms {
		byTerm[term.Term] = append(byTerm[term.Term], term)
	}
	uniqueTerms := make([]string, 0, len(byTerm))
	for term := range byTerm {
		uniqueTerms = append(uniqueTerms, term)
	}
	sort.Strings(uniqueTerms)

	var pairs []targetPair
	for i := 0; i < len(uniqueTerms); i++ {
		for j := i + 1; j < len(uniqueTerms); j++ {
			left := byTerm[uniqueTerms[i]]
			right := byTerm[uniqueTerms[j]]
			for _, a := range left {
				for _, b := range right {
					if a.PackagePath != b.PackagePath {
						continue
					}
					pairs = append(pairs, targetPair{
						key: pairKey{
							PackagePath: a.PackagePath,
							TermA:       uniqueTerms[i],
							TermB:       uniqueTerms[j],
						},
						evidence: []string{a.ID, b.ID},
						trust:    minTrust(a.TrustLevel, b.TrustLevel),
					})
				}
			}
		}
	}
	return pairs
}

func observationsForPairs(pairs map[pairKey]*pairAccumulator) []core.Observation {
	observations := make([]core.Observation, 0, len(pairs))
	for _, acc := range pairs {
		observations = append(observations, core.Observation{
			ID:    observationID(acc.key),
			Kind:  core.FactKindObservation,
			Name:  core.ObservationNameVocabularyCooccurrence,
			Value: acc.key.TermA + " " + acc.key.TermB,
			Attributes: map[string]string{
				"packagePath": acc.key.PackagePath,
				"termA":       acc.key.TermA,
				"termB":       acc.key.TermB,
				"count":       strconv.Itoa(acc.count),
			},
			Evidence:   sortedEvidence(acc.evidence),
			Source:     core.ObservationSourceInferred,
			TrustLevel: acc.trust,
			Freshness:  core.FreshnessFresh,
		})
	}
	sort.Slice(observations, func(i, j int) bool { return observations[i].ID < observations[j].ID })
	return observations
}

func observationID(key pairKey) string {
	return core.IDPrefixObservation + ID + ":" + core.HashBytes([]byte(strings.Join([]string{key.PackagePath, key.TermA, key.TermB}, "\x00")))
}

func sortedEvidence(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func minTrust(a, b core.TrustLevel) core.TrustLevel {
	if core.TrustRank(a) <= core.TrustRank(b) {
		return a
	}
	return b
}
