package ublang

import (
	"encoding/json"
	"sort"
	"strings"

	"go-safedesign/internal/core"
	"go-safedesign/internal/pipeline"
	"go-safedesign/internal/wordcase"
)

const (
	ID      = "ubiquitous_language"
	Version = "prototype-1"

	AlignmentMetricName = "ubiquitous_language_alignment"
	MetricUnit          = "percent"
)

type Config struct {
	Contexts     []ContextConfig `json:"contexts"`
	IgnoredTerms []string        `json:"ignoredTerms"`
}

type ContextConfig struct {
	ID               string            `json:"id"`
	PackagePrefixes  []string          `json:"packagePrefixes"`
	Terms            []string          `json:"terms"`
	Synonyms         map[string]string `json:"synonyms"`
	DiscouragedTerms []string          `json:"discouragedTerms"`
}

type Analyzer struct{}

type normalizedContext struct {
	ID          string
	Prefixes    []string
	Terms       map[string]bool
	Synonyms    map[string]string
	Discouraged map[string]bool
}

func (Analyzer) Metadata() pipeline.AnalyzerMetadata {
	return pipeline.AnalyzerMetadata{
		ID:                   ID,
		Version:              Version,
		Stage:                pipeline.StageDDDClassification,
		InputFactKinds:       []string{"module", "package", "function", "method", "type", "interface", "struct", "field"},
		MinimumRequiredTrust: core.TrustSyntaxObserved,
		MaximumEmittedTrust:  core.TrustTypeResolved,
		EmittedFactKinds:     []string{"label", "metric", "warning"},
		ConfigurationSection: "ubiquitousLanguage",
		FailureMode:          "partial",
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
	labels, metrics, warnings := a.evaluate(context.Graph, cfg, context.ConfigurationHash)
	return pipeline.AnalyzerResult{Labels: labels, Metrics: metrics, Warnings: warnings}, nil
}

func parseConfig(raw []byte) (Config, error) {
	if len(raw) == 0 {
		return Config{}, nil
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (Analyzer) evaluate(graph core.Graph, cfg Config, configHash string) ([]core.Label, []core.Metric, []core.Warning) {
	contexts := normalizeContexts(cfg.Contexts)
	if len(contexts) == 0 {
		return nil, nil, nil
	}
	ignored := normalizeSet(cfg.IgnoredTerms)
	packages := packageNodes(graph)
	nodesByPackage := scopedIdentityNodes(graph)

	var labels []core.Label
	var metrics []core.Metric
	var warnings []core.Warning
	for _, pkg := range packages {
		ctx, ok := contextForPackage(pkg.PackagePath, contexts)
		if !ok {
			continue
		}
		labels = append(labels, contextLabelFor(pkg, ctx))

		recognized := map[string]bool{}
		unknown := map[string]string{}
		discouraged := map[string]string{}
		considered := map[string]bool{}
		for _, node := range append([]core.Node{pkg}, nodesByPackage[pkg.PackagePath]...) {
			for _, raw := range wordsFromNode(node) {
				term := canonicalTerm(raw, ctx)
				if term == "" || ignored[term] {
					continue
				}
				considered[term] = true
				switch {
				case ctx.Discouraged[term]:
					discouraged[term] = node.ID
				case ctx.Terms[term]:
					recognized[term] = true
				default:
					unknown[term] = node.ID
				}
			}
		}
		for term := range recognized {
			labels = append(labels, termLabelFor(pkg, term, ctx))
		}
		for term, nodeID := range discouraged {
			warnings = append(warnings, termWarningFor("ul_discouraged_term", pkg, term, nodeID, "discouraged ubiquitous language term "+term))
		}
		for term, nodeID := range unknown {
			warnings = append(warnings, termWarningFor("ul_unknown_term", pkg, term, nodeID, "unknown ubiquitous language term "+term))
		}
		metrics = append(metrics, alignmentMetricFor(pkg, len(recognized), len(considered), configHash))
	}
	sort.Slice(labels, func(i, j int) bool { return labels[i].ID < labels[j].ID })
	sort.Slice(metrics, func(i, j int) bool { return metrics[i].ID < metrics[j].ID })
	sort.Slice(warnings, func(i, j int) bool { return warnings[i].ID < warnings[j].ID })
	return labels, metrics, warnings
}

func normalizeContexts(configs []ContextConfig) []normalizedContext {
	var out []normalizedContext
	for _, cfg := range configs {
		if cfg.ID == "" {
			continue
		}
		ctx := normalizedContext{
			ID:          strings.ToLower(cfg.ID),
			Prefixes:    cfg.PackagePrefixes,
			Terms:       normalizeSet(cfg.Terms),
			Synonyms:    map[string]string{},
			Discouraged: normalizeSet(cfg.DiscouragedTerms),
		}
		for from, to := range cfg.Synonyms {
			ctx.Synonyms[strings.ToLower(from)] = strings.ToLower(to)
		}
		out = append(out, ctx)
	}
	return out
}

func packageNodes(graph core.Graph) []core.Node {
	var nodes []core.Node
	for _, node := range graph.Nodes {
		if node.Kind == "package" && node.PackagePath != "" {
			nodes = append(nodes, node)
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	return nodes
}

func scopedIdentityNodes(graph core.Graph) map[string][]core.Node {
	out := map[string][]core.Node{}
	for _, node := range graph.Nodes {
		switch node.Kind {
		case "module", "function", "method", "type", "interface", "struct", "field":
		default:
			continue
		}
		if node.PackagePath == "" {
			continue
		}
		out[node.PackagePath] = append(out[node.PackagePath], node)
	}
	for key := range out {
		sort.Slice(out[key], func(i, j int) bool { return out[key][i].ID < out[key][j].ID })
	}
	return out
}

func contextForPackage(pkgPath string, contexts []normalizedContext) (normalizedContext, bool) {
	best := normalizedContext{}
	bestLen := -1
	for _, ctx := range contexts {
		for _, prefix := range ctx.Prefixes {
			if pkgPath == prefix || strings.HasPrefix(pkgPath, prefix+"/") {
				if len(prefix) > bestLen {
					best = ctx
					bestLen = len(prefix)
				}
			}
		}
	}
	return best, bestLen >= 0
}

func wordsFromNode(node core.Node) []string {
	if node.Kind == "package" {
		return SplitWords(node.PackagePath)
	}
	return SplitWords(node.Name)
}

func SplitWords(value string) []string {
	return wordcase.SplitWords(value)
}

func canonicalTerm(raw string, ctx normalizedContext) string {
	term := strings.ToLower(raw)
	if mapped := ctx.Synonyms[term]; mapped != "" {
		return mapped
	}
	return term
}

func contextLabelFor(pkg core.Node, ctx normalizedContext) core.Label {
	return core.Label{
		ID:         "label:ddd.context:" + pkg.ID,
		Kind:       "label",
		Name:       "ddd.context",
		Value:      ctx.ID,
		TargetID:   pkg.ID,
		TargetKind: "node",
		Source:     "configured",
		Evidence:   []string{pkg.ID},
		TrustLevel: pkg.TrustLevel,
		Freshness:  "fresh",
	}
}

func termLabelFor(pkg core.Node, term string, ctx normalizedContext) core.Label {
	return core.Label{
		ID:         "label:ul.term:" + pkg.ID + ":" + term,
		Kind:       "label",
		Name:       "ul.term",
		Value:      term,
		TargetID:   pkg.ID,
		TargetKind: "node",
		Source:     "observed",
		Evidence:   []string{pkg.ID, "context:" + ctx.ID},
		TrustLevel: pkg.TrustLevel,
		Freshness:  "fresh",
	}
}

func alignmentMetricFor(pkg core.Node, recognized, considered int, configHash string) core.Metric {
	value := 100
	if considered > 0 {
		value = recognized * 100 / considered
	}
	return core.Metric{
		ID:                "metric:" + AlignmentMetricName + ":" + pkg.ID,
		Kind:              "metric",
		Name:              AlignmentMetricName,
		Value:             value,
		Unit:              MetricUnit,
		Scope:             pkg.PackagePath,
		Subject:           pkg.ID,
		AnalyzerID:        ID,
		Stage:             string(pipeline.StageDDDClassification),
		Status:            "pass",
		Reason:            "ubiquitous language alignment",
		Evidence:          []string{pkg.ID},
		TrustLevel:        pkg.TrustLevel,
		ConfigurationHash: configHash,
		SourceFile:        pkg.SourceFile,
		LineRange:         pkg.LineRange,
	}
}

func termWarningFor(kind string, pkg core.Node, term, evidenceID, reason string) core.Warning {
	return core.Warning{
		ID:                  "warning:" + kind + ":" + pkg.ID + ":" + term,
		Kind:                kind,
		Reason:              reason,
		SuggestedNextAction: "Update the configured ubiquitous language terms or rename the code element to match the context language.",
		AffectedNodeID:      evidenceID,
		Evidence:            []string{pkg.ID, evidenceID},
		TrustLevel:          pkg.TrustLevel,
		SourceFile:          pkg.SourceFile,
		LineRange:           pkg.LineRange,
		Freshness:           "fresh",
	}
}

func normalizeSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			out[value] = true
		}
	}
	return out
}
