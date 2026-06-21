package core

import (
	"reflect"
	"sort"
	"strings"
)

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
	ObservationNameBridgeSymbol                   = "vocabulary.bridge_symbol"
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

func NodeKinds() []string {
	return sortedStrings([]string{
		NodeKindModule,
		NodeKindPackage,
		NodeKindFile,
		NodeKindFunction,
		NodeKindMethod,
		NodeKindType,
		NodeKindInterface,
		NodeKindStruct,
		NodeKindField,
		NodeKindImport,
		NodeKindPlaceholder,
		NodeKindRuntimeMarker,
		NodeKindUnresolvedCall,
	})
}

func EdgeKinds() []string {
	return sortedStrings([]string{
		EdgeKindContains,
		EdgeKindDeclares,
		EdgeKindImports,
		EdgeKindCalls,
		EdgeKindDependsOn,
		EdgeKindViolates,
		EdgeKindContainsRuntimeMarker,
	})
}

func FactKinds() []string {
	return sortedStrings([]string{
		FactKindLabel,
		FactKindMetric,
		FactKindWarning,
		FactKindPolicyResult,
		FactKindObservation,
		FactKindDiagnostic,
		FactKindEdge,
	})
}

func Statuses() []string {
	return sortedStrings([]string{
		StatusPass,
		StatusFail,
		StatusWarning,
		StatusUnknown,
		StatusCompleted,
		StatusPartial,
		StatusAnalysisError,
	})
}

func FreshnessStatuses() []string {
	return sortedStrings([]string{
		FreshnessFresh,
		FreshnessStale,
		FreshnessSuperseded,
		FreshnessInvalidated,
	})
}

func ObservationNames() []string {
	return sortedStrings([]string{
		ObservationNameVocabularyTerm,
		ObservationNameVocabularyIncompleteDependency,
		ObservationNameVocabularyCooccurrence,
		ObservationNameLanguageZoneCandidate,
		ObservationNameBridgeSymbol,
	})
}

func ObservationSources() []string {
	return sortedStrings([]string{
		ObservationSourceObserved,
		ObservationSourceConfigured,
		ObservationSourceInferred,
		ObservationSourceImported,
	})
}

func GraphJSONSections() []string {
	graphType := reflect.TypeOf(Graph{})
	sections := make([]string, 0, graphType.NumField())
	for i := 0; i < graphType.NumField(); i++ {
		tag := graphType.Field(i).Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name != "" && name != "-" {
			sections = append(sections, name)
		}
	}
	return sections
}

func CanonicalGraphJSONSection(value string) (string, bool) {
	normalized := normalizeDiscoveryValue(value)
	for _, section := range GraphJSONSections() {
		if normalizeDiscoveryValue(section) == normalized {
			return section, true
		}
	}
	return "", false
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func normalizeDiscoveryValue(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, " ", "")
	return value
}
