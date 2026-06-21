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
