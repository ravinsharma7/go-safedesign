package vocabco

import (
	"reflect"
	"testing"

	"github.com/ravinsharma7/go-safedesign/internal/core"
	"github.com/ravinsharma7/go-safedesign/internal/pipeline"
)

func TestAnalyzerEmitsPairCountPerPackageForSameTargetTerms(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		term("obs:order", "target:one", "example.com/shop/order", "order"),
		term("obs:payment", "target:one", "example.com/shop/order", "payment"),
		term("obs:second-order", "target:two", "example.com/shop/order", "order"),
		term("obs:second-payment", "target:two", "example.com/shop/order", "payment"),
	}}
	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Observations) != 1 {
		t.Fatalf("observations = %#v, want one cooccurrence", result.Observations)
	}
	observation := result.Observations[0]
	if observation.Name != core.ObservationNameVocabularyCooccurrence || observation.Attributes["termA"] != "order" || observation.Attributes["termB"] != "payment" || observation.Attributes["count"] != "2" {
		t.Fatalf("observation = %#v, want order/payment count 2", observation)
	}
	if !reflect.DeepEqual(observation.Evidence, []string{"obs:order", "obs:payment", "obs:second-order", "obs:second-payment"}) {
		t.Fatalf("evidence = %#v, want sorted contributing ids", observation.Evidence)
	}
}

func TestAnalyzerDeterministicIDsRegardlessOfInputOrder(t *testing.T) {
	first := core.Graph{Observations: []core.Observation{
		term("obs:b", "target:one", "example.com/app", "beta"),
		term("obs:a", "target:one", "example.com/app", "alpha"),
	}}
	second := core.Graph{Observations: []core.Observation{
		term("obs:a", "target:one", "example.com/app", "alpha"),
		term("obs:b", "target:one", "example.com/app", "beta"),
	}}
	firstResult, err := Analyzer{}.Run(pipeline.GraphContext{Graph: first})
	if err != nil {
		t.Fatal(err)
	}
	secondResult, err := Analyzer{}.Run(pipeline.GraphContext{Graph: second})
	if err != nil {
		t.Fatal(err)
	}
	if len(firstResult.Observations) != 1 || len(secondResult.Observations) != 1 {
		t.Fatalf("results = %#v %#v, want one observation each", firstResult.Observations, secondResult.Observations)
	}
	if firstResult.Observations[0].ID != secondResult.Observations[0].ID {
		t.Fatalf("ids differ: %s != %s", firstResult.Observations[0].ID, secondResult.Observations[0].ID)
	}
}

func TestAnalyzerIgnoresNonTermsSelfPairsMissingPackageAndCrossTargetTerms(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		term("obs:order-a", "target:one", "example.com/shop/order", "order"),
		term("obs:order-b", "target:one", "example.com/shop/order", "order"),
		term("obs:payment-other-target", "target:two", "example.com/shop/order", "payment"),
		term("obs:missing-package", "target:one", "", "payment"),
		{ID: "obs:incomplete", Name: core.ObservationNameVocabularyIncompleteDependency, TargetID: "target:one", Value: "payment"},
	}}
	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Observations) != 0 {
		t.Fatalf("observations = %#v, want none", result.Observations)
	}
}

func TestAnalyzerUsesMinimumTrust(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		termWithTrust("obs:type", "target:one", "example.com/app", "type", core.TrustTypeResolved),
		termWithTrust("obs:syntax", "target:one", "example.com/app", "syntax", core.TrustSyntaxObserved),
	}}
	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Observations) != 1 || result.Observations[0].TrustLevel != core.TrustSyntaxObserved {
		t.Fatalf("observations = %#v, want syntax observed trust", result.Observations)
	}
}

func TestMetadata(t *testing.T) {
	metadata := Metadata()
	if metadata.ID != ID || metadata.Stage != pipeline.StageDDDClassification || metadata.IncompleteInputPolicy != pipeline.IncompleteInputAllow {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func term(id, target, pkg, value string) core.Observation {
	return termWithTrust(id, target, pkg, value, core.TrustSyntaxObserved)
}

func termWithTrust(id, target, pkg, value string, trust core.TrustLevel) core.Observation {
	return core.Observation{
		ID:         id,
		Kind:       core.FactKindObservation,
		Name:       core.ObservationNameVocabularyTerm,
		Value:      value,
		TargetID:   target,
		TargetKind: "node",
		Attributes: map[string]string{
			"packagePath": pkg,
		},
		TrustLevel: trust,
	}
}
