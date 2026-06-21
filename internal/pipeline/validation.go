package pipeline

import (
	"fmt"
	"strings"

	"go-safedesign/internal/core"
)

func ValidateAnalyzerResult(graph core.Graph, metadata AnalyzerMetadata, result AnalyzerResult) []core.Diagnostic {
	known := knownFactIDs(graph, result)
	emitted := emittedFactKinds(result)
	allowed := map[string]bool{}
	for _, kind := range metadata.EmittedFactKinds {
		allowed[kind] = true
	}

	var diagnostics []core.Diagnostic
	add := func(reason string) {
		diagnostics = append(diagnostics, core.Diagnostic{
			Level:      "error",
			Source:     "analyzer:" + metadata.ID,
			Reason:     reason,
			Stage:      string(metadata.Stage),
			AnalyzerID: metadata.ID,
			Status:     "analysis_error",
			TrustLevel: metadata.MaximumEmittedTrust,
		})
	}
	for kind := range emitted {
		if !allowed[kind] {
			add("analyzer emitted undeclared fact kind " + kind)
		}
	}

	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Source == "" {
			add("diagnostic missing source")
		}
		if diagnostic.Reason == "" {
			add("diagnostic missing reason")
		}
	}
	for _, observation := range result.Observations {
		validateCommon(add, metadata, known, "observation", observation.ID, observation.Kind, observation.TrustLevel, observation.Evidence)
		if observation.Name == "" {
			add("observation " + observation.ID + " missing name")
		}
		validateTarget(add, known, "observation", observation.ID, observation.TargetID, observation.TargetKind)
		validateObservationSource(add, observation)
	}
	for _, label := range result.Labels {
		validateCommon(add, metadata, known, "label", label.ID, label.Kind, label.TrustLevel, label.Evidence)
		validateTarget(add, known, "label", label.ID, label.TargetID, label.TargetKind)
	}
	for _, warning := range result.Warnings {
		validateCommon(add, metadata, known, "warning", warning.ID, warning.Kind, warning.TrustLevel, warning.Evidence)
		if warning.AffectedNodeID != "" && !known[warning.AffectedNodeID] {
			add("warning " + warning.ID + " references missing affected node " + warning.AffectedNodeID)
		}
		if warning.AffectedEdgeID != "" && !known[warning.AffectedEdgeID] {
			add("warning " + warning.ID + " references missing affected edge " + warning.AffectedEdgeID)
		}
	}
	for _, metric := range result.Metrics {
		validateCommon(add, metadata, known, "metric", metric.ID, metric.Kind, metric.TrustLevel, metric.Evidence)
		if metric.Subject == "" {
			add("metric " + metric.ID + " missing subject")
		} else if looksLikeFactID(metric.Subject) && !known[metric.Subject] {
			add("metric " + metric.ID + " references missing subject " + metric.Subject)
		}
	}
	for _, policy := range result.PolicyResults {
		validateCommon(add, metadata, known, "policy_result", policy.ID, policy.Kind, policy.TrustLevel, policy.Evidence)
	}
	for _, edge := range result.Edges {
		validateCommon(add, metadata, known, "edge", edge.ID, edge.Kind, edge.TrustLevel, nil)
		if edge.From == "" || edge.To == "" {
			add("edge " + edge.ID + " missing endpoints")
			continue
		}
		if !known[edge.From] {
			add("edge " + edge.ID + " references missing from node " + edge.From)
		}
		if !known[edge.To] {
			add("edge " + edge.ID + " references missing to node " + edge.To)
		}
	}
	return diagnostics
}

func validateCommon(add func(string), metadata AnalyzerMetadata, known map[string]bool, factType, id, kind string, trust core.TrustLevel, evidence []string) {
	if id == "" {
		add(factType + " missing id")
	}
	if kind == "" {
		add(factType + " " + id + " missing kind")
	}
	if trust == "" {
		add(factType + " " + id + " missing trust level")
	}
	if metadata.MaximumEmittedTrust != "" && core.TrustRank(trust) > core.TrustRank(metadata.MaximumEmittedTrust) {
		add(fmt.Sprintf("%s %s trust %s exceeds analyzer maximum %s", factType, id, trust, metadata.MaximumEmittedTrust))
	}
	for _, ref := range evidence {
		if looksLikeFactID(ref) && !known[ref] {
			add(factType + " " + id + " references missing evidence " + ref)
		}
	}
}

func validateTarget(add func(string), known map[string]bool, factType, id, targetID, targetKind string) {
	if targetID == "" {
		return
	}
	if !known[targetID] {
		add(factType + " " + id + " references missing target " + targetID)
		return
	}
	if targetKind != "" && targetKind != "node" && targetKind != "edge" && targetKind != "policy_result" && targetKind != "metric" && targetKind != "observation" {
		add(factType + " " + id + " has unsupported target kind " + targetKind)
	}
}

func validateObservationSource(add func(string), observation core.Observation) {
	switch observation.Source {
	case "observed", "configured", "inferred", "imported":
	case "":
		add("observation " + observation.ID + " missing source")
	default:
		add("observation " + observation.ID + " has unsupported source " + observation.Source)
	}
}

func emittedFactKinds(result AnalyzerResult) map[string]bool {
	out := map[string]bool{}
	if len(result.Diagnostics) > 0 {
		out["diagnostic"] = true
	}
	if len(result.PolicyResults) > 0 {
		out["policy_result"] = true
	}
	if len(result.Metrics) > 0 {
		out["metric"] = true
	}
	if len(result.Observations) > 0 {
		out["observation"] = true
	}
	if len(result.Labels) > 0 {
		out["label"] = true
	}
	if len(result.Warnings) > 0 {
		out["warning"] = true
	}
	if len(result.Edges) > 0 {
		out["edge"] = true
	}
	return out
}

func knownFactIDs(graph core.Graph, result AnalyzerResult) map[string]bool {
	known := map[string]bool{}
	for _, node := range graph.Nodes {
		known[node.ID] = true
	}
	for _, edge := range graph.Edges {
		known[edge.ID] = true
	}
	for _, source := range graph.SourceRecords {
		known[source.ID] = true
	}
	for _, observation := range graph.Observations {
		known[observation.ID] = true
	}
	for _, label := range graph.Labels {
		known[label.ID] = true
	}
	for _, warning := range graph.Warnings {
		known[warning.ID] = true
	}
	for _, query := range graph.Queries {
		known[query.ID] = true
	}
	for _, path := range graph.PathJobs {
		known[path.ID] = true
	}
	for _, policy := range graph.PolicyResults {
		known[policy.ID] = true
	}
	for _, metric := range graph.Metrics {
		known[metric.ID] = true
	}
	for _, diagnostic := range graph.Diagnostics {
		if diagnostic.Source != "" && diagnostic.Reason != "" {
			known[diagnostic.Source+":"+diagnostic.Reason] = true
		}
	}
	for _, observation := range result.Observations {
		known[observation.ID] = true
	}
	for _, label := range result.Labels {
		known[label.ID] = true
	}
	for _, warning := range result.Warnings {
		known[warning.ID] = true
	}
	for _, metric := range result.Metrics {
		known[metric.ID] = true
	}
	for _, policy := range result.PolicyResults {
		known[policy.ID] = true
	}
	for _, edge := range result.Edges {
		known[edge.ID] = true
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Source != "" && diagnostic.Reason != "" {
			known[diagnostic.Source+":"+diagnostic.Reason] = true
		}
	}
	return known
}

func looksLikeFactID(value string) bool {
	prefixes := []string{
		"module:",
		"package:",
		"file:",
		"function:",
		"method:",
		"type:",
		"interface:",
		"struct:",
		"field:",
		"import:",
		"runtime_marker:",
		"unresolved_call:",
		"placeholder:",
		"edge:",
		"label:",
		"warning:",
		"metric:",
		"policy_result:",
		"observation:",
		"query:",
		"path:",
		"source_record:",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
