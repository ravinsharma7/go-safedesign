package core

import "testing"

func TestIDConstructorsPreserveWireFormat(t *testing.T) {
	if got := PackageID("example.com/shop/order"); got != "package:example.com/shop/order" {
		t.Fatalf("PackageID = %q", got)
	}
	if got := ModuleID("example.com/shop"); got != "module:example.com/shop" {
		t.Fatalf("ModuleID = %q", got)
	}
	if got := PlaceholderPackageID("example.com/missing/notify"); got != "placeholder:package:example.com/missing/notify" {
		t.Fatalf("PlaceholderPackageID = %q", got)
	}
	if got := PlaceholderModuleID("example.com/missing"); got != "placeholder:module:example.com/missing" {
		t.Fatalf("PlaceholderModuleID = %q", got)
	}
	if got := EdgeID(EdgeKindImports, PackageID("a"), PackageID("b")); got != "edge:imports:package:a->package:b" {
		t.Fatalf("EdgeID = %q", got)
	}
}

func TestPlaceholderHelpers(t *testing.T) {
	if !IsPlaceholderID("placeholder:package:example.com/missing") {
		t.Fatal("placeholder package id not recognized")
	}
	if !IsPlaceholderNode(Node{ID: "package:example.com/missing", Kind: NodeKindPackage, Synthetic: true}) {
		t.Fatal("synthetic node not recognized as placeholder-backed")
	}
	if !IsPlaceholderTarget(Edge{To: "placeholder:module:example.com/missing"}) {
		t.Fatal("placeholder target not recognized")
	}
}

func TestIncompleteEdgeHelper(t *testing.T) {
	cases := []Edge{
		{Complete: false},
		{Complete: true, Synthetic: true},
		{Complete: true, To: "placeholder:package:example.com/missing"},
	}
	for _, edge := range cases {
		if !IsIncompleteEdge(edge) {
			t.Fatalf("edge = %#v, want incomplete", edge)
		}
	}
	if IsIncompleteEdge(Edge{Complete: true, To: "package:example.com/ok"}) {
		t.Fatal("complete real edge reported incomplete")
	}
}

func TestObservationSourceValidation(t *testing.T) {
	for _, source := range []string{ObservationSourceObserved, ObservationSourceConfigured, ObservationSourceInferred, ObservationSourceImported} {
		if !IsObservationSource(source) {
			t.Fatalf("source %q should be valid", source)
		}
	}
	if IsObservationSource("external") {
		t.Fatal("unexpected source accepted")
	}
}
