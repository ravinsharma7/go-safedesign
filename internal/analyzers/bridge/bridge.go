package bridge

import (
	"sort"
	"strings"

	"go-safedesign/internal/core"
	"go-safedesign/internal/pipeline"
)

const (
	ID      = "bridge_symbol"
	Version = "prototype-1"
)

type Analyzer struct{}

type candidate struct {
	ID          string
	PackagePath string
	TrustLevel  core.TrustLevel
}

type bridgeObservation struct {
	edge      core.Edge
	from      candidate
	to        candidate
	trust     core.TrustLevel
	evidence  []string
	edgeKind  string
	value     string
	attribute map[string]string
}

func (Analyzer) Metadata() pipeline.AnalyzerMetadata {
	return pipeline.AnalyzerMetadata{
		ID:                    ID,
		Version:               Version,
		Stage:                 pipeline.StageDDDClassification,
		InputFactKinds:        []string{core.FactKindObservation, core.EdgeKindImports, core.EdgeKindDependsOn},
		MinimumRequiredTrust:  core.TrustSyntaxObserved,
		MaximumEmittedTrust:   core.TrustTypeResolved,
		EmittedFactKinds:      []string{core.FactKindObservation},
		ConfigurationSection:  "bridgeSymbol",
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
	candidates := candidatesByPackage(graph.Observations)
	var bridges []bridgeObservation
	for _, edge := range graph.Edges {
		fromPackage, toPackage, ok := bridgePackages(edge)
		if !ok {
			continue
		}
		fromCandidate, ok := candidates[fromPackage]
		if !ok {
			continue
		}
		toCandidate, ok := candidates[toPackage]
		if !ok {
			continue
		}
		bridges = append(bridges, newBridgeObservation(edge, fromCandidate, toCandidate))
	}
	return observationsForBridges(bridges)
}

func candidatesByPackage(observations []core.Observation) map[string]candidate {
	out := map[string]candidate{}
	for _, observation := range observations {
		if observation.Name != core.ObservationNameLanguageZoneCandidate || observation.ID == "" {
			continue
		}
		pkg := observation.Attributes["packagePath"]
		if pkg == "" || core.IsPlaceholderID(observation.TargetID) {
			continue
		}
		out[pkg] = candidate{
			ID:          observation.ID,
			PackagePath: pkg,
			TrustLevel:  observation.TrustLevel,
		}
	}
	return out
}

func bridgePackages(edge core.Edge) (string, string, bool) {
	if edge.Kind != core.EdgeKindImports && edge.Kind != core.EdgeKindDependsOn {
		return "", "", false
	}
	if core.IsIncompleteEdge(edge) || edge.From == edge.To {
		return "", "", false
	}
	fromPackage, ok := packagePathFromID(edge.From)
	if !ok {
		return "", "", false
	}
	toPackage, ok := packagePathFromID(edge.To)
	if !ok || fromPackage == toPackage {
		return "", "", false
	}
	return fromPackage, toPackage, true
}

func packagePathFromID(id string) (string, bool) {
	return strings.CutPrefix(id, core.IDPrefixPackage)
}

func newBridgeObservation(edge core.Edge, from, to candidate) bridgeObservation {
	trust := minTrust(edge.TrustLevel, minTrust(from.TrustLevel, to.TrustLevel))
	evidence := []string{edge.ID, from.ID, to.ID}
	sort.Strings(evidence)
	value := from.PackagePath + " -> " + to.PackagePath
	return bridgeObservation{
		edge:     edge,
		from:     from,
		to:       to,
		trust:    trust,
		evidence: evidence,
		edgeKind: edge.Kind,
		value:    value,
		attribute: map[string]string{
			"fromPackagePath": from.PackagePath,
			"toPackagePath":   to.PackagePath,
			"edgeKind":        edge.Kind,
			"fromCandidateId": from.ID,
			"toCandidateId":   to.ID,
		},
	}
}

func observationsForBridges(bridges []bridgeObservation) []core.Observation {
	observations := make([]core.Observation, 0, len(bridges))
	for _, bridge := range bridges {
		observations = append(observations, core.Observation{
			ID:         observationID(bridge.edge.ID, bridge.from.PackagePath, bridge.to.PackagePath),
			Kind:       core.FactKindObservation,
			Name:       core.ObservationNameBridgeSymbol,
			Value:      bridge.value,
			TargetID:   bridge.edge.ID,
			TargetKind: core.FactKindEdge,
			Attributes: bridge.attribute,
			Evidence:   bridge.evidence,
			Source:     core.ObservationSourceInferred,
			TrustLevel: bridge.trust,
			Freshness:  core.FreshnessFresh,
		})
	}
	sort.Slice(observations, func(i, j int) bool { return observations[i].ID < observations[j].ID })
	return observations
}

func observationID(edgeID, fromPackage, toPackage string) string {
	return core.IDPrefixObservation + ID + ":" + core.HashBytes([]byte(strings.Join([]string{edgeID, fromPackage, toPackage}, "\x00")))
}

func minTrust(a, b core.TrustLevel) core.TrustLevel {
	if core.TrustRank(a) <= core.TrustRank(b) {
		return a
	}
	return b
}
