package indexer

import (
	"strings"

	"go-safedesign/internal/core"
	"go-safedesign/internal/pipeline"
)

func (b *graphBuilder) readinessDiagnostics(metadata pipeline.AnalyzerMetadata) []core.Diagnostic {
	policy := metadata.IncompleteInputPolicy
	if policy == "" {
		policy = pipeline.IncompleteInputRequireComplete
	}
	if policy == pipeline.IncompleteInputAllow {
		return nil
	}
	incomplete := b.incompleteInputFacts(metadata.InputFactKinds)
	if len(incomplete) == 0 {
		return nil
	}
	reason := "analyzer input scope incomplete"
	status := core.DiagnosticStatusAnalysisSkippedIncomplete
	if policy == pipeline.IncompleteInputRequireComplete {
		reason = "analyzer requires complete input scope"
	}
	return []core.Diagnostic{{
		Level:      "warning",
		Source:     "analyzer:" + metadata.ID,
		Reason:     reason + ": " + strings.Join(incomplete, ", "),
		Stage:      string(metadata.Stage),
		AnalyzerID: metadata.ID,
		Status:     status,
		TrustLevel: metadata.MinimumRequiredTrust,
		Evidence:   incomplete,
	}}
}

func (b *graphBuilder) incompleteInputFacts(kinds []string) []string {
	wanted := map[string]bool{}
	for _, kind := range kinds {
		wanted[kind] = true
	}
	var out []string
	for _, node := range b.nodes {
		if !wanted[node.Kind] {
			continue
		}
		if core.IsPlaceholderNode(node) {
			out = append(out, node.ID)
		}
	}
	for _, edge := range b.edges {
		if !wanted[edge.Kind] {
			continue
		}
		if core.IsIncompleteEdge(edge) {
			out = append(out, edge.ID)
		}
	}
	return out
}

func diagnosticMessages(diagnostics []core.Diagnostic) []string {
	out := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		out = append(out, diagnostic.Source+": "+diagnostic.Reason)
	}
	return out
}
