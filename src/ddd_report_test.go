package main

import (
	"testing"

	"go-safedesign/internal/core"
)

func TestDDDReportIncludesOnlyDDDEvidenceObservations(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		candidateObservationForReport("obs:candidate:payment", "example.com/payment", "payment,checkout", "2", "3"),
		bridgeObservationForReport("obs:bridge:order-payment", "example.com/order", "example.com/payment"),
		incompleteObservationForReport("obs:incomplete:notify", "package:example.com/order", "placeholder:package:example.com/notify"),
		{ID: "obs:term", Name: core.ObservationNameVocabularyTerm, Value: "payment"},
		{ID: "obs:cooccurrence", Name: core.ObservationNameVocabularyCooccurrence, Value: "payment checkout"},
	}}

	report := buildDDDReport(graph, compactReportOptions{Limit: 50})
	if len(report.Candidates) != 1 || len(report.Bridges) != 1 || len(report.IncompleteEvidence) != 1 {
		t.Fatalf("report = %#v, want only candidate, bridge, and incomplete evidence sections", report)
	}
	if report.Summary.Candidates != 1 || report.Summary.Bridges != 1 || report.Summary.IncompleteEvidence != 1 || report.Summary.Truncated {
		t.Fatalf("summary = %#v", report.Summary)
	}
}

func TestDDDReportParsesCandidateAttributes(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		candidateObservationForReport("obs:candidate", "example.com/payment", " payment, checkout ,", "2", "7"),
	}}

	report := buildDDDReport(graph, compactReportOptions{Limit: 50})
	if len(report.Candidates) != 1 {
		t.Fatalf("candidates = %#v", report.Candidates)
	}
	candidate := report.Candidates[0]
	if candidate.PackagePath != "example.com/payment" || candidate.TargetID != core.PackageID("example.com/payment") {
		t.Fatalf("candidate = %#v, package/target not parsed", candidate)
	}
	if !same(candidate.Terms, []string{"payment", "checkout"}) || candidate.TermCount != 2 || candidate.CooccurrenceCount != 7 {
		t.Fatalf("candidate = %#v, terms/counts not parsed", candidate)
	}
	if candidate.EvidenceCount != 2 || candidate.TrustLevel != core.TrustSyntaxObserved || candidate.ObservationID != "obs:candidate" {
		t.Fatalf("candidate = %#v, metadata not parsed", candidate)
	}
}

func TestDDDReportParsesBridgeAttributes(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		bridgeObservationForReport("obs:bridge", "example.com/order", "example.com/payment"),
	}}

	report := buildDDDReport(graph, compactReportOptions{Limit: 50})
	if len(report.Bridges) != 1 {
		t.Fatalf("bridges = %#v", report.Bridges)
	}
	bridge := report.Bridges[0]
	if bridge.FromPackagePath != "example.com/order" || bridge.ToPackagePath != "example.com/payment" || bridge.EdgeKind != core.EdgeKindImports {
		t.Fatalf("bridge = %#v, attributes not parsed", bridge)
	}
	if bridge.TargetID != core.EdgeID(core.EdgeKindImports, core.PackageID("example.com/order"), core.PackageID("example.com/payment")) || bridge.EvidenceCount != 3 || bridge.ObservationID != "obs:bridge" {
		t.Fatalf("bridge = %#v, metadata not parsed", bridge)
	}
}

func TestDDDReportParsesIncompleteDependencyEvidence(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		incompleteObservationForReport("obs:incomplete", "package:example.com/order", "placeholder:package:example.com/notify"),
	}}

	report := buildDDDReport(graph, compactReportOptions{Limit: 50})
	if len(report.IncompleteEvidence) != 1 {
		t.Fatalf("incomplete evidence = %#v", report.IncompleteEvidence)
	}
	incomplete := report.IncompleteEvidence[0]
	if incomplete.From != "package:example.com/order" || incomplete.To != "placeholder:package:example.com/notify" || incomplete.EdgeKind != core.EdgeKindImports || incomplete.Reason != "import_target_not_parsed_or_loaded" {
		t.Fatalf("incomplete = %#v, attributes not parsed", incomplete)
	}
	if incomplete.TargetID == "" || incomplete.EvidenceCount != 1 || incomplete.ObservationID != "obs:incomplete" {
		t.Fatalf("incomplete = %#v, metadata not parsed", incomplete)
	}
}

func TestDDDReportSortsDeterministically(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		candidateObservationForReport("obs:candidate:z", "example.com/z", "z,alpha", "2", "1"),
		candidateObservationForReport("obs:candidate:a", "example.com/a", "a,beta", "2", "1"),
		bridgeObservationForReport("obs:bridge:z-a", "example.com/z", "example.com/a"),
		bridgeObservationForReport("obs:bridge:a-z", "example.com/a", "example.com/z"),
		incompleteObservationForReport("obs:incomplete:z", "package:example.com/z", "placeholder:package:example.com/missing"),
		incompleteObservationForReport("obs:incomplete:a", "package:example.com/a", "placeholder:package:example.com/missing"),
	}}

	report := buildDDDReport(graph, compactReportOptions{Limit: 50})
	if got := ids(report.Candidates, func(item dddCandidateSummary) string { return item.PackagePath }); !same(got, []string{"example.com/a", "example.com/z"}) {
		t.Fatalf("candidate order = %#v", got)
	}
	if got := ids(report.Bridges, func(item dddBridgeSummary) string { return item.FromPackagePath }); !same(got, []string{"example.com/a", "example.com/z"}) {
		t.Fatalf("bridge order = %#v", got)
	}
	if got := ids(report.IncompleteEvidence, func(item dddIncompleteEvidenceSummary) string { return item.From }); !same(got, []string{"package:example.com/a", "package:example.com/z"}) {
		t.Fatalf("incomplete order = %#v", got)
	}
}

func TestDDDReportLimitIsPerListAndSummaryCountsArePreLimit(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		candidateObservationForReport("obs:candidate:a", "example.com/a", "a,b", "2", "1"),
		candidateObservationForReport("obs:candidate:b", "example.com/b", "b,c", "2", "1"),
		bridgeObservationForReport("obs:bridge:a-b", "example.com/a", "example.com/b"),
		bridgeObservationForReport("obs:bridge:b-c", "example.com/b", "example.com/c"),
		incompleteObservationForReport("obs:incomplete:a", "package:example.com/a", "placeholder:package:example.com/missing-a"),
		incompleteObservationForReport("obs:incomplete:b", "package:example.com/b", "placeholder:package:example.com/missing-b"),
	}}

	report := buildDDDReport(graph, compactReportOptions{Limit: 1})
	if len(report.Candidates) != 1 || len(report.Bridges) != 1 || len(report.IncompleteEvidence) != 1 {
		t.Fatalf("limited report = %#v", report)
	}
	if report.Summary.Candidates != 2 || report.Summary.Bridges != 2 || report.Summary.IncompleteEvidence != 2 || !report.Summary.Truncated {
		t.Fatalf("summary = %#v, want pre-limit counts and truncated", report.Summary)
	}
}

func TestDDDReportTreatsNonPositiveLimitAsOne(t *testing.T) {
	graph := core.Graph{Observations: []core.Observation{
		candidateObservationForReport("obs:candidate:a", "example.com/a", "a,b", "2", "1"),
		candidateObservationForReport("obs:candidate:b", "example.com/b", "b,c", "2", "1"),
	}}

	report := buildDDDReport(graph, compactReportOptions{Limit: 0})
	if len(report.Candidates) != 1 || report.Summary.Candidates != 2 || !report.Summary.Truncated {
		t.Fatalf("report = %#v", report)
	}
}

func TestDDDReportEmptyGraphReturnsEmptySections(t *testing.T) {
	report := buildDDDReport(core.Graph{}, compactReportOptions{Limit: 50})
	if len(report.Candidates) != 0 || len(report.Bridges) != 0 || len(report.IncompleteEvidence) != 0 {
		t.Fatalf("report = %#v, want empty sections", report)
	}
	if report.Summary != (dddReportSummary{}) {
		t.Fatalf("summary = %#v, want zero value", report.Summary)
	}
}

func TestFixtureDDDReportHasReusableEvidence(t *testing.T) {
	report := buildDDDReport(buildFixtureGraph(t), compactReportOptions{Limit: 50})
	if len(report.Candidates) == 0 {
		t.Fatalf("report = %#v, want at least one language-zone candidate", report)
	}
	if len(report.Bridges) == 0 {
		t.Fatalf("report = %#v, want at least one bridge evidence item", report)
	}
	if report.Summary.Candidates != len(report.Candidates) || report.Summary.Bridges != len(report.Bridges) {
		t.Fatalf("summary = %#v, report = %#v", report.Summary, report)
	}
}

func candidateObservationForReport(id, packagePath, terms, termCount, cooccurrenceCount string) core.Observation {
	return core.Observation{
		ID:         id,
		Name:       core.ObservationNameLanguageZoneCandidate,
		TargetID:   core.PackageID(packagePath),
		TargetKind: "node",
		Attributes: map[string]string{
			"packagePath":       packagePath,
			"terms":             terms,
			"termCount":         termCount,
			"cooccurrenceCount": cooccurrenceCount,
		},
		Evidence:   []string{"obs:term", "obs:cooccurrence"},
		TrustLevel: core.TrustSyntaxObserved,
	}
}

func bridgeObservationForReport(id, fromPackagePath, toPackagePath string) core.Observation {
	return core.Observation{
		ID:         id,
		Name:       core.ObservationNameBridgeSymbol,
		TargetID:   core.EdgeID(core.EdgeKindImports, core.PackageID(fromPackagePath), core.PackageID(toPackagePath)),
		TargetKind: core.FactKindEdge,
		Attributes: map[string]string{
			"fromPackagePath": fromPackagePath,
			"toPackagePath":   toPackagePath,
			"edgeKind":        core.EdgeKindImports,
		},
		Evidence:   []string{"edge:imports", "obs:candidate:from", "obs:candidate:to"},
		TrustLevel: core.TrustSyntaxObserved,
	}
}

func incompleteObservationForReport(id, from, to string) core.Observation {
	return core.Observation{
		ID:         id,
		Name:       core.ObservationNameVocabularyIncompleteDependency,
		TargetID:   core.EdgeID(core.EdgeKindImports, from, to),
		TargetKind: core.FactKindEdge,
		Attributes: map[string]string{
			"from":     from,
			"to":       to,
			"edgeKind": core.EdgeKindImports,
			"reason":   "import_target_not_parsed_or_loaded",
		},
		Evidence:   []string{core.EdgeID(core.EdgeKindImports, from, to)},
		TrustLevel: core.TrustSyntaxObserved,
	}
}
