package ublang

import (
	"reflect"
	"testing"

	"go-safedesign/internal/core"
	"go-safedesign/internal/pipeline"
)

func TestSplitWordsHandlesCommonIdentifierStyles(t *testing.T) {
	got := SplitWords("HTTPGateway_OrderID/payment-client")
	want := []string{"http", "gateway", "order", "id", "payment", "client"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("words = %#v, want %#v", got, want)
	}
}

func TestAnalyzerEmitsContextTermsWarningsAndAlignment(t *testing.T) {
	graph := core.Graph{Nodes: []core.Node{
		{ID: "package:example.com/shop/order", Kind: "package", Name: "example.com/shop/order", PackagePath: "example.com/shop/order", TrustLevel: core.TrustSyntaxObserved},
		{ID: "function:example.com/shop/order.PlaceOrder", Kind: "function", Name: "PlaceOrder", PackagePath: "example.com/shop/order", TrustLevel: core.TrustSyntaxObserved},
		{ID: "struct:example.com/shop/order.OrderRequest", Kind: "struct", Name: "OrderRequest", PackagePath: "example.com/shop/order", TrustLevel: core.TrustSyntaxObserved},
		{ID: "field:example.com/shop/order.OrderRequest.InvoiceCode", Kind: "field", Name: "InvoiceCode", PackagePath: "example.com/shop/order", TrustLevel: core.TrustSyntaxObserved},
		{ID: "field:example.com/shop/order.OrderRequest.LegacyFlag", Kind: "field", Name: "LegacyFlag", PackagePath: "example.com/shop/order", TrustLevel: core.TrustSyntaxObserved},
	}}
	result, err := Analyzer{}.Run(pipeline.GraphContext{
		Graph: graph,
		Configuration: []byte(`{
			"contexts":[{
				"id":"ordering",
				"packagePrefixes":["example.com/shop/order"],
				"terms":["order","place","request","code"],
				"synonyms":{"purchase":"order"},
				"discouragedTerms":["invoice"]
			}],
			"ignoredTerms":["example","com","shop","flag"]
		}`),
		ConfigurationHash: "config-hash",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !hasLabel(result.Labels, "ddd.context", "ordering", "package:example.com/shop/order") {
		t.Fatalf("labels = %#v, missing ddd context label", result.Labels)
	}
	for _, term := range []string{"order", "place", "request", "code"} {
		if !hasLabel(result.Labels, "ul.term", term, "package:example.com/shop/order") {
			t.Fatalf("labels = %#v, missing term %s", result.Labels, term)
		}
	}
	if !hasWarning(result.Warnings, "ul_discouraged_term", "discouraged ubiquitous language term invoice") {
		t.Fatalf("warnings = %#v, missing discouraged term warning", result.Warnings)
	}
	if !hasWarning(result.Warnings, "ul_unknown_term", "unknown ubiquitous language term legacy") {
		t.Fatalf("warnings = %#v, missing unknown term warning", result.Warnings)
	}
	metric := metricNamed(result.Metrics, AlignmentMetricName)
	if metric == nil || metric.Subject != "package:example.com/shop/order" || metric.Value == 0 || metric.ConfigurationHash != "config-hash" {
		t.Fatalf("metrics = %#v, missing alignment metric", result.Metrics)
	}
}

func TestAnalyzerUsesLongestPackagePrefix(t *testing.T) {
	graph := core.Graph{Nodes: []core.Node{
		{ID: "package:example.com/shop/order", Kind: "package", Name: "example.com/shop/order", PackagePath: "example.com/shop/order", TrustLevel: core.TrustSyntaxObserved},
	}}
	result, err := Analyzer{}.Run(pipeline.GraphContext{
		Graph: graph,
		Configuration: []byte(`{"contexts":[
			{"id":"shop","packagePrefixes":["example.com/shop"],"terms":["shop"]},
			{"id":"ordering","packagePrefixes":["example.com/shop/order"],"terms":["order"]}
		]}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !hasLabel(result.Labels, "ddd.context", "ordering", "package:example.com/shop/order") {
		t.Fatalf("labels = %#v, want longest prefix context", result.Labels)
	}
}

func TestAnalyzerNoConfigEmitsNothing(t *testing.T) {
	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: core.Graph{Nodes: []core.Node{{ID: "package:p", Kind: "package", PackagePath: "p"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Labels) != 0 || len(result.Metrics) != 0 || len(result.Warnings) != 0 {
		t.Fatalf("result = %#v, want no facts without config", result)
	}
}

func TestMetadataUsesDDDClassificationStage(t *testing.T) {
	metadata := Metadata()
	if metadata.ID != ID || metadata.Stage != pipeline.StageDDDClassification {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func hasLabel(labels []core.Label, name, value, target string) bool {
	for _, label := range labels {
		if label.Name == name && label.Value == value && label.TargetID == target {
			return true
		}
	}
	return false
}

func hasWarning(warnings []core.Warning, kind, reason string) bool {
	for _, warning := range warnings {
		if warning.Kind == kind && warning.Reason == reason && len(warning.Evidence) > 0 {
			return true
		}
	}
	return false
}

func metricNamed(metrics []core.Metric, name string) *core.Metric {
	for i := range metrics {
		if metrics[i].Name == name {
			return &metrics[i]
		}
	}
	return nil
}
