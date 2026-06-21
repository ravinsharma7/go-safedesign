package langzone

import (
	"sort"
	"strconv"
	"strings"

	"github.com/ravinsharma7/go-safedesign/internal/core"
	"github.com/ravinsharma7/go-safedesign/internal/pipeline"
)

const (
	ID      = "language_zone_candidate"
	Version = "prototype-1"
)

var candidateStopTerms = map[string]bool{
	"com":       true,
	"example":   true,
	"github":    true,
	"gitlab":    true,
	"go":        true,
	"internal":  true,
	"io":        true,
	"net":       true,
	"org":       true,
	"testdata":  true,
	"workspace": true,
}

type Analyzer struct{}

type packageEvidence struct {
	packagePath       string
	packageNodeID     string
	terms             map[string]bool
	cooccurrenceCount int
	evidence          map[string]bool
	trust             core.TrustLevel
	hasCooccurrence   bool
}

type parsedObservation struct {
	packagePath string
	terms       []string
	count       int
	id          string
	trust       core.TrustLevel
	targetID    string
}

func (Analyzer) Metadata() pipeline.AnalyzerMetadata {
	return pipeline.AnalyzerMetadata{
		ID:                    ID,
		Version:               Version,
		Stage:                 pipeline.StageDDDClassification,
		InputFactKinds:        []string{core.FactKindObservation, core.NodeKindPackage},
		MinimumRequiredTrust:  core.TrustSyntaxObserved,
		MaximumEmittedTrust:   core.TrustTypeResolved,
		EmittedFactKinds:      []string{core.FactKindObservation},
		ConfigurationSection:  "languageZoneCandidate",
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
	packages := realPackageNodes(graph.Nodes)
	placeholderPackages := placeholderPackagePaths(graph.Nodes)
	evidence := map[string]*packageEvidence{}
	for _, observation := range graph.Observations {
		parsed, ok := parseEvidenceObservation(observation)
		if !ok || core.IsPlaceholderID(parsed.targetID) || placeholderPackages[parsed.packagePath] {
			continue
		}
		acc := evidence[parsed.packagePath]
		if acc == nil {
			acc = &packageEvidence{
				packagePath:     parsed.packagePath,
				packageNodeID:   packages[parsed.packagePath],
				terms:           map[string]bool{},
				evidence:        map[string]bool{},
				trust:           parsed.trust,
				hasCooccurrence: parsed.count > 0,
			}
			evidence[parsed.packagePath] = acc
		}
		acc.add(parsed)
	}
	return candidateObservations(evidence)
}

func realPackageNodes(nodes []core.Node) map[string]string {
	out := map[string]string{}
	for _, node := range nodes {
		if node.Kind != core.NodeKindPackage || node.PackagePath == "" || core.IsPlaceholderNode(node) {
			continue
		}
		out[node.PackagePath] = node.ID
	}
	return out
}

func placeholderPackagePaths(nodes []core.Node) map[string]bool {
	out := map[string]bool{}
	for _, node := range nodes {
		if node.PackagePath == "" || !core.IsPlaceholderNode(node) {
			continue
		}
		out[node.PackagePath] = true
	}
	for pkg := range realPackageNodes(nodes) {
		delete(out, pkg)
	}
	return out
}

func parseEvidenceObservation(observation core.Observation) (parsedObservation, bool) {
	switch observation.Name {
	case core.ObservationNameVocabularyTerm:
		return parseTermObservation(observation)
	case core.ObservationNameVocabularyCooccurrence:
		return parseCooccurrenceObservation(observation)
	default:
		return parsedObservation{}, false
	}
}

func parseTermObservation(observation core.Observation) (parsedObservation, bool) {
	pkg := observation.Attributes["packagePath"]
	if pkg == "" || observation.Value == "" || isCandidateStopTerm(observation.Value) || observation.Source != core.ObservationSourceObserved {
		return parsedObservation{}, false
	}
	return parsedObservation{
		packagePath: pkg,
		terms:       []string{observation.Value},
		id:          observation.ID,
		trust:       observation.TrustLevel,
		targetID:    observation.TargetID,
	}, true
}

func parseCooccurrenceObservation(observation core.Observation) (parsedObservation, bool) {
	pkg := observation.Attributes["packagePath"]
	termA := observation.Attributes["termA"]
	termB := observation.Attributes["termB"]
	if pkg == "" || termA == "" || termB == "" || termA == termB || isCandidateStopTerm(termA) || isCandidateStopTerm(termB) || observation.Source != core.ObservationSourceInferred {
		return parsedObservation{}, false
	}
	count, err := strconv.Atoi(observation.Attributes["count"])
	if err != nil || count <= 0 {
		return parsedObservation{}, false
	}
	return parsedObservation{
		packagePath: pkg,
		terms:       []string{termA, termB},
		count:       count,
		id:          observation.ID,
		trust:       observation.TrustLevel,
		targetID:    observation.TargetID,
	}, true
}

func isCandidateStopTerm(term string) bool {
	return candidateStopTerms[term]
}

func (acc *packageEvidence) add(observation parsedObservation) {
	for _, term := range observation.terms {
		acc.terms[term] = true
	}
	if observation.count > 0 {
		acc.hasCooccurrence = true
		acc.cooccurrenceCount += observation.count
	}
	acc.evidence[observation.id] = true
	if acc.trust == "" || core.TrustRank(observation.trust) < core.TrustRank(acc.trust) {
		acc.trust = observation.trust
	}
}

func candidateObservations(packages map[string]*packageEvidence) []core.Observation {
	observations := make([]core.Observation, 0, len(packages))
	for _, acc := range packages {
		terms := sortedKeys(acc.terms)
		if !acc.hasCooccurrence || len(terms) < 2 {
			continue
		}
		targetKind := ""
		if acc.packageNodeID != "" {
			targetKind = "node"
		}
		observations = append(observations, core.Observation{
			ID:         observationID(acc.packagePath, terms),
			Kind:       core.FactKindObservation,
			Name:       core.ObservationNameLanguageZoneCandidate,
			Value:      strings.Join(terms, " "),
			TargetID:   acc.packageNodeID,
			TargetKind: targetKind,
			Attributes: map[string]string{
				"packagePath":       acc.packagePath,
				"terms":             strings.Join(terms, ","),
				"termCount":         strconv.Itoa(len(terms)),
				"cooccurrenceCount": strconv.Itoa(acc.cooccurrenceCount),
			},
			Evidence:   sortedKeys(acc.evidence),
			Source:     core.ObservationSourceInferred,
			TrustLevel: acc.trust,
			Freshness:  core.FreshnessFresh,
		})
	}
	sort.Slice(observations, func(i, j int) bool { return observations[i].ID < observations[j].ID })
	return observations
}

func observationID(packagePath string, terms []string) string {
	parts := append([]string{packagePath}, terms...)
	return core.IDPrefixObservation + ID + ":" + core.HashBytes([]byte(strings.Join(parts, "\x00")))
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
