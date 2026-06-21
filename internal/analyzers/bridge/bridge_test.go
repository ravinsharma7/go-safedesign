package bridge

import (
	"reflect"
	"testing"

	"github.com/ravinsharma7/go-safedesign/internal/core"
	"github.com/ravinsharma7/go-safedesign/internal/pipeline"
)

func TestAnalyzerEmitsBridgeForCompletePackageEdgeBetweenCandidates(t *testing.T) {
	edge := importEdge("edge:imports:package:example.com/order->package:example.com/payment", "example.com/order", "example.com/payment")
	graph := core.Graph{
		Edges: []core.Edge{edge},
		Observations: []core.Observation{
			candidateObservation("obs:candidate:order", "example.com/order", core.TrustSyntaxObserved),
			candidateObservation("obs:candidate:payment", "example.com/payment", core.TrustSyntaxObserved),
		},
	}

	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Observations) != 1 {
		t.Fatalf("observations = %#v, want one bridge", result.Observations)
	}
	observation := result.Observations[0]
	if observation.Name != core.ObservationNameBridgeSymbol || observation.TargetID != edge.ID || observation.TargetKind != core.FactKindEdge {
		t.Fatalf("observation = %#v, want edge-targeted bridge symbol", observation)
	}
	if observation.Attributes["fromPackagePath"] != "example.com/order" || observation.Attributes["toPackagePath"] != "example.com/payment" || observation.Attributes["edgeKind"] != core.EdgeKindImports {
		t.Fatalf("attributes = %#v", observation.Attributes)
	}
	if !reflect.DeepEqual(observation.Evidence, []string{edge.ID, "obs:candidate:order", "obs:candidate:payment"}) {
		t.Fatalf("evidence = %#v, want sorted edge and candidate evidence", observation.Evidence)
	}
	if observation.Source != core.ObservationSourceInferred || observation.Freshness != core.FreshnessFresh {
		t.Fatalf("observation = %#v, want inferred fresh bridge", observation)
	}
}

func TestAnalyzerSkipsIncompletePlaceholderSamePackageAndMissingCandidateEdges(t *testing.T) {
	graph := core.Graph{
		Edges: []core.Edge{
			{ID: "edge:incomplete", Kind: core.EdgeKindImports, From: core.PackageID("example.com/order"), To: core.PackageID("example.com/payment"), TrustLevel: core.TrustSyntaxObserved, Complete: false},
			{ID: "edge:placeholder", Kind: core.EdgeKindImports, From: core.PackageID("example.com/order"), To: core.PlaceholderPackageID("example.com/missing"), TrustLevel: core.TrustSyntaxObserved, Complete: true},
			importEdge("edge:same", "example.com/order", "example.com/order"),
			importEdge("edge:no-candidate", "example.com/order", "example.com/notification"),
			{ID: "edge:contains", Kind: core.EdgeKindContains, From: core.PackageID("example.com/order"), To: core.PackageID("example.com/payment"), TrustLevel: core.TrustSyntaxObserved, Complete: true},
		},
		Observations: []core.Observation{
			candidateObservation("obs:candidate:order", "example.com/order", core.TrustSyntaxObserved),
			candidateObservation("obs:candidate:payment", "example.com/payment", core.TrustSyntaxObserved),
		},
	}

	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Observations) != 0 {
		t.Fatalf("observations = %#v, want skipped invalid bridge sources", result.Observations)
	}
}

func TestAnalyzerDeterministicOutputRegardlessOfInputOrder(t *testing.T) {
	edge := importEdge("edge:imports:package:example.com/order->package:example.com/payment", "example.com/order", "example.com/payment")
	first := core.Graph{
		Edges: []core.Edge{edge},
		Observations: []core.Observation{
			candidateObservation("obs:b", "example.com/payment", core.TrustSyntaxObserved),
			candidateObservation("obs:a", "example.com/order", core.TrustSyntaxObserved),
		},
	}
	second := core.Graph{
		Edges: []core.Edge{edge},
		Observations: []core.Observation{
			candidateObservation("obs:a", "example.com/order", core.TrustSyntaxObserved),
			candidateObservation("obs:b", "example.com/payment", core.TrustSyntaxObserved),
		},
	}

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
	if !reflect.DeepEqual(firstResult.Observations[0].Evidence, []string{edge.ID, "obs:a", "obs:b"}) {
		t.Fatalf("evidence = %#v", firstResult.Observations[0].Evidence)
	}
}

func TestAnalyzerUsesMinimumTrust(t *testing.T) {
	edge := importEdge("edge:imports:package:example.com/order->package:example.com/payment", "example.com/order", "example.com/payment")
	edge.TrustLevel = core.TrustTypeResolved
	graph := core.Graph{
		Edges: []core.Edge{edge},
		Observations: []core.Observation{
			candidateObservation("obs:type", "example.com/order", core.TrustTypeResolved),
			candidateObservation("obs:syntax", "example.com/payment", core.TrustSyntaxObserved),
		},
	}

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

func importEdge(id, fromPackage, toPackage string) core.Edge {
	return core.Edge{
		ID:         id,
		Kind:       core.EdgeKindImports,
		From:       core.PackageID(fromPackage),
		To:         core.PackageID(toPackage),
		TrustLevel: core.TrustSyntaxObserved,
		Complete:   true,
	}
}

func candidateObservation(id, pkg string, trust core.TrustLevel) core.Observation {
	return core.Observation{
		ID:         id,
		Kind:       core.FactKindObservation,
		Name:       core.ObservationNameLanguageZoneCandidate,
		TargetID:   core.PackageID(pkg),
		TargetKind: "node",
		Attributes: map[string]string{
			"packagePath": pkg,
		},
		Source:     core.ObservationSourceInferred,
		TrustLevel: trust,
	}
}
