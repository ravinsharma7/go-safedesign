package indexer

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go-safedesign/internal/core"
	srcutil "go-safedesign/internal/source"
)

type complexityScopeMode string

const (
	complexityScopePackage complexityScopeMode = "package"
	complexityScopeDomain  complexityScopeMode = "domain"
)

func (b *graphBuilder) addQueriesAndPathJobs() {
	for _, scope := range b.policyQueryScopes() {
		b.queries = append(b.queries, b.policyQuery(scope))
	}
	for _, scope := range b.complexityPackageQueryScopes() {
		b.queries = append(b.queries, b.complexityQuery(scope, complexityScopePackage))
	}
	for _, scope := range b.complexityDomainQueryScopes() {
		b.queries = append(b.queries, b.complexityQuery(scope, complexityScopeDomain))
	}
}

func (b *graphBuilder) policyQueryScopes() []string {
	scopes := map[string]bool{}
	for _, result := range b.policyResults {
		if result.Scope != "" {
			scopes[result.Scope] = true
		}
	}
	return sortedKeys(scopes)
}

func (b *graphBuilder) complexityPackageQueryScopes() []string {
	scopes := map[string]bool{}
	for _, node := range b.nodes {
		if node.PackagePath != "" && (node.Kind == "function" || node.Kind == "method") {
			scopes[node.PackagePath] = true
		}
	}
	for _, metric := range b.metrics {
		if metric.Scope != "" && metric.Name == "cyclomatic_complexity" {
			scopes[metric.Scope] = true
		}
	}
	return sortedKeys(scopes)
}

func (b *graphBuilder) complexityDomainQueryScopes() []string {
	scopes := map[string]bool{}
	for _, edge := range b.edges {
		if edge.Kind != "imports" || !strings.HasPrefix(edge.From, "package:") {
			continue
		}
		scope := strings.TrimPrefix(edge.From, "package:")
		if scope != "" {
			scopes[scope] = true
		}
	}
	return sortedKeys(scopes)
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (b *graphBuilder) policyQuery(scope string) QueryResult {
	status := "pass"
	reason := "all_policy_results_passed"
	actual := TrustPackageLoaded
	var evidence []string
	for _, result := range b.policyResults {
		if result.Scope != scope {
			continue
		}
		evidence = append(evidence, result.ID)
		if core.TrustRank(result.TrustLevel) < core.TrustRank(actual) {
			actual = result.TrustLevel
		}
		switch result.Status {
		case "fail":
			status = "fail"
			reason = "policy_result_failed"
		case "unknown":
			if status != "fail" {
				status = "unknown"
				reason = "policy_result_unknown"
			}
		}
	}
	if len(evidence) == 0 {
		status = "unknown"
		reason = "no_policy_results_for_scope"
		actual = TrustSyntaxObserved
		evidence = []string{"policy_results_missing_for_scope"}
	}
	return QueryResult{
		ID:                 "query:policy:" + scope,
		Status:             status,
		Query:              "package dependency policy",
		Reason:             reason,
		Scope:              scope,
		RequiredTrustLevel: TrustSyntaxObserved,
		ActualTrustLevel:   actual,
		ProofStatus:        proofStatusForQuery(status),
		Evidence:           evidence,
	}
}

func (b *graphBuilder) complexityQuery(scope string, mode complexityScopeMode) QueryResult {
	status := "pass"
	reason := "all_complexity_metrics_passed"
	actual := TrustSyntaxObserved
	scopes := b.complexityScopes(scope, mode)
	var evidence []string
	metricSubjects := map[string]bool{}
	for _, metric := range b.metrics {
		if !scopes[metric.Scope] || metric.Name != "cyclomatic_complexity" {
			continue
		}
		evidence = append(evidence, metric.ID)
		metricSubjects[metric.Subject] = true
		if core.TrustRank(metric.TrustLevel) < core.TrustRank(actual) {
			actual = metric.TrustLevel
		}
		if metric.Status == "warning" {
			status = "warning"
			reason = "complexity_metric_warning"
		}
	}
	if b.hasComplexityRun() {
		incompleteEvidence := b.complexityIncompleteEvidence(scopes, metricSubjects)
		if mode == complexityScopeDomain {
			incompleteEvidence = append(incompleteEvidence, b.domainImportIncompleteEvidence(scope)...)
		}
		if len(incompleteEvidence) > 0 {
			sort.Strings(incompleteEvidence)
			status = "unknown"
			reason = "complexity_analysis_incomplete"
			evidence = append(evidence, incompleteEvidence...)
		}
	}
	if len(evidence) == 0 {
		status = "unknown"
		reason = "no_complexity_metrics_for_scope"
		evidence = []string{"complexity_metrics_missing_for_scope"}
	}
	return QueryResult{
		ID:                 "query:complexity:" + string(mode) + ":" + scope,
		Status:             status,
		Query:              "cyclomatic complexity " + string(mode) + " threshold",
		Reason:             reason,
		Scope:              scope,
		RequiredTrustLevel: TrustSyntaxObserved,
		ActualTrustLevel:   actual,
		ProofStatus:        proofStatusForQuery(status),
		Evidence:           evidence,
	}
}

func proofStatusForQuery(status string) string {
	switch status {
	case "unknown", "analysis_error":
		return status
	default:
		return "exists"
	}
}

func (b *graphBuilder) hasComplexityRun() bool {
	for _, run := range b.runs {
		if run.Stage == "complexity_metrics" {
			return true
		}
	}
	return false
}

func (b *graphBuilder) complexityIncompleteEvidence(scopes map[string]bool, metricSubjects map[string]bool) []string {
	var evidence []string
	scopedFiles := map[string]bool{}
	for _, node := range b.nodes {
		if !scopes[node.PackagePath] {
			continue
		}
		if node.SourceFile != "" {
			scopedFiles[node.SourceFile] = true
		}
		if (node.Kind == "function" || node.Kind == "method") && !metricSubjects[node.ID] {
			evidence = append(evidence, "missing_complexity_metric:"+node.ID)
		}
	}
	for _, diagnostic := range b.diagnostics {
		if diagnostic.Stage != "complexity_metrics" || diagnostic.Status != "analysis_error" {
			continue
		}
		if diagnostic.SourceFile != "" && scopedFiles[diagnostic.SourceFile] {
			evidence = append(evidence, "complexity_analysis_error:"+diagnostic.SourceFile)
		}
	}
	sort.Strings(evidence)
	return evidence
}

func (b *graphBuilder) complexityScopes(scope string, mode complexityScopeMode) map[string]bool {
	scopes := map[string]bool{scope: true}
	if mode == complexityScopePackage {
		return scopes
	}
	fromID := "package:" + scope
	for _, edge := range b.edges {
		if edge.Kind != "imports" || edge.From != fromID {
			continue
		}
		target, ok := strings.CutPrefix(edge.To, "package:")
		if ok {
			scopes[target] = true
		}
	}
	return scopes
}

func (b *graphBuilder) domainImportIncompleteEvidence(scope string) []string {
	fromID := "package:" + scope
	var evidence []string
	for _, edge := range b.edges {
		if edge.Kind != "imports" || edge.From != fromID {
			continue
		}
		if !edge.Complete || edge.Synthetic || strings.HasPrefix(edge.To, "placeholder:package:") {
			evidence = append(evidence, "incomplete_import_scope:"+edge.ID)
			continue
		}
		target, ok := strings.CutPrefix(edge.To, "package:")
		if !ok || target == "" || !b.hasNode(edge.To) {
			evidence = append(evidence, "unresolved_import_scope:"+edge.To)
		}
	}
	sort.Strings(evidence)
	return evidence
}

func (b *graphBuilder) simulateFreshnessChange() {
	file := filepath.Join(b.root, "order", "service.go")
	src, err := os.ReadFile(file)
	if err != nil {
		return
	}
	oldHash := srcutil.HashBytes(src)
	changed := append([]byte{}, src...)
	changed = append(changed, []byte("\n// simulated unsaved edit\n")...)
	sourceFile := b.rel(file)
	b.freshness = append(b.freshness, Freshness{FactID: "file:" + sourceFile, SourceFile: sourceFile, OldHash: oldHash, NewHash: srcutil.HashBytes(changed), Status: "superseded", Reason: "simulated_changed_content_requires_reindex", Extractor: core.ExtractorVersion, FactMetadata: b.metadataForCurrentRun()})
}
