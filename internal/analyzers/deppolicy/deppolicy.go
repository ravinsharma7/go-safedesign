package deppolicy

import (
	"encoding/json"
	"sort"
	"strings"

	"go-safedesign/internal/core"
	"go-safedesign/internal/pipeline"
)

const (
	ID      = "dependency_policy"
	Version = "prototype-1"
)

type PackageImportPolicy struct {
	Rules []Rule `json:"rules"`
}

type Rule struct {
	ID           string   `json:"id,omitempty"`
	Package      string   `json:"package,omitempty"`
	Allow        []string `json:"allow,omitempty"`
	Deny         []string `json:"deny,omitempty"`
	Module       string   `json:"module,omitempty"`
	AllowModules []string `json:"allowModules,omitempty"`
	DenyModules  []string `json:"denyModules,omitempty"`
}

type Analyzer struct{}

func (Analyzer) Metadata() pipeline.AnalyzerMetadata {
	return pipeline.AnalyzerMetadata{
		ID:                    ID,
		Version:               Version,
		Stage:                 pipeline.StagePolicyEvaluation,
		InputFactKinds:        []string{"package", "imports"},
		MinimumRequiredTrust:  core.TrustSyntaxObserved,
		MaximumEmittedTrust:   core.TrustSyntaxObserved,
		EmittedFactKinds:      []string{"policy_result", "warning", "edge", "diagnostic"},
		ConfigurationSection:  "packageImportPolicy",
		FailureMode:           "partial",
		IncompleteInputPolicy: pipeline.IncompleteInputAllow,
	}
}

func Metadata() pipeline.AnalyzerMetadata {
	return Analyzer{}.Metadata()
}

func (a Analyzer) Run(context pipeline.GraphContext) (pipeline.AnalyzerResult, error) {
	if len(context.Configuration) == 0 {
		return pipeline.AnalyzerResult{}, nil
	}
	var policy PackageImportPolicy
	if err := json.Unmarshal(context.Configuration, &policy); err != nil {
		return pipeline.AnalyzerResult{}, err
	}
	results, diagnostics, warnings, edges := a.evaluate(context.Graph, policy, context.ConfigurationHash)
	return pipeline.AnalyzerResult{PolicyResults: results, Diagnostics: diagnostics, Warnings: warnings, Edges: edges}, nil
}

func (Analyzer) evaluate(graph core.Graph, policy PackageImportPolicy, configHash string) ([]core.PolicyResult, []core.Diagnostic, []core.Warning, []core.Edge) {
	packageRules := map[string]Rule{}
	moduleRules := map[string]Rule{}
	for _, rule := range policy.Rules {
		rule = normalizedRule(rule)
		if rule.Package != "" {
			packageRules[rule.Package] = rule
		}
		if rule.Module != "" {
			moduleRules[rule.Module] = rule
		}
	}

	var results []core.PolicyResult
	var diagnostics []core.Diagnostic
	var warnings []core.Warning
	var edges []core.Edge
	for _, edge := range graph.Edges {
		switch edge.Kind {
		case "imports":
			result, ok := packageImportResult(edge, packageRules, configHash)
			if !ok {
				continue
			}
			results = append(results, result)
			if result.Status != "pass" {
				diagnostics = append(diagnostics, diagnosticFor(result, edge.ID))
				warnings = append(warnings, warningFor(result, edge.ID))
			}
			if violation, ok := violationEdgeFor(result, edge); ok {
				edges = append(edges, violation)
			}
		case "depends_on":
			result, ok := moduleDependencyResult(edge, moduleRules, configHash)
			if !ok {
				continue
			}
			results = append(results, result)
			if result.Status != "pass" {
				diagnostics = append(diagnostics, diagnosticFor(result, edge.ID))
				warnings = append(warnings, warningFor(result, edge.ID))
			}
			if violation, ok := violationEdgeFor(result, edge); ok {
				edges = append(edges, violation)
			}
		}
	}
	for _, diagnostic := range graph.Diagnostics {
		result, ok := packageLoadingPolicyResult(diagnostic, configHash)
		if !ok {
			continue
		}
		results = append(results, result)
		if result.Status != "pass" {
			diagnostics = append(diagnostics, diagnosticFor(result, ""))
			warnings = append(warnings, warningFor(result, ""))
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].ID < results[j].ID
	})
	sort.Slice(diagnostics, func(i, j int) bool {
		if diagnostics[i].Source == diagnostics[j].Source {
			return diagnostics[i].Reason < diagnostics[j].Reason
		}
		return diagnostics[i].Source < diagnostics[j].Source
	})
	sort.Slice(warnings, func(i, j int) bool {
		return warnings[i].ID < warnings[j].ID
	})
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].ID < edges[j].ID
	})
	return results, diagnostics, warnings, edges
}

func normalizedRule(rule Rule) Rule {
	if rule.ID == "" {
		if rule.Package != "" {
			rule.ID = "package_import:" + rule.Package
		} else {
			rule.ID = "module_dependency:" + rule.Module
		}
	}
	sort.Strings(rule.Allow)
	sort.Strings(rule.Deny)
	sort.Strings(rule.AllowModules)
	sort.Strings(rule.DenyModules)
	return rule
}

func packageImportResult(edge core.Edge, rules map[string]Rule, configHash string) (core.PolicyResult, bool) {
	fromPkg, ok := strings.CutPrefix(edge.From, "package:")
	if !ok {
		return core.PolicyResult{}, false
	}
	rule, ok := rules[fromPkg]
	if !ok {
		return core.PolicyResult{}, false
	}
	targetPkg := packageFromImportTarget(edge.To)
	if targetPkg == "" {
		return core.PolicyResult{}, false
	}
	result := basePolicyResult(rule.ID, fromPkg, targetPkg, []string{edge.ID}, edge.TrustLevel, configHash)
	result.SourceFile = edge.SourceFile
	result.LineRange = edge.LineRange
	switch {
	case !edge.Complete || edge.Synthetic || strings.HasPrefix(edge.To, "placeholder:"):
		result.Status = "unknown"
		result.Reason = "import target incomplete for " + targetPkg
	case contains(rule.Deny, targetPkg):
		result.Status = "fail"
		result.Reason = fromPkg + " imports denied package " + targetPkg
	case len(rule.Allow) > 0 && !contains(rule.Allow, targetPkg):
		result.Status = "fail"
		result.Reason = fromPkg + " imports package outside allow list " + targetPkg
	default:
		result.Status = "pass"
		result.Reason = fromPkg + " import allowed for " + targetPkg
	}
	result.ID = "policy_result:" + rule.ID + ":" + edge.ID
	return result, true
}

func moduleDependencyResult(edge core.Edge, rules map[string]Rule, configHash string) (core.PolicyResult, bool) {
	fromModule, ok := strings.CutPrefix(edge.From, "module:")
	if !ok {
		return core.PolicyResult{}, false
	}
	rule, ok := rules[fromModule]
	if !ok {
		return core.PolicyResult{}, false
	}
	targetModule := moduleFromDependencyTarget(edge.To)
	if targetModule == "" {
		return core.PolicyResult{}, false
	}
	result := basePolicyResult(rule.ID, fromModule, targetModule, []string{edge.ID}, edge.TrustLevel, configHash)
	result.SourceFile = edge.SourceFile
	result.LineRange = edge.LineRange
	switch {
	case !edge.Complete || edge.Synthetic || strings.HasPrefix(edge.To, "placeholder:"):
		result.Status = "unknown"
		result.Reason = "module dependency target incomplete for " + targetModule
	case contains(rule.DenyModules, targetModule):
		result.Status = "fail"
		result.Reason = fromModule + " depends on denied module " + targetModule
	case len(rule.AllowModules) > 0 && !contains(rule.AllowModules, targetModule):
		result.Status = "fail"
		result.Reason = fromModule + " depends on module outside allow list " + targetModule
	default:
		result.Status = "pass"
		result.Reason = fromModule + " module dependency allowed for " + targetModule
	}
	result.ID = "policy_result:" + rule.ID + ":" + edge.ID
	return result, true
}

func packageLoadingPolicyResult(diagnostic core.Diagnostic, configHash string) (core.PolicyResult, bool) {
	switch diagnostic.Status {
	case "missing_dependency", "import_cycle":
	default:
		return core.PolicyResult{}, false
	}
	status := "unknown"
	reason := "missing dependency: " + diagnostic.Reason
	if diagnostic.Status == "import_cycle" {
		status = "fail"
		reason = "import cycle detected: " + diagnostic.Reason
	}
	scope := strings.TrimPrefix(diagnostic.Source, "go/packages:")
	evidence := diagnostic.Source + ":" + diagnostic.Reason
	result := basePolicyResult("package_loading:"+diagnostic.Status, scope, diagnostic.Source, []string{evidence}, core.TrustSyntaxObserved, configHash)
	result.ID = "policy_result:package_loading:" + diagnostic.Status + ":" + core.HashBytes([]byte(evidence))
	result.Status = status
	result.Reason = reason
	result.SourceFile = diagnostic.SourceFile
	result.LineRange = diagnostic.LineRange
	return result, true
}

func basePolicyResult(ruleID, scope, subject string, evidence []string, trust core.TrustLevel, configHash string) core.PolicyResult {
	return core.PolicyResult{
		Kind:              "policy_result",
		RuleID:            ruleID,
		AnalyzerID:        ID,
		Stage:             string(pipeline.StagePolicyEvaluation),
		Scope:             scope,
		Subject:           subject,
		Evidence:          evidence,
		TrustLevel:        trust,
		ConfigurationHash: configHash,
	}
}

func diagnosticFor(result core.PolicyResult, edgeID string) core.Diagnostic {
	diagnostic := core.Diagnostic{
		Source:         "policy:" + result.Scope,
		Stage:          result.Stage,
		AnalyzerID:     result.AnalyzerID,
		Status:         result.Status,
		PolicyResultID: result.ID,
		EdgeID:         edgeID,
		SourceFile:     result.SourceFile,
		LineRange:      result.LineRange,
		TrustLevel:     result.TrustLevel,
	}
	switch result.Status {
	case "unknown":
		diagnostic.Level = "warning"
		diagnostic.Reason = "policy_unknown: " + result.Reason
	default:
		diagnostic.Level = "error"
		diagnostic.Reason = "policy_violation: " + result.Reason
	}
	return diagnostic
}

func warningFor(result core.PolicyResult, edgeID string) core.Warning {
	reasonPrefix := "policy_violation: "
	action := "Update the code dependency or adjust the configured policy rule."
	if result.Status == "unknown" {
		reasonPrefix = "policy_unknown: "
		action = "Resolve incomplete analysis evidence before treating this policy result as pass or fail."
	}
	evidence := []string{result.ID}
	if edgeID != "" {
		evidence = append(evidence, edgeID)
	}
	return core.Warning{
		ID:                  "warning:" + result.ID,
		Kind:                "policy_warning",
		Reason:              reasonPrefix + result.Reason,
		SuggestedNextAction: action,
		AffectedEdgeID:      edgeID,
		Evidence:            evidence,
		TrustLevel:          result.TrustLevel,
		SourceFile:          result.SourceFile,
		LineRange:           result.LineRange,
		Freshness:           "fresh",
	}
}

func violationEdgeFor(result core.PolicyResult, evaluated core.Edge) (core.Edge, bool) {
	if result.Status != "fail" || evaluated.ID == "" {
		return core.Edge{}, false
	}
	from := policyScopeNodeID(result, evaluated)
	if from == "" || evaluated.To == "" {
		return core.Edge{}, false
	}
	return core.Edge{
		ID:         "edge:violates:" + result.ID,
		From:       from,
		To:         evaluated.To,
		Kind:       "violates",
		TrustLevel: result.TrustLevel,
		Complete:   evaluated.Complete,
		Reason:     result.Reason,
		SourceFile: result.SourceFile,
		LineRange:  result.LineRange,
	}, true
}

func policyScopeNodeID(result core.PolicyResult, evaluated core.Edge) string {
	if strings.HasPrefix(evaluated.From, "package:") || strings.HasPrefix(evaluated.From, "module:") {
		return evaluated.From
	}
	if strings.HasPrefix(result.RuleID, "module_dependency:") {
		return "module:" + result.Scope
	}
	return "package:" + result.Scope
}

func packageFromImportTarget(id string) string {
	for _, prefix := range []string{"package:", "placeholder:package:"} {
		if value, ok := strings.CutPrefix(id, prefix); ok {
			return value
		}
	}
	return ""
}

func moduleFromDependencyTarget(id string) string {
	for _, prefix := range []string{"module:", "placeholder:module:"} {
		if value, ok := strings.CutPrefix(id, prefix); ok {
			return value
		}
	}
	return ""
}

func contains(values []string, want string) bool {
	i := sort.SearchStrings(values, want)
	return i < len(values) && values[i] == want
}
