package langzone

import (
	"reflect"
	"testing"

	"go-safedesign/internal/core"
	"go-safedesign/internal/pipeline"
)

func TestAnalyzerEmitsCandidatePerPackage(t *testing.T) {
	graph := core.Graph{
		Nodes: []core.Node{{ID: core.PackageID("example.com/shop/order"), Kind: core.NodeKindPackage, PackagePath: "example.com/shop/order"}},
		Observations: []core.Observation{
			term("obs:order", "example.com/shop/order", "order", "function:order"),
			term("obs:payment", "example.com/shop/order", "payment", "function:order"),
			cooccurrence("obs:co", "example.com/shop/order", "order", "payment", "2"),
		},
	}

	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Observations) != 1 {
		t.Fatalf("observations = %#v, want one candidate", result.Observations)
	}
	observation := result.Observations[0]
	if observation.Name != core.ObservationNameLanguageZoneCandidate || observation.TargetID != core.PackageID("example.com/shop/order") || observation.TargetKind != "node" {
		t.Fatalf("candidate = %#v, want package-targeted language zone candidate", observation)
	}
	if observation.Attributes["packagePath"] != "example.com/shop/order" || observation.Attributes["terms"] != "order,payment" || observation.Attributes["termCount"] != "2" || observation.Attributes["cooccurrenceCount"] != "2" {
		t.Fatalf("attributes = %#v", observation.Attributes)
	}
	if !reflect.DeepEqual(observation.Evidence, []string{"obs:co", "obs:order", "obs:payment"}) {
		t.Fatalf("evidence = %#v, want sorted evidence", observation.Evidence)
	}
	if observation.Source != core.ObservationSourceInferred || observation.Freshness != core.FreshnessFresh {
		t.Fatalf("candidate = %#v, want inferred fresh observation", observation)
	}
}

func TestAnalyzerSkipsPackagesWithoutCooccurrencesOrEnoughTerms(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		term("obs:order", "example.com/shop/order", "order", "function:order"),
		term("obs:payment", "example.com/shop/order", "payment", "function:payment"),
		cooccurrence("obs:self", "example.com/shop/payment", "payment", "payment", "1"),
		cooccurrence("obs:one-term", "example.com/shop/customer", "customer", "", "1"),
	}}

	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Observations) != 0 {
		t.Fatalf("observations = %#v, want no candidates", result.Observations)
	}
}

func TestAnalyzerFiltersCandidateStopTermsWithoutFilteringRoleWords(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		term("obs:com", "example.com/shop/order", "com", "function:order"),
		term("obs:example", "example.com/shop/order", "example", "function:order"),
		term("obs:go", "example.com/shop/order", "go", "function:order"),
		term("obs:order", "example.com/shop/order", "order", "function:order"),
		term("obs:service", "example.com/shop/order", "service", "function:order"),
		cooccurrence("obs:filtered-pair", "example.com/shop/order", "com", "order", "5"),
		cooccurrence("obs:kept-pair", "example.com/shop/order", "order", "service", "2"),
	}}

	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Observations) != 1 {
		t.Fatalf("observations = %#v, want one candidate", result.Observations)
	}
	observation := result.Observations[0]
	if observation.Attributes["terms"] != "order,service" || observation.Attributes["termCount"] != "2" || observation.Attributes["cooccurrenceCount"] != "2" {
		t.Fatalf("attributes = %#v, want filtered candidate terms and retained role word", observation.Attributes)
	}
	if !reflect.DeepEqual(observation.Evidence, []string{"obs:kept-pair", "obs:order", "obs:service"}) {
		t.Fatalf("evidence = %#v, want only surviving term/pair evidence", observation.Evidence)
	}
}

func TestAnalyzerSkipsWhenFilteringLeavesFewerThanTwoTerms(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		term("obs:com", "example.com/shop/order", "com", "function:order"),
		term("obs:example", "example.com/shop/order", "example", "function:order"),
		term("obs:order", "example.com/shop/order", "order", "function:order"),
		cooccurrence("obs:filtered-pair", "example.com/shop/order", "com", "order", "5"),
	}}

	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Observations) != 0 {
		t.Fatalf("observations = %#v, want no candidate after filtering", result.Observations)
	}
}

func TestAnalyzerIgnoresIncompleteDependenciesAndPlaceholderTargets(t *testing.T) {
	graph := core.Graph{
		Nodes: []core.Node{{ID: core.PlaceholderPackageID("example.com/missing"), Kind: core.NodeKindPlaceholder, PackagePath: "example.com/missing", Synthetic: true}},
		Observations: []core.Observation{
			term("obs:missing-a", "example.com/missing", "missing", core.PlaceholderPackageID("example.com/missing")),
			term("obs:missing-b", "example.com/missing", "notify", core.PlaceholderPackageID("example.com/missing")),
			cooccurrence("obs:missing-co", "example.com/missing", "missing", "notify", "1"),
			{ID: "obs:incomplete", Name: core.ObservationNameVocabularyIncompleteDependency, Attributes: map[string]string{"packagePath": "example.com/missing"}},
		},
	}

	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Observations) != 0 {
		t.Fatalf("observations = %#v, want placeholder-backed evidence skipped", result.Observations)
	}
}

func TestAnalyzerLeavesTargetEmptyWhenPackageNodeMissing(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		term("obs:order", "example.com/shop/order", "order", "function:order"),
		term("obs:payment", "example.com/shop/order", "payment", "function:order"),
		cooccurrence("obs:co", "example.com/shop/order", "order", "payment", "1"),
	}}

	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Observations) != 1 || result.Observations[0].TargetID != "" || result.Observations[0].TargetKind != "" {
		t.Fatalf("observations = %#v, want candidate without target", result.Observations)
	}
}

func TestAnalyzerDeterministicIDsAndEvidenceRegardlessOfInputOrder(t *testing.T) {
	first := core.Graph{Observations: []core.Observation{
		cooccurrence("obs:z", "example.com/app", "alpha", "beta", "1"),
		term("obs:b", "example.com/app", "beta", "function:app"),
		term("obs:a", "example.com/app", "alpha", "function:app"),
	}}
	second := core.Graph{Observations: []core.Observation{
		term("obs:a", "example.com/app", "alpha", "function:app"),
		term("obs:b", "example.com/app", "beta", "function:app"),
		cooccurrence("obs:z", "example.com/app", "alpha", "beta", "1"),
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
		t.Fatalf("results = %#v %#v", firstResult.Observations, secondResult.Observations)
	}
	if firstResult.Observations[0].ID != secondResult.Observations[0].ID {
		t.Fatalf("ids differ: %s != %s", firstResult.Observations[0].ID, secondResult.Observations[0].ID)
	}
	if !reflect.DeepEqual(firstResult.Observations[0].Evidence, []string{"obs:a", "obs:b", "obs:z"}) {
		t.Fatalf("evidence = %#v", firstResult.Observations[0].Evidence)
	}
}

func TestAnalyzerUsesMinimumContributingTrust(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		termWithTrust("obs:type", "example.com/app", "type", "function:app", core.TrustTypeResolved),
		termWithTrust("obs:syntax", "example.com/app", "syntax", "function:app", core.TrustSyntaxObserved),
		cooccurrenceWithTrust("obs:co", "example.com/app", "syntax", "type", "1", core.TrustTypeResolved),
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
	if diagnostics := pipeline.ValidateAnalyzerMetadata(metadata); len(diagnostics) != 0 {
		t.Fatalf("metadata diagnostics = %#v", diagnostics)
	}
}

func term(id, pkg, value, target string) core.Observation {
	return termWithTrust(id, pkg, value, target, core.TrustSyntaxObserved)
}

func termWithTrust(id, pkg, value, target string, trust core.TrustLevel) core.Observation {
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
		Source:     core.ObservationSourceObserved,
		TrustLevel: trust,
	}
}

func cooccurrence(id, pkg, termA, termB, count string) core.Observation {
	return cooccurrenceWithTrust(id, pkg, termA, termB, count, core.TrustSyntaxObserved)
}

func cooccurrenceWithTrust(id, pkg, termA, termB, count string, trust core.TrustLevel) core.Observation {
	return core.Observation{
		ID:    id,
		Kind:  core.FactKindObservation,
		Name:  core.ObservationNameVocabularyCooccurrence,
		Value: termA + " " + termB,
		Attributes: map[string]string{
			"packagePath": pkg,
			"termA":       termA,
			"termB":       termB,
			"count":       count,
		},
		Source:     core.ObservationSourceInferred,
		TrustLevel: trust,
	}
}
