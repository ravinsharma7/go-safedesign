package vocab

import (
	"reflect"
	"testing"

	"go-safedesign/internal/core"
	"go-safedesign/internal/pipeline"
)

func TestSplitWordsHandlesIdentifierStyles(t *testing.T) {
	got := SplitWords("HTTPServer_UserID/payment-client/SCREAMING_SNAKE")
	want := []string{"http", "server", "user", "id", "payment", "client", "screaming", "snake"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("words = %#v, want %#v", got, want)
	}
}

func TestAnalyzerEmitsVocabularyObservations(t *testing.T) {
	graph := core.Graph{Nodes: []core.Node{
		{
			ID:          "function:example.com/shop/order.PlaceOrder",
			Kind:        "function",
			Name:        "PlaceOrder",
			TrustLevel:  core.TrustSyntaxObserved,
			Freshness:   "fresh",
			SourceFile:  "order/service.go",
			LineRange:   "10:1-12:2",
			PackagePath: "example.com/shop/order",
			ModulePath:  "example.com/shop",
		},
		{
			ID:          "field:example.com/shop/order.OrderRequest.UserID",
			Kind:        "field",
			Name:        "UserID",
			TrustLevel:  core.TrustSyntaxObserved,
			Freshness:   "fresh",
			SourceFile:  "order/service.go",
			LineRange:   "8:2-8:8",
			PackagePath: "example.com/shop/order",
			ModulePath:  "example.com/shop",
		},
		{
			ID:         "placeholder:package:example.com/missing",
			Kind:       "placeholder",
			Name:       "MissingPackage",
			Synthetic:  true,
			TrustLevel: core.TrustSyntaxObserved,
		},
	}}
	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph})
	if err != nil {
		t.Fatal(err)
	}
	for _, term := range []string{"place", "order", "user", "id"} {
		if !hasTerm(result.Observations, term) {
			t.Fatalf("observations = %#v, missing term %s", result.Observations, term)
		}
	}
	if hasTerm(result.Observations, "missing") || hasTerm(result.Observations, "package") {
		t.Fatalf("observations = %#v, placeholder terms should be skipped", result.Observations)
	}
	userID := observationForTerm(result.Observations, "user")
	if userID == nil || userID.Attributes["original"] != "UserID" || userID.SourceFile != "order/service.go" || userID.TargetID == "" {
		t.Fatalf("user observation = %#v, want spelling and source metadata", userID)
	}
}

func TestAnalyzerEmitsIncompleteDependencyObservation(t *testing.T) {
	edgeID := "edge:imports:package:example.com/shop/order->placeholder:package:example.com/missing/notify"
	graph := core.Graph{Edges: []core.Edge{{
		ID:         edgeID,
		From:       "package:example.com/shop/order",
		To:         "placeholder:package:example.com/missing/notify",
		Kind:       "imports",
		TrustLevel: core.TrustSyntaxObserved,
		Synthetic:  true,
		Complete:   false,
		Reason:     "import_target_not_parsed_or_loaded",
		SourceFile: "order/service.go",
		LineRange:  "3:1-3:44",
	}}}
	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Observations) != 1 {
		t.Fatalf("observations = %#v, want one incomplete dependency observation", result.Observations)
	}
	observation := result.Observations[0]
	if observation.Name != "vocabulary.incomplete_dependency" || observation.TargetID != edgeID || observation.Evidence[0] != edgeID {
		t.Fatalf("observation = %#v, want edge-backed incompleteness evidence", observation)
	}
}

func TestObservationIDsAreDeterministic(t *testing.T) {
	graph := core.Graph{Nodes: []core.Node{{
		ID:          "function:example.com/shop/order.PlaceOrder",
		Kind:        "function",
		Name:        "PlaceOrder",
		TrustLevel:  core.TrustSyntaxObserved,
		PackagePath: "example.com/shop/order",
	}}}
	first, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Observations) != len(second.Observations) {
		t.Fatalf("observation counts differ: %#v %#v", first.Observations, second.Observations)
	}
	for i := range first.Observations {
		if first.Observations[i].ID != second.Observations[i].ID {
			t.Fatalf("ids differ: %#v %#v", first.Observations, second.Observations)
		}
	}
}

func hasTerm(observations []core.Observation, term string) bool {
	return observationForTerm(observations, term) != nil
}

func observationForTerm(observations []core.Observation, term string) *core.Observation {
	for i := range observations {
		if observations[i].Name == "vocabulary.term" && observations[i].Value == term {
			return &observations[i]
		}
	}
	return nil
}
