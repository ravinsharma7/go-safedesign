package deppolicy

import (
	"encoding/json"
	"testing"

	"github.com/ravinsharma7/go-safedesign/internal/core"
	"github.com/ravinsharma7/go-safedesign/internal/pipeline"
)

func TestAnalyzerReportsFailUnknownAndAllowedImports(t *testing.T) {
	graph := core.Graph{
		Edges: []core.Edge{
			{
				ID:         "edge:imports:package:example.com/shop/order->package:example.com/shop/paymentclient",
				From:       "package:example.com/shop/order",
				To:         "package:example.com/shop/paymentclient",
				Kind:       "imports",
				TrustLevel: core.TrustSyntaxObserved,
				Complete:   true,
			},
			{
				ID:         "edge:imports:package:example.com/shop/order->package:example.com/payments/gateway",
				From:       "package:example.com/shop/order",
				To:         "package:example.com/payments/gateway",
				Kind:       "imports",
				TrustLevel: core.TrustSyntaxObserved,
				Complete:   true,
			},
			{
				ID:         "edge:imports:package:example.com/shop/order->placeholder:package:example.com/missing-notification/notify",
				From:       "package:example.com/shop/order",
				To:         "placeholder:package:example.com/missing-notification/notify",
				Kind:       "imports",
				TrustLevel: core.TrustSyntaxObserved,
				Synthetic:  true,
				Complete:   false,
			},
		},
	}
	policy := PackageImportPolicy{Rules: []Rule{{
		ID:      "order-package-imports",
		Package: "example.com/shop/order",
		Allow:   []string{"example.com/shop/paymentclient"},
	}}}
	configuration, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}

	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph, Configuration: configuration})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.PolicyResults) != 3 {
		t.Fatalf("policy results = %#v, want 3", result.PolicyResults)
	}
	if !hasPolicyResult(result.PolicyResults, "pass", "example.com/shop/paymentclient") {
		t.Fatalf("policy results = %#v, missing pass", result.PolicyResults)
	}
	if !hasPolicyResult(result.PolicyResults, "fail", "example.com/payments/gateway") {
		t.Fatalf("policy results = %#v, missing fail", result.PolicyResults)
	}
	if !hasPolicyResult(result.PolicyResults, "unknown", "example.com/missing-notification/notify") {
		t.Fatalf("policy results = %#v, missing unknown", result.PolicyResults)
	}
	diagnostics := result.Diagnostics
	if len(diagnostics) != 2 {
		t.Fatalf("diagnostics = %#v, want 2", diagnostics)
	}
	if diagnostics[0].Stage != string(pipeline.StagePolicyEvaluation) || diagnostics[0].AnalyzerID != ID {
		t.Fatalf("diagnostic provenance = %#v", diagnostics[0])
	}
	if !hasDiagnostic(diagnostics, "fail", "policy_violation: example.com/shop/order imports package outside allow list example.com/payments/gateway") {
		t.Fatalf("diagnostics = %#v, missing policy failure", diagnostics)
	}
	if !hasDiagnostic(diagnostics, "unknown", "policy_unknown: import target incomplete for example.com/missing-notification/notify") {
		t.Fatalf("diagnostics = %#v, missing unknown placeholder result", diagnostics)
	}
	if len(result.Warnings) != 2 {
		t.Fatalf("warnings = %#v, want fail and unknown warnings", result.Warnings)
	}
	failPolicy := policyResultWithSubject(result.PolicyResults, "example.com/payments/gateway")
	failEdgeID := "edge:imports:package:example.com/shop/order->package:example.com/payments/gateway"
	if failPolicy == nil || !hasWarning(result.Warnings, failPolicy.ID, failEdgeID, "policy_violation: example.com/shop/order imports package outside allow list example.com/payments/gateway") {
		t.Fatalf("warnings = %#v, missing fail warning with evidence", result.Warnings)
	}
	unknownPolicy := policyResultWithSubject(result.PolicyResults, "example.com/missing-notification/notify")
	unknownEdgeID := "edge:imports:package:example.com/shop/order->placeholder:package:example.com/missing-notification/notify"
	if unknownPolicy == nil || !hasWarning(result.Warnings, unknownPolicy.ID, unknownEdgeID, "policy_unknown: import target incomplete for example.com/missing-notification/notify") {
		t.Fatalf("warnings = %#v, missing unknown warning with evidence", result.Warnings)
	}
	if len(result.Edges) != 1 {
		t.Fatalf("edges = %#v, want one violation edge for failed import only", result.Edges)
	}
	if result.Edges[0].Kind != "violates" || result.Edges[0].From != "package:example.com/shop/order" || result.Edges[0].To != "package:example.com/payments/gateway" {
		t.Fatalf("violation edge = %#v", result.Edges[0])
	}
}

func TestRuleIDDefaultsToPackageWhenMissing(t *testing.T) {
	rule := normalizedRule(Rule{Package: "example.com/shop/order"})
	if rule.ID != "package_import:example.com/shop/order" {
		t.Fatalf("rule ID = %q", rule.ID)
	}
}

func TestRuleIDDefaultsToModuleWhenMissing(t *testing.T) {
	rule := normalizedRule(Rule{Module: "example.com/shop"})
	if rule.ID != "module_dependency:example.com/shop" {
		t.Fatalf("rule ID = %q", rule.ID)
	}
}

func TestAnalyzerReportsModuleAllowDenyAndUnknownDependencies(t *testing.T) {
	graph := core.Graph{
		Edges: []core.Edge{
			{
				ID:         "edge:depends_on:module:example.com/shop->module:example.com/payments",
				From:       "module:example.com/shop",
				To:         "module:example.com/payments",
				Kind:       "depends_on",
				TrustLevel: core.TrustSyntaxObserved,
				Complete:   true,
			},
			{
				ID:         "edge:depends_on:module:example.com/shop->module:example.com/blocked",
				From:       "module:example.com/shop",
				To:         "module:example.com/blocked",
				Kind:       "depends_on",
				TrustLevel: core.TrustSyntaxObserved,
				Complete:   true,
			},
			{
				ID:         "edge:depends_on:module:example.com/shop->placeholder:module:example.com/missing",
				From:       "module:example.com/shop",
				To:         "placeholder:module:example.com/missing",
				Kind:       "depends_on",
				TrustLevel: core.TrustSyntaxObserved,
				Synthetic:  true,
				Complete:   false,
			},
		},
	}
	policy := PackageImportPolicy{Rules: []Rule{{
		ID:           "shop-module-dependencies",
		Module:       "example.com/shop",
		AllowModules: []string{"example.com/payments"},
	}}}
	configuration, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}

	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph, Configuration: configuration})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.PolicyResults) != 3 {
		t.Fatalf("policy results = %#v, want 3", result.PolicyResults)
	}
	if !hasPolicyResultWithRule(result.PolicyResults, "shop-module-dependencies", "pass", "example.com/payments") {
		t.Fatalf("policy results = %#v, missing module pass", result.PolicyResults)
	}
	if !hasPolicyResultWithRule(result.PolicyResults, "shop-module-dependencies", "fail", "example.com/blocked") {
		t.Fatalf("policy results = %#v, missing module fail", result.PolicyResults)
	}
	if !hasPolicyResultWithRule(result.PolicyResults, "shop-module-dependencies", "unknown", "example.com/missing") {
		t.Fatalf("policy results = %#v, missing module unknown", result.PolicyResults)
	}
	if len(result.Warnings) != 2 {
		t.Fatalf("warnings = %#v, want module fail and unknown warnings", result.Warnings)
	}
	if len(result.Edges) != 1 || result.Edges[0].Kind != "violates" || result.Edges[0].From != "module:example.com/shop" || result.Edges[0].To != "module:example.com/blocked" {
		t.Fatalf("violation edges = %#v, want one failed module dependency violation", result.Edges)
	}
}

func TestAnalyzerReportsPackageLoadingDiagnosticsAsPolicyResults(t *testing.T) {
	graph := core.Graph{Diagnostics: []core.Diagnostic{
		{
			Source: "go/packages:example.com/shop/order",
			Reason: "no required module provides package example.com/missing",
			Status: "missing_dependency",
		},
		{
			Source: "go/packages:example.com/shop/cycle",
			Reason: "import cycle not allowed",
			Status: "import_cycle",
		},
	}}
	policy := PackageImportPolicy{Rules: []Rule{{ID: "unused", Package: "example.com/shop/order"}}}
	configuration, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}

	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph, Configuration: configuration})
	if err != nil {
		t.Fatal(err)
	}
	if !hasPolicyResultWithRule(result.PolicyResults, "package_loading:missing_dependency", "unknown", "go/packages:example.com/shop/order") {
		t.Fatalf("policy results = %#v, missing dependency classification", result.PolicyResults)
	}
	if !hasPolicyResultWithRule(result.PolicyResults, "package_loading:import_cycle", "fail", "go/packages:example.com/shop/cycle") {
		t.Fatalf("policy results = %#v, missing import cycle classification", result.PolicyResults)
	}
	if !hasDiagnostic(result.Diagnostics, "unknown", "policy_unknown: missing dependency: no required module provides package example.com/missing") {
		t.Fatalf("diagnostics = %#v, missing dependency diagnostic", result.Diagnostics)
	}
	if !hasDiagnostic(result.Diagnostics, "fail", "policy_violation: import cycle detected: import cycle not allowed") {
		t.Fatalf("diagnostics = %#v, missing import cycle diagnostic", result.Diagnostics)
	}
	if len(result.Warnings) != 2 {
		t.Fatalf("warnings = %#v, want package loading warnings", result.Warnings)
	}
	if len(result.Edges) != 0 {
		t.Fatalf("edges = %#v, package loading diagnostics should not emit violation edges without an evaluated source edge", result.Edges)
	}
}

func TestMetadataUsesPolicyEvaluationStage(t *testing.T) {
	metadata := Metadata()
	if metadata.ID != ID || metadata.Stage != pipeline.StagePolicyEvaluation {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func hasDiagnostic(diagnostics []core.Diagnostic, status, reason string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Status == status && diagnostic.Reason == reason {
			return true
		}
	}
	return false
}

func hasPolicyResult(results []core.PolicyResult, status, subject string) bool {
	return hasPolicyResultWithRule(results, "order-package-imports", status, subject)
}

func hasPolicyResultWithRule(results []core.PolicyResult, ruleID, status, subject string) bool {
	for _, result := range results {
		if result.Status == status && result.Subject == subject && result.RuleID == ruleID && len(result.Evidence) == 1 {
			return true
		}
	}
	return false
}

func policyResultWithSubject(results []core.PolicyResult, subject string) *core.PolicyResult {
	for i := range results {
		if results[i].Subject == subject {
			return &results[i]
		}
	}
	return nil
}

func hasWarning(warnings []core.Warning, policyResultID, edgeID, reason string) bool {
	for _, warning := range warnings {
		if warning.Kind != "policy_warning" || warning.Reason != reason || warning.AffectedEdgeID != edgeID {
			continue
		}
		if len(warning.Evidence) < 2 || warning.Evidence[0] != policyResultID || warning.Evidence[1] != edgeID {
			continue
		}
		return true
	}
	return false
}
