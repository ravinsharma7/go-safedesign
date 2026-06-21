package pipeline

import (
	"fmt"
	"strings"

	"github.com/ravinsharma7/go-safedesign/internal/core"
)

func ValidateAnalyzerMetadata(metadata AnalyzerMetadata) []core.Diagnostic {
	validation := metadataValidation{metadata: metadata}
	validation.require("id", metadata.ID)
	validation.require("version", metadata.Version)
	validation.requireStage()
	validation.requireFactKinds("input fact kinds", metadata.InputFactKinds, nil)
	validation.requireTrust("minimum required trust", metadata.MinimumRequiredTrust)
	validation.requireTrust("maximum emitted trust", metadata.MaximumEmittedTrust)
	validation.validateTrustRange()
	validation.requireFactKinds("emitted fact kinds", metadata.EmittedFactKinds, isSupportedEmittedFactKind)
	validation.requireFailureMode()
	validation.requireIncompleteInputPolicy()
	return validation.diagnostics
}

type metadataValidation struct {
	metadata    AnalyzerMetadata
	diagnostics []core.Diagnostic
}

func (v *metadataValidation) add(reason string) {
	v.diagnostics = append(v.diagnostics, core.Diagnostic{
		Level:      "error",
		Source:     "analyzer:" + v.metadata.ID,
		Reason:     reason,
		Stage:      string(v.metadata.Stage),
		AnalyzerID: v.metadata.ID,
		Status:     core.StatusAnalysisError,
		TrustLevel: v.metadata.MaximumEmittedTrust,
	})
}

func (v *metadataValidation) require(name, value string) {
	if value == "" {
		v.add("analyzer metadata missing " + name)
	}
}

func (v *metadataValidation) requireStage() {
	if v.metadata.Stage == "" {
		v.add("analyzer metadata missing stage")
		return
	}
	if !isSupportedStage(v.metadata.Stage) {
		v.add("analyzer metadata has unsupported stage " + string(v.metadata.Stage))
	}
}

func (v *metadataValidation) requireFactKinds(name string, kinds []string, supported func(string) bool) {
	if len(kinds) == 0 {
		v.add("analyzer metadata missing " + name)
		return
	}
	if supported == nil {
		return
	}
	for _, kind := range kinds {
		if !supported(kind) {
			v.add("analyzer metadata has unsupported " + singularFactKindName(name) + " " + kind)
		}
	}
}

func (v *metadataValidation) requireTrust(name string, trust core.TrustLevel) {
	if trust == "" {
		v.add("analyzer metadata missing " + name)
		return
	}
	if !isSupportedTrust(trust) {
		v.add("analyzer metadata has unsupported " + name + " " + string(trust))
	}
}

func (v *metadataValidation) validateTrustRange() {
	minimum := v.metadata.MinimumRequiredTrust
	maximum := v.metadata.MaximumEmittedTrust
	if isSupportedTrust(minimum) && isSupportedTrust(maximum) && core.TrustRank(minimum) > core.TrustRank(maximum) {
		v.add(fmt.Sprintf("analyzer metadata minimum required trust %s exceeds maximum emitted trust %s", minimum, maximum))
	}
}

func (v *metadataValidation) requireFailureMode() {
	if v.metadata.FailureMode == "" {
		v.add("analyzer metadata missing failure mode")
		return
	}
	if v.metadata.FailureMode != FailureModePartial {
		v.add("analyzer metadata has unsupported failure mode " + v.metadata.FailureMode)
	}
}

func (v *metadataValidation) requireIncompleteInputPolicy() {
	if v.metadata.IncompleteInputPolicy == "" {
		v.add("analyzer metadata missing incomplete input policy")
		return
	}
	if !isSupportedIncompleteInputPolicy(v.metadata.IncompleteInputPolicy) {
		v.add("analyzer metadata has unsupported incomplete input policy " + string(v.metadata.IncompleteInputPolicy))
	}
}

func singularFactKindName(name string) string {
	if name == "emitted fact kinds" {
		return "emitted fact kind"
	}
	return name
}

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
			Status:     core.StatusAnalysisError,
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
		validateCommon(add, metadata, known, core.FactKindObservation, observation.ID, observation.Kind, observation.TrustLevel, observation.Evidence)
		if observation.Name == "" {
			add(core.FactKindObservation + " " + observation.ID + " missing name")
		}
		validateTarget(add, known, core.FactKindObservation, observation.ID, observation.TargetID, observation.TargetKind)
		validateObservationSource(add, observation)
	}
	for _, label := range result.Labels {
		validateCommon(add, metadata, known, core.FactKindLabel, label.ID, label.Kind, label.TrustLevel, label.Evidence)
		validateTarget(add, known, core.FactKindLabel, label.ID, label.TargetID, label.TargetKind)
	}
	for _, warning := range result.Warnings {
		validateCommon(add, metadata, known, core.FactKindWarning, warning.ID, warning.Kind, warning.TrustLevel, warning.Evidence)
		if warning.AffectedNodeID != "" && !known[warning.AffectedNodeID] {
			add("warning " + warning.ID + " references missing affected node " + warning.AffectedNodeID)
		}
		if warning.AffectedEdgeID != "" && !known[warning.AffectedEdgeID] {
			add("warning " + warning.ID + " references missing affected edge " + warning.AffectedEdgeID)
		}
	}
	for _, metric := range result.Metrics {
		validateCommon(add, metadata, known, core.FactKindMetric, metric.ID, metric.Kind, metric.TrustLevel, metric.Evidence)
		if metric.Subject == "" {
			add("metric " + metric.ID + " missing subject")
		} else if looksLikeFactID(metric.Subject) && !known[metric.Subject] {
			add("metric " + metric.ID + " references missing subject " + metric.Subject)
		}
	}
	for _, policy := range result.PolicyResults {
		validateCommon(add, metadata, known, core.FactKindPolicyResult, policy.ID, policy.Kind, policy.TrustLevel, policy.Evidence)
	}
	for _, edge := range result.Edges {
		validateCommon(add, metadata, known, core.FactKindEdge, edge.ID, edge.Kind, edge.TrustLevel, nil)
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
	if targetKind != "" && targetKind != "node" && targetKind != core.FactKindEdge && targetKind != core.FactKindPolicyResult && targetKind != core.FactKindMetric && targetKind != core.FactKindObservation {
		add(factType + " " + id + " has unsupported target kind " + targetKind)
	}
}

func validateObservationSource(add func(string), observation core.Observation) {
	switch {
	case core.IsObservationSource(observation.Source):
	case observation.Source == "":
		add("observation " + observation.ID + " missing source")
	default:
		add("observation " + observation.ID + " has unsupported source " + observation.Source)
	}
}

func isSupportedStage(stage Stage) bool {
	switch stage {
	case StageSourceDiscovery,
		StageModuleExtraction,
		StageSyntaxExtraction,
		StageBaseGraphAssembly,
		StagePackageLoading,
		StageTypeResolution,
		StageModuleDependencyEnrichment,
		StageThirdPartyBehaviorLabeling,
		StageFrameworkExtraction,
		StageDDDClassification,
		StageComplexityMetrics,
		StagePolicyEvaluation,
		StageQueryMaterialization,
		StageRendering:
		return true
	default:
		return false
	}
}

func isSupportedTrust(trust core.TrustLevel) bool {
	return core.TrustRank(trust) > 0
}

func isSupportedEmittedFactKind(kind string) bool {
	switch kind {
	case core.FactKindLabel,
		core.FactKindMetric,
		core.FactKindWarning,
		core.FactKindPolicyResult,
		core.FactKindObservation,
		core.FactKindDiagnostic,
		core.FactKindEdge:
		return true
	default:
		return false
	}
}

func isSupportedIncompleteInputPolicy(policy IncompleteInputPolicy) bool {
	switch policy {
	case IncompleteInputRequireComplete, IncompleteInputSkipScope, IncompleteInputAllow:
		return true
	default:
		return false
	}
}

func emittedFactKinds(result AnalyzerResult) map[string]bool {
	out := map[string]bool{}
	if len(result.Diagnostics) > 0 {
		out[core.FactKindDiagnostic] = true
	}
	if len(result.PolicyResults) > 0 {
		out[core.FactKindPolicyResult] = true
	}
	if len(result.Metrics) > 0 {
		out[core.FactKindMetric] = true
	}
	if len(result.Observations) > 0 {
		out[core.FactKindObservation] = true
	}
	if len(result.Labels) > 0 {
		out[core.FactKindLabel] = true
	}
	if len(result.Warnings) > 0 {
		out[core.FactKindWarning] = true
	}
	if len(result.Edges) > 0 {
		out[core.FactKindEdge] = true
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
		core.IDPrefixModule,
		core.IDPrefixPackage,
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
		core.IDPrefixPlaceholder,
		core.IDPrefixEdge,
		core.FactKindLabel + ":",
		core.FactKindWarning + ":",
		core.FactKindMetric + ":",
		core.FactKindPolicyResult + ":",
		core.IDPrefixObservation,
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
