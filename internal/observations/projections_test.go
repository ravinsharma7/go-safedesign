package observations

import (
	"reflect"
	"testing"

	"go-safedesign/internal/core"
)

func TestTermsByPackageCountsAndSortsVocabularyTerms(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		termObservation("obs:z", "payment", "PaymentID", "example.com/shop/order", core.NodeKindField),
		termObservation("obs:a", "order", "PlaceOrder", "example.com/shop/order", core.NodeKindFunction),
		termObservation("obs:b", "order", "OrderID", "example.com/shop/order", core.NodeKindField),
		termObservation("obs:c", "gateway", "Gateway", "example.com/payments/gateway", core.NodeKindPackage),
		{Name: core.ObservationNameVocabularyIncompleteDependency, Value: "placeholder:package:missing"},
		termObservation("obs:no-package", "ignored", "Ignored", "", core.NodeKindFunction),
	}}

	got := TermsByPackage(graph)
	orderTerms := got["example.com/shop/order"]
	if len(orderTerms) != 2 {
		t.Fatalf("order terms = %#v, want two terms", orderTerms)
	}
	if orderTerms[0].Term != "order" || orderTerms[0].Count != 2 {
		t.Fatalf("first term = %#v, want order count 2", orderTerms[0])
	}
	if orderTerms[1].Term != "payment" || orderTerms[1].Count != 1 {
		t.Fatalf("second term = %#v, want payment count 1", orderTerms[1])
	}
	if !reflect.DeepEqual(orderTerms[0].NodeKinds, []string{core.NodeKindField, core.NodeKindFunction}) {
		t.Fatalf("node kinds = %#v", orderTerms[0].NodeKinds)
	}
	if _, ok := got[""]; ok {
		t.Fatalf("empty package should be excluded: %#v", got)
	}
}

func TestTopTermsByPackageLimitsAfterSorting(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		termObservation("obs:1", "beta", "Beta", "example.com/app", core.NodeKindFunction),
		termObservation("obs:2", "alpha", "Alpha", "example.com/app", core.NodeKindFunction),
		termObservation("obs:3", "alpha", "AlphaAgain", "example.com/app", core.NodeKindField),
	}}

	got := TopTermsByPackage(graph, 1)["example.com/app"]
	if len(got) != 1 || got[0].Term != "alpha" || got[0].Count != 2 {
		t.Fatalf("top terms = %#v, want alpha only", got)
	}
}

func TestSpellingsByTermGroupsOriginalSpellings(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		termObservation("obs:1", "id", "UserID", "example.com/app", core.NodeKindField),
		termObservation("obs:2", "id", "UserId", "example.com/app", core.NodeKindField),
		termObservation("obs:3", "id", "UserID", "example.com/app", core.NodeKindFunction),
		termObservation("obs:4", "user", "UserID", "example.com/app", core.NodeKindField),
	}}

	got := SpellingsByTerm(graph)["id"]
	if len(got) != 2 {
		t.Fatalf("spellings = %#v, want two spellings", got)
	}
	if got[0].Original != "UserID" || got[0].Count != 2 {
		t.Fatalf("first spelling = %#v, want UserID count 2", got[0])
	}
	if got[1].Original != "UserId" || got[1].Count != 1 {
		t.Fatalf("second spelling = %#v, want UserId count 1", got[1])
	}
}

func TestObservationsByTargetGroupsAndSorts(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		{ID: "obs:z", TargetID: "function:example.com/app.Run"},
		{ID: "obs:a", TargetID: "function:example.com/app.Run"},
		{ID: "obs:ignored"},
	}}

	got := ObservationsByTarget(graph)["function:example.com/app.Run"]
	if len(got) != 2 || got[0].ID != "obs:a" || got[1].ID != "obs:z" {
		t.Fatalf("target observations = %#v, want sorted group", got)
	}
}

func TestIncompleteDependenciesByPackageGroupsSourcePackage(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		{
			ID:   "obs:z",
			Name: core.ObservationNameVocabularyIncompleteDependency,
			Attributes: map[string]string{
				"from": core.PackageID("example.com/shop/order"),
			},
		},
		{
			ID:   "obs:a",
			Name: core.ObservationNameVocabularyIncompleteDependency,
			Attributes: map[string]string{
				"from": core.PackageID("example.com/shop/order"),
			},
		},
		{
			ID:   "obs:module",
			Name: core.ObservationNameVocabularyIncompleteDependency,
			Attributes: map[string]string{
				"from": core.ModuleID("example.com/shop"),
			},
		},
	}}

	got := IncompleteDependenciesByPackage(graph)["example.com/shop/order"]
	if len(got) != 2 || got[0].ID != "obs:a" || got[1].ID != "obs:z" {
		t.Fatalf("incomplete dependencies = %#v, want package-only sorted group", got)
	}
}

func TestCooccurrencesByPackageCountsAndSorts(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		cooccurrenceObservation("obs:z", "example.com/shop/order", "order", "payment", "2"),
		cooccurrenceObservation("obs:a", "example.com/shop/order", "order", "payment", "3"),
		cooccurrenceObservation("obs:b", "example.com/shop/order", "customer", "order", "5"),
		cooccurrenceObservation("obs:c", "example.com/shop/payment", "gateway", "payment", "1"),
		termObservation("obs:term", "order", "Order", "example.com/shop/order", core.NodeKindFunction),
	}}

	got := CooccurrencesByPackage(graph)
	orderPairs := got["example.com/shop/order"]
	if len(orderPairs) != 2 {
		t.Fatalf("cooccurrences = %#v, want two order package pairs", orderPairs)
	}
	if orderPairs[0].TermA != "customer" || orderPairs[0].TermB != "order" || orderPairs[0].Count != 5 {
		t.Fatalf("first pair = %#v, want customer/order count 5", orderPairs[0])
	}
	if orderPairs[1].TermA != "order" || orderPairs[1].TermB != "payment" || orderPairs[1].Count != 5 {
		t.Fatalf("second pair = %#v, want order/payment count 5", orderPairs[1])
	}
	if !reflect.DeepEqual(orderPairs[1].ObservationIDs, []string{"obs:a", "obs:z"}) {
		t.Fatalf("observation ids = %#v, want sorted contributing ids", orderPairs[1].ObservationIDs)
	}
}

func TestTopCooccurrencesByPackageLimitsAfterSorting(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		cooccurrenceObservation("obs:1", "example.com/app", "beta", "gamma", "1"),
		cooccurrenceObservation("obs:2", "example.com/app", "alpha", "beta", "4"),
		cooccurrenceObservation("obs:3", "example.com/app", "alpha", "gamma", "2"),
	}}

	got := TopCooccurrencesByPackage(graph, 2)["example.com/app"]
	if len(got) != 2 || got[0].TermA != "alpha" || got[0].TermB != "beta" || got[1].TermA != "alpha" || got[1].TermB != "gamma" {
		t.Fatalf("top cooccurrences = %#v", got)
	}
}

func TestCooccurrencesByTermGroupsEitherSide(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		cooccurrenceObservation("obs:1", "example.com/shop/order", "customer", "order", "2"),
		cooccurrenceObservation("obs:2", "example.com/shop/payment", "order", "payment", "3"),
		cooccurrenceObservation("obs:3", "example.com/shop/order", "customer", "order", "1"),
	}}

	got := CooccurrencesByTerm(graph)["order"]
	if len(got) != 2 {
		t.Fatalf("order cooccurrences = %#v, want two pairs", got)
	}
	if got[0].TermA != "customer" || got[0].TermB != "order" || got[0].Count != 3 {
		t.Fatalf("first order pair = %#v, want customer/order count 3", got[0])
	}
	if got[1].TermA != "order" || got[1].TermB != "payment" || got[1].Count != 3 {
		t.Fatalf("second order pair = %#v, want order/payment count 3", got[1])
	}
	if !reflect.DeepEqual(got[0].Packages, []string{"example.com/shop/order"}) {
		t.Fatalf("packages = %#v", got[0].Packages)
	}
}

func TestCooccurrenceProjectionsIgnoreMalformedObservations(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		cooccurrenceObservation("obs:valid", "example.com/app", "alpha", "beta", "1"),
		cooccurrenceObservation("obs:missing-package", "", "alpha", "beta", "1"),
		cooccurrenceObservation("obs:missing-term", "example.com/app", "alpha", "", "1"),
		cooccurrenceObservation("obs:bad-count", "example.com/app", "alpha", "gamma", "bad"),
		cooccurrenceObservation("obs:zero-count", "example.com/app", "alpha", "delta", "0"),
		cooccurrenceObservation("obs:self-pair", "example.com/app", "alpha", "alpha", "1"),
		{Name: core.ObservationNameVocabularyIncompleteDependency, Attributes: map[string]string{"packagePath": "example.com/app", "termA": "alpha", "termB": "epsilon", "count": "9"}},
	}}

	got := CooccurrencesByPackage(graph)["example.com/app"]
	if len(got) != 1 || got[0].TermA != "alpha" || got[0].TermB != "beta" {
		t.Fatalf("cooccurrences = %#v, want only valid pair", got)
	}
	if byTerm := CooccurrencesByTerm(graph)["alpha"]; len(byTerm) != 1 {
		t.Fatalf("term cooccurrences = %#v, want only valid pair", byTerm)
	}
}

func TestEmptyGraphReturnsEmptyProjectionMaps(t *testing.T) {
	graph := core.Graph{}
	if len(TermsByPackage(graph)) != 0 {
		t.Fatal("TermsByPackage should be empty")
	}
	if len(SpellingsByTerm(graph)) != 0 {
		t.Fatal("SpellingsByTerm should be empty")
	}
	if len(ObservationsByTarget(graph)) != 0 {
		t.Fatal("ObservationsByTarget should be empty")
	}
	if len(IncompleteDependenciesByPackage(graph)) != 0 {
		t.Fatal("IncompleteDependenciesByPackage should be empty")
	}
	if len(TopTermsByPackage(graph, 3)) != 0 {
		t.Fatal("TopTermsByPackage should be empty")
	}
	if len(CooccurrencesByPackage(graph)) != 0 {
		t.Fatal("CooccurrencesByPackage should be empty")
	}
	if len(TopCooccurrencesByPackage(graph, 3)) != 0 {
		t.Fatal("TopCooccurrencesByPackage should be empty")
	}
	if len(CooccurrencesByTerm(graph)) != 0 {
		t.Fatal("CooccurrencesByTerm should be empty")
	}
}

func termObservation(id, term, original, pkg, nodeKind string) core.Observation {
	return core.Observation{
		ID:    id,
		Kind:  core.FactKindObservation,
		Name:  core.ObservationNameVocabularyTerm,
		Value: term,
		Attributes: map[string]string{
			"original":    original,
			"packagePath": pkg,
			"nodeKind":    nodeKind,
		},
	}
}

func cooccurrenceObservation(id, pkg, termA, termB, count string) core.Observation {
	return core.Observation{
		ID:   id,
		Kind: core.FactKindObservation,
		Name: core.ObservationNameVocabularyCooccurrence,
		Attributes: map[string]string{
			"packagePath": pkg,
			"termA":       termA,
			"termB":       termB,
			"count":       count,
		},
	}
}
