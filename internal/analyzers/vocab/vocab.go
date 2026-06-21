package vocab

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"go-safedesign/internal/core"
	"go-safedesign/internal/pipeline"
	"go-safedesign/internal/wordcase"
)

const (
	ID      = "vocabulary_extraction"
	Version = "prototype-1"
)

type Analyzer struct{}

func (Analyzer) Metadata() pipeline.AnalyzerMetadata {
	return pipeline.AnalyzerMetadata{
		ID:                    ID,
		Version:               Version,
		Stage:                 pipeline.StageDDDClassification,
		InputFactKinds:        []string{"package", "file", "type", "interface", "struct", "function", "method", "field", "import", "unresolved_call", "imports", "depends_on"},
		MinimumRequiredTrust:  core.TrustSyntaxObserved,
		MaximumEmittedTrust:   core.TrustTypeResolved,
		EmittedFactKinds:      []string{"observation"},
		ConfigurationSection:  "vocabularyExtraction",
		FailureMode:           "partial",
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
	var observations []core.Observation
	for _, node := range graph.Nodes {
		observations = append(observations, observationsForNode(node)...)
	}
	for _, edge := range graph.Edges {
		if edge.Kind != "imports" && edge.Kind != "depends_on" {
			continue
		}
		if edge.Complete && !edge.Synthetic && !strings.HasPrefix(edge.To, "placeholder:") {
			continue
		}
		observations = append(observations, incompleteDependencyObservation(edge))
	}
	sort.Slice(observations, func(i, j int) bool { return observations[i].ID < observations[j].ID })
	return observations
}

func observationsForNode(node core.Node) []core.Observation {
	if node.Synthetic || node.Kind == "placeholder" || strings.HasPrefix(node.ID, "placeholder:") {
		return nil
	}
	if !vocabularyNodeKind(node.Kind) {
		return nil
	}
	original := originalForNode(node)
	if original == "" {
		return nil
	}
	tokens := SplitWords(original)
	var observations []core.Observation
	for idx, token := range tokens {
		if token == "" {
			continue
		}
		observations = append(observations, core.Observation{
			ID:         observationID("term", node.ID, strconv.Itoa(idx), token),
			Kind:       "observation",
			Name:       "vocabulary.term",
			Value:      token,
			TargetID:   node.ID,
			TargetKind: "node",
			Attributes: map[string]string{
				"original":    original,
				"tokenIndex":  strconv.Itoa(idx),
				"nodeKind":    node.Kind,
				"packagePath": node.PackagePath,
				"modulePath":  node.ModulePath,
			},
			Evidence:   []string{node.ID},
			Source:     "observed",
			TrustLevel: node.TrustLevel,
			Freshness:  freshness(node.Freshness),
			SourceFile: node.SourceFile,
			LineRange:  node.LineRange,
		})
	}
	return observations
}

func incompleteDependencyObservation(edge core.Edge) core.Observation {
	return core.Observation{
		ID:         observationID("incomplete_dependency", edge.ID),
		Kind:       "observation",
		Name:       "vocabulary.incomplete_dependency",
		Value:      edge.To,
		TargetID:   edge.ID,
		TargetKind: "edge",
		Attributes: map[string]string{
			"edgeKind": edge.Kind,
			"from":     edge.From,
			"to":       edge.To,
			"reason":   edge.Reason,
		},
		Evidence:   []string{edge.ID},
		Source:     "observed",
		TrustLevel: edge.TrustLevel,
		Freshness:  "fresh",
		SourceFile: edge.SourceFile,
		LineRange:  edge.LineRange,
	}
}

func vocabularyNodeKind(kind string) bool {
	switch kind {
	case "package", "file", "type", "interface", "struct", "function", "method", "field", "import", "unresolved_call":
		return true
	default:
		return false
	}
}

func originalForNode(node core.Node) string {
	if node.Kind == "file" && node.SourceFile != "" {
		base := filepath.Base(node.SourceFile)
		return strings.TrimSuffix(base, filepath.Ext(base))
	}
	return node.Name
}

func freshness(value string) string {
	if value == "" {
		return "fresh"
	}
	return value
}

func observationID(parts ...string) string {
	return "observation:vocabulary:" + core.HashBytes([]byte(strings.Join(parts, "\x00")))
}

func SplitWords(value string) []string {
	return wordcase.SplitWords(value)
}
