package main

import (
	"reflect"
	"testing"

	"github.com/ravinsharma7/go-safedesign/internal/core"
	"github.com/ravinsharma7/go-safedesign/internal/indexer"
)

func TestBuildValueDiscoveryReportUsesCurrentTaxonomyAndPlanner(t *testing.T) {
	report := buildValueDiscoveryReport()

	if !reflect.DeepEqual(report.JSONSections, graphJSONSectionNames()) {
		t.Fatalf("json sections drifted: %#v", report.JSONSections)
	}
	if !reflect.DeepEqual(report.NodeKinds, core.NodeKinds()) {
		t.Fatalf("node kinds drifted: %#v", report.NodeKinds)
	}
	if !reflect.DeepEqual(report.EdgeKinds, core.EdgeKinds()) {
		t.Fatalf("edge kinds drifted: %#v", report.EdgeKinds)
	}
	if !reflect.DeepEqual(report.FactKinds, core.FactKinds()) {
		t.Fatalf("fact kinds drifted: %#v", report.FactKinds)
	}
	if !reflect.DeepEqual(report.Statuses, core.Statuses()) {
		t.Fatalf("statuses drifted: %#v", report.Statuses)
	}
	if !reflect.DeepEqual(report.Freshness, core.FreshnessStatuses()) {
		t.Fatalf("freshness values drifted: %#v", report.Freshness)
	}
	if !reflect.DeepEqual(report.ObservationNames, core.ObservationNames()) {
		t.Fatalf("observation names drifted: %#v", report.ObservationNames)
	}
	if !reflect.DeepEqual(report.ObservationSources, core.ObservationSources()) {
		t.Fatalf("observation sources drifted: %#v", report.ObservationSources)
	}
	if !reflect.DeepEqual(report.TrustLevels, core.TrustLevelInfos()) {
		t.Fatalf("trust levels drifted: %#v", report.TrustLevels)
	}
	if !reflect.DeepEqual(report.AnalyzerIDs, indexer.KnownAnalyzerIDs()) {
		t.Fatalf("analyzer IDs drifted: %#v", report.AnalyzerIDs)
	}
}

func TestBuildValueDiscoveryReportIncludesUsefulValues(t *testing.T) {
	report := buildValueDiscoveryReport()

	if !containsString(report.JSONSections, "observations") {
		t.Fatalf("expected observations section: %#v", report.JSONSections)
	}
	if !containsString(report.NodeKinds, core.NodeKindPackage) {
		t.Fatalf("expected package node kind: %#v", report.NodeKinds)
	}
	if !containsString(report.EdgeKinds, core.EdgeKindImports) {
		t.Fatalf("expected imports edge kind: %#v", report.EdgeKinds)
	}
	if !containsString(report.FactKinds, core.FactKindObservation) {
		t.Fatalf("expected observation fact kind: %#v", report.FactKinds)
	}
	if !containsString(report.ObservationSources, core.ObservationSourceInferred) {
		t.Fatalf("expected inferred observation source: %#v", report.ObservationSources)
	}
	if !containsString(report.ObservationNames, core.ObservationNameLanguageZoneCandidate) {
		t.Fatalf("expected language-zone observation name: %#v", report.ObservationNames)
	}
	if !containsTrustLevel(report.TrustLevels, core.TrustSyntaxObserved) {
		t.Fatalf("expected syntax-observed trust level: %#v", report.TrustLevels)
	}
	if !containsString(report.AnalyzerIDs, indexer.AnalyzerIDLanguageZoneCandidate) {
		t.Fatalf("expected language-zone analyzer ID: %#v", report.AnalyzerIDs)
	}
}

func containsTrustLevel(values []core.TrustLevelInfo, want core.TrustLevel) bool {
	for _, value := range values {
		if value.Level == want {
			return true
		}
	}
	return false
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
