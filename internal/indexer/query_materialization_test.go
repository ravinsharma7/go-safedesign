package indexer

import (
	"testing"

	"go-safedesign/internal/core"
	"go-safedesign/internal/pipeline"
)

func TestComplexityQueryUnknownWhenFunctionMetricMissing(t *testing.T) {
	b := &graphBuilder{
		nodes: map[string]Node{
			"function:example.com/shop/order.PlaceOrder": {
				ID:          "function:example.com/shop/order.PlaceOrder",
				Kind:        "function",
				PackagePath: "example.com/shop/order",
				SourceFile:  "workspace/shop/order/service.go",
			},
			"function:example.com/shop/order.ContinuePayment": {
				ID:          "function:example.com/shop/order.ContinuePayment",
				Kind:        "function",
				PackagePath: "example.com/shop/order",
				SourceFile:  "workspace/shop/order/fragment.go",
			},
		},
		metrics: []Metric{{
			ID:         "metric:cyclomatic_complexity:function:example.com/shop/order.PlaceOrder",
			Name:       "cyclomatic_complexity",
			Status:     "pass",
			Scope:      "example.com/shop/order",
			Subject:    "function:example.com/shop/order.PlaceOrder",
			TrustLevel: core.TrustSyntaxObserved,
		}},
		runs: []core.RunRecord{{Stage: string(pipeline.StageComplexityMetrics)}},
	}

	query := b.complexityQuery("example.com/shop/order", complexityScopePackage)
	if query.Status != "unknown" || query.Reason != "complexity_analysis_incomplete" {
		t.Fatalf("query = %#v, want incomplete unknown", query)
	}
	if !contains(query.Evidence, "missing_complexity_metric:function:example.com/shop/order.ContinuePayment") {
		t.Fatalf("evidence = %#v, missing subject completeness evidence", query.Evidence)
	}
}

func TestComplexityQueryUnknownWhenScopedFileHasAnalysisError(t *testing.T) {
	b := &graphBuilder{
		nodes: map[string]Node{
			"file:workspace/shop/order/service.go": {
				ID:          "file:workspace/shop/order/service.go",
				Kind:        "file",
				PackagePath: "example.com/shop/order",
				SourceFile:  "workspace/shop/order/service.go",
			},
			"function:example.com/shop/order.PlaceOrder": {
				ID:          "function:example.com/shop/order.PlaceOrder",
				Kind:        "function",
				PackagePath: "example.com/shop/order",
				SourceFile:  "workspace/shop/order/service.go",
			},
		},
		metrics: []Metric{{
			ID:         "metric:cyclomatic_complexity:function:example.com/shop/order.PlaceOrder",
			Name:       "cyclomatic_complexity",
			Status:     "pass",
			Scope:      "example.com/shop/order",
			Subject:    "function:example.com/shop/order.PlaceOrder",
			TrustLevel: core.TrustSyntaxObserved,
		}},
		diagnostics: []Diagnostic{{
			Stage:      string(pipeline.StageComplexityMetrics),
			Status:     "analysis_error",
			SourceFile: "workspace/shop/order/service.go",
		}},
		runs: []core.RunRecord{{Stage: string(pipeline.StageComplexityMetrics)}},
	}

	query := b.complexityQuery("example.com/shop/order", complexityScopePackage)
	if query.Status != "unknown" || query.Reason != "complexity_analysis_incomplete" {
		t.Fatalf("query = %#v, want incomplete unknown", query)
	}
	if !contains(query.Evidence, "complexity_analysis_error:workspace/shop/order/service.go") {
		t.Fatalf("evidence = %#v, missing analysis error evidence", query.Evidence)
	}
}

func TestComplexityQueryPackageScopeExcludesDirectImports(t *testing.T) {
	b := complexityScopeFixtureBuilder()

	query := b.complexityQuery("example.com/shop/order", complexityScopePackage)
	if query.ID != "query:complexity:package:example.com/shop/order" || query.Status != "pass" {
		t.Fatalf("query = %#v, want package pass", query)
	}
	if contains(query.Evidence, "metric:cyclomatic_complexity:function:example.com/shop/paymentclient.Await") {
		t.Fatalf("evidence = %#v, package query should not include imported package metric", query.Evidence)
	}
}

func TestComplexityQueryDomainScopeIncludesDirectImports(t *testing.T) {
	b := complexityScopeFixtureBuilder()

	query := b.complexityQuery("example.com/shop/order", complexityScopeDomain)
	if query.ID != "query:complexity:domain:example.com/shop/order" || query.Status != "warning" {
		t.Fatalf("query = %#v, want domain warning", query)
	}
	if !contains(query.Evidence, "metric:cyclomatic_complexity:function:example.com/shop/paymentclient.Await") {
		t.Fatalf("evidence = %#v, missing direct import metric", query.Evidence)
	}
}

func TestComplexityQueryDomainScopeUnknownForPlaceholderDirectImport(t *testing.T) {
	b := complexityScopeFixtureBuilder()
	edgeID := "edge:imports:package:example.com/shop/order->placeholder:package:example.com/missing/notify"
	b.edges[edgeID] = Edge{
		ID:        edgeID,
		Kind:      "imports",
		From:      "package:example.com/shop/order",
		To:        "placeholder:package:example.com/missing/notify",
		Synthetic: true,
		Complete:  false,
	}

	query := b.complexityQuery("example.com/shop/order", complexityScopeDomain)
	if query.Status != "unknown" || query.Reason != "complexity_analysis_incomplete" {
		t.Fatalf("query = %#v, want domain unknown for placeholder import", query)
	}
	if !contains(query.Evidence, "incomplete_import_scope:"+edgeID) {
		t.Fatalf("evidence = %#v, missing incomplete import evidence", query.Evidence)
	}
}

func TestComplexityQueryPackageScopeIgnoresPlaceholderDirectImport(t *testing.T) {
	b := complexityScopeFixtureBuilder()
	b.edges["edge:imports:package:example.com/shop/order->placeholder:package:example.com/missing/notify"] = Edge{
		ID:        "edge:imports:package:example.com/shop/order->placeholder:package:example.com/missing/notify",
		Kind:      "imports",
		From:      "package:example.com/shop/order",
		To:        "placeholder:package:example.com/missing/notify",
		Synthetic: true,
		Complete:  false,
	}

	query := b.complexityQuery("example.com/shop/order", complexityScopePackage)
	if query.Status != "pass" {
		t.Fatalf("query = %#v, want package scope to ignore placeholder direct import", query)
	}
}

func TestComplexityQueryDomainScopeUnknownForUnresolvedDirectImportScope(t *testing.T) {
	b := complexityScopeFixtureBuilder()
	edgeID := "edge:imports:package:example.com/shop/order->package:example.com/shop/missing-real-node"
	b.edges[edgeID] = Edge{
		ID:       edgeID,
		Kind:     "imports",
		From:     "package:example.com/shop/order",
		To:       "package:example.com/shop/missing-real-node",
		Complete: true,
	}

	query := b.complexityQuery("example.com/shop/order", complexityScopeDomain)
	if query.Status != "unknown" || query.Reason != "complexity_analysis_incomplete" {
		t.Fatalf("query = %#v, want domain unknown for unresolved import scope", query)
	}
	if !contains(query.Evidence, "unresolved_import_scope:package:example.com/shop/missing-real-node") {
		t.Fatalf("evidence = %#v, missing unresolved import evidence", query.Evidence)
	}
}

func complexityScopeFixtureBuilder() *graphBuilder {
	return &graphBuilder{
		nodes: map[string]Node{
			"package:example.com/shop/order": {
				ID:          "package:example.com/shop/order",
				Kind:        "package",
				PackagePath: "example.com/shop/order",
			},
			"package:example.com/shop/paymentclient": {
				ID:          "package:example.com/shop/paymentclient",
				Kind:        "package",
				PackagePath: "example.com/shop/paymentclient",
			},
			"function:example.com/shop/order.PlaceOrder": {
				ID:          "function:example.com/shop/order.PlaceOrder",
				Kind:        "function",
				PackagePath: "example.com/shop/order",
				SourceFile:  "workspace/shop/order/service.go",
			},
			"function:example.com/shop/paymentclient.Await": {
				ID:          "function:example.com/shop/paymentclient.Await",
				Kind:        "function",
				PackagePath: "example.com/shop/paymentclient",
				SourceFile:  "workspace/shop/paymentclient/runtime.go",
			},
		},
		edges: map[string]Edge{
			"edge:imports:package:example.com/shop/order->package:example.com/shop/paymentclient": {
				ID:       "edge:imports:package:example.com/shop/order->package:example.com/shop/paymentclient",
				Kind:     "imports",
				From:     "package:example.com/shop/order",
				To:       "package:example.com/shop/paymentclient",
				Complete: true,
			},
		},
		metrics: []Metric{
			{
				ID:         "metric:cyclomatic_complexity:function:example.com/shop/order.PlaceOrder",
				Name:       "cyclomatic_complexity",
				Status:     "pass",
				Scope:      "example.com/shop/order",
				Subject:    "function:example.com/shop/order.PlaceOrder",
				TrustLevel: core.TrustSyntaxObserved,
			},
			{
				ID:         "metric:cyclomatic_complexity:function:example.com/shop/paymentclient.Await",
				Name:       "cyclomatic_complexity",
				Status:     "warning",
				Scope:      "example.com/shop/paymentclient",
				Subject:    "function:example.com/shop/paymentclient.Await",
				TrustLevel: core.TrustSyntaxObserved,
			},
		},
		runs: []core.RunRecord{{Stage: string(pipeline.StageComplexityMetrics)}},
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
