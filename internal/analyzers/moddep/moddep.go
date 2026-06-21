package moddep

import (
	"encoding/json"
	"sort"

	"go-safedesign/internal/core"
	"go-safedesign/internal/pipeline"
)

const (
	ID      = "module_dependency_enrichment"
	Version = "prototype-1"
)

type Config struct {
	CheckOutdatedModules bool   `json:"checkOutdatedModules"`
	NetworkAccess        string `json:"networkAccess"`
}

type Analyzer struct{}

func (Analyzer) Metadata() pipeline.AnalyzerMetadata {
	return pipeline.AnalyzerMetadata{
		ID:                    ID,
		Version:               Version,
		Stage:                 pipeline.StageModuleDependencyEnrichment,
		InputFactKinds:        []string{"module", "depends_on", "diagnostic"},
		MinimumRequiredTrust:  core.TrustSyntaxObserved,
		MaximumEmittedTrust:   core.TrustSyntaxObserved,
		EmittedFactKinds:      []string{"label", "warning"},
		ConfigurationSection:  "moduleDependencyEnrichment",
		FailureMode:           "partial",
		IncompleteInputPolicy: pipeline.IncompleteInputAllow,
	}
}

func Metadata() pipeline.AnalyzerMetadata {
	return Analyzer{}.Metadata()
}

func (a Analyzer) Run(context pipeline.GraphContext) (pipeline.AnalyzerResult, error) {
	cfg, err := parseConfig(context.Configuration)
	if err != nil {
		return pipeline.AnalyzerResult{}, err
	}
	labels, warnings := a.evaluate(context.Graph, cfg)
	return pipeline.AnalyzerResult{Labels: labels, Warnings: warnings}, nil
}

func parseConfig(raw []byte) (Config, error) {
	cfg := Config{NetworkAccess: "disabled"}
	if len(raw) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.NetworkAccess == "" {
		cfg.NetworkAccess = "disabled"
	}
	return cfg, nil
}

func (Analyzer) evaluate(graph core.Graph, cfg Config) ([]core.Label, []core.Warning) {
	var labels []core.Label
	var warnings []core.Warning
	for _, edge := range graph.Edges {
		if edge.Kind != "depends_on" {
			continue
		}
		labels = append(labels, dependencyLabelFor(edge))
		if cfg.CheckOutdatedModules {
			warnings = append(warnings, outdatedWarningFor(edge, cfg))
		}
	}
	for _, diagnostic := range graph.Diagnostics {
		if diagnostic.Status != "missing_dependency" {
			continue
		}
		warnings = append(warnings, missingDependencyWarningFor(diagnostic))
	}
	sort.Slice(labels, func(i, j int) bool { return labels[i].ID < labels[j].ID })
	sort.Slice(warnings, func(i, j int) bool { return warnings[i].ID < warnings[j].ID })
	return labels, warnings
}

func dependencyLabelFor(edge core.Edge) core.Label {
	return core.Label{
		ID:         "label:module.dependency:" + edge.ID,
		Kind:       "label",
		Name:       "module.dependency",
		Value:      "direct",
		TargetID:   edge.ID,
		TargetKind: "edge",
		Source:     "observed",
		Evidence:   []string{edge.ID},
		TrustLevel: edge.TrustLevel,
		Freshness:  "fresh",
	}
}

func missingDependencyWarningFor(diagnostic core.Diagnostic) core.Warning {
	evidence := diagnostic.Source + ":" + diagnostic.Reason
	return core.Warning{
		ID:                  "warning:missing_dependency:" + core.HashBytes([]byte(evidence)),
		Kind:                "missing_dependency",
		Reason:              diagnostic.Reason,
		SuggestedNextAction: "Run go mod tidy, add the dependency, or resolve the import path.",
		Evidence:            []string{evidence},
		TrustLevel:          core.TrustSyntaxObserved,
		SourceFile:          diagnostic.SourceFile,
		LineRange:           diagnostic.LineRange,
		Freshness:           "fresh",
	}
}

func outdatedWarningFor(edge core.Edge, cfg Config) core.Warning {
	reason := "outdated module check unknown because module metadata cache is not available"
	if cfg.NetworkAccess == "disabled" {
		reason = "outdated module check unknown because network access is disabled"
	}
	return core.Warning{
		ID:                  "warning:outdated_module:" + core.HashBytes([]byte(edge.ID)),
		Kind:                "outdated_module",
		Reason:              reason,
		SuggestedNextAction: "Provide module metadata or enable a future metadata fetch implementation before treating this dependency as current.",
		AffectedEdgeID:      edge.ID,
		Evidence:            []string{edge.ID},
		TrustLevel:          edge.TrustLevel,
		SourceFile:          edge.SourceFile,
		LineRange:           edge.LineRange,
		Freshness:           "fresh",
	}
}
