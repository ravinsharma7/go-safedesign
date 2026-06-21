package core

import "strings"

const (
	NodeKindModule         = "module"
	NodeKindPackage        = "package"
	NodeKindFile           = "file"
	NodeKindFunction       = "function"
	NodeKindMethod         = "method"
	NodeKindType           = "type"
	NodeKindInterface      = "interface"
	NodeKindStruct         = "struct"
	NodeKindField          = "field"
	NodeKindImport         = "import"
	NodeKindPlaceholder    = "placeholder"
	NodeKindRuntimeMarker  = "runtime_marker"
	NodeKindUnresolvedCall = "unresolved_call"
)

const (
	EdgeKindContains              = "contains"
	EdgeKindDeclares              = "declares"
	EdgeKindImports               = "imports"
	EdgeKindCalls                 = "calls"
	EdgeKindDependsOn             = "depends_on"
	EdgeKindViolates              = "violates"
	EdgeKindContainsRuntimeMarker = "contains_runtime_marker"
)

const (
	FactKindLabel        = "label"
	FactKindMetric       = "metric"
	FactKindWarning      = "warning"
	FactKindPolicyResult = "policy_result"
	FactKindObservation  = "observation"
	FactKindDiagnostic   = "diagnostic"
	FactKindEdge         = "edge"
)

const (
	StatusPass          = "pass"
	StatusFail          = "fail"
	StatusWarning       = "warning"
	StatusUnknown       = "unknown"
	StatusCompleted     = "completed"
	StatusPartial       = "partial"
	StatusAnalysisError = "analysis_error"
)

const (
	DiagnosticStatusMissingDependency         = "missing_dependency"
	DiagnosticStatusImportCycle               = "import_cycle"
	DiagnosticStatusPackageLoadingDiagnostic  = "package_loading_diagnostic"
	DiagnosticStatusAnalysisSkippedIncomplete = "analysis_skipped_incomplete_scope"
)

const (
	FreshnessFresh       = "fresh"
	FreshnessStale       = "stale"
	FreshnessSuperseded  = "superseded"
	FreshnessInvalidated = "invalidated"
)

const (
	ObservationSourceObserved   = "observed"
	ObservationSourceConfigured = "configured"
	ObservationSourceInferred   = "inferred"
	ObservationSourceImported   = "imported"

	ObservationNameVocabularyTerm                 = "vocabulary.term"
	ObservationNameVocabularyIncompleteDependency = "vocabulary.incomplete_dependency"
	ObservationNameVocabularyCooccurrence         = "vocabulary.cooccurrence"
	ObservationNameLanguageZoneCandidate          = "vocabulary.language_zone_candidate"
)

const (
	IDPrefixPackage            = "package:"
	IDPrefixModule             = "module:"
	IDPrefixPlaceholder        = "placeholder:"
	IDPrefixPlaceholderPackage = "placeholder:package:"
	IDPrefixPlaceholderModule  = "placeholder:module:"
	IDPrefixEdge               = "edge:"
	IDPrefixObservation        = "observation:"
)

func PackageID(path string) string {
	return IDPrefixPackage + path
}

func ModuleID(path string) string {
	return IDPrefixModule + path
}

func PlaceholderPackageID(path string) string {
	return IDPrefixPlaceholderPackage + path
}

func PlaceholderModuleID(path string) string {
	return IDPrefixPlaceholderModule + path
}

func EdgeID(kind, from, to string) string {
	return IDPrefixEdge + kind + ":" + from + "->" + to
}

func IsPlaceholderID(id string) bool {
	return strings.HasPrefix(id, IDPrefixPlaceholder)
}

func IsPlaceholderNode(node Node) bool {
	return node.Synthetic || node.Kind == NodeKindPlaceholder || IsPlaceholderID(node.ID)
}

func IsPlaceholderTarget(edge Edge) bool {
	return IsPlaceholderID(edge.To)
}

func IsIncompleteEdge(edge Edge) bool {
	return !edge.Complete || edge.Synthetic || IsPlaceholderTarget(edge)
}

func IsObservationSource(value string) bool {
	switch value {
	case ObservationSourceObserved, ObservationSourceConfigured, ObservationSourceInferred, ObservationSourceImported:
		return true
	default:
		return false
	}
}
