package main

import (
	"testing"

	"go-safedesign/internal/core"
)

func TestStatsReportEmptyGraphReturnsEmptySections(t *testing.T) {
	report := buildStatsReport(core.Graph{}, compactReportOptions{Limit: 50})
	if report.Summary != (graphStatsSummary{}) {
		t.Fatalf("summary = %#v, want zero value", report.Summary)
	}
	if len(report.Modules) != 0 || len(report.Packages) != 0 || len(report.Files) != 0 {
		t.Fatalf("report = %#v, want empty sections", report)
	}
}

func TestStatsReportCountsSyntheticGraph(t *testing.T) {
	graph := statsGraphFixture()
	report := buildStatsReport(graph, compactReportOptions{Limit: 50})

	if report.Summary.Modules != 3 || report.Summary.Packages != 2 || report.Summary.Files != 2 {
		t.Fatalf("summary = %#v, module/package/file counts wrong", report.Summary)
	}
	if report.Summary.Functions != 1 || report.Summary.Methods != 1 || report.Summary.Types != 1 || report.Summary.Structs != 1 || report.Summary.Interfaces != 1 || report.Summary.Imports != 1 {
		t.Fatalf("summary = %#v, node kind counts wrong", report.Summary)
	}
	if report.Summary.Edges != 5 || report.Summary.Observations != 5 || report.Summary.Placeholders != 2 || report.Summary.IncompleteEdges != 2 {
		t.Fatalf("summary = %#v, edge/observation/incomplete counts wrong", report.Summary)
	}
}

func TestStatsReportWorkspaceMetadataWarnsWhenNestedGoModsAreNotIndexed(t *testing.T) {
	graph := core.Graph{
		SourceRecords: []core.SourceRecord{
			{Kind: "project_root", Path: "."},
			{Kind: "workspace_root", Path: "."},
			{Kind: "go_mod", Path: "go.mod"},
			{Kind: "go_mod", Path: "examples/go.mod"},
		},
		Nodes: []core.Node{
			{ID: core.ModuleID("example.com/app"), Kind: core.NodeKindModule, ModulePath: "example.com/app"},
		},
	}

	report := buildStatsReport(graph, compactReportOptions{Limit: 50})
	if !report.Workspace.SingleModuleScan || report.Workspace.IndexedModules != 1 {
		t.Fatalf("workspace = %#v, want single-module scan with one indexed module", report.Workspace)
	}
	if !same(report.Workspace.GoModFiles, []string{"examples/go.mod", "go.mod"}) {
		t.Fatalf("workspace = %#v, want sorted go.mod files", report.Workspace)
	}
	if report.Workspace.Warning == "" {
		t.Fatalf("workspace = %#v, want nested go.mod warning", report.Workspace)
	}
}

func TestStatsReportWorkspaceMetadataDoesNotWarnForExplicitWorkspace(t *testing.T) {
	graph := core.Graph{
		SourceRecords: []core.SourceRecord{
			{Kind: "project_root", Path: "shop"},
			{Kind: "workspace_root", Path: "."},
			{Kind: "go_mod", Path: "shop/go.mod"},
			{Kind: "go_mod", Path: "payments/go.mod"},
		},
		Nodes: []core.Node{
			{ID: core.ModuleID("example.com/shop"), Kind: core.NodeKindModule, ModulePath: "example.com/shop"},
			{ID: core.ModuleID("example.com/payments"), Kind: core.NodeKindModule, ModulePath: "example.com/payments"},
		},
	}

	report := buildStatsReport(graph, compactReportOptions{Limit: 50})
	if report.Workspace.SingleModuleScan || report.Workspace.IndexedModules != 2 || report.Workspace.Warning != "" {
		t.Fatalf("workspace = %#v, want explicit multi-module workspace metadata", report.Workspace)
	}
}

func TestStatsReportModuleStatsIncludeRelationshipsAndMissingDependencies(t *testing.T) {
	report := buildStatsReport(statsGraphFixture(), compactReportOptions{Limit: 50})

	app := moduleStatsByPath(report.Modules, "example.com/app")
	if app == nil {
		t.Fatalf("modules = %#v, missing app module stats", report.Modules)
	}
	if app.Dir != "app" || !same(app.DiscoveryReasons, []string{"entry_module", "go_mod_replace"}) {
		t.Fatalf("app module stats = %#v, discovery metadata wrong", *app)
	}
	if app.Packages != 1 || app.Files != 1 || app.Imports != 1 || app.IncompleteImports != 1 {
		t.Fatalf("app module stats = %#v, basic counts wrong", *app)
	}
	if app.DependsOn != 1 || app.DependedOnBy != 0 || app.MissingDependencies != 1 || app.ExternalDependencies != 1 {
		t.Fatalf("app module stats = %#v, dependency counts wrong", *app)
	}
	if app.CrossModuleImportsOut != 1 || app.CrossModuleImportsIn != 0 || app.Candidates != 1 || app.BridgesOut != 1 || app.BridgesIn != 0 {
		t.Fatalf("app module stats = %#v, relationship evidence wrong", *app)
	}

	payment := moduleStatsByPath(report.Modules, "example.com/payment")
	if payment == nil || payment.DependedOnBy != 1 || payment.CrossModuleImportsIn != 1 || payment.BridgesIn != 1 {
		t.Fatalf("payment module stats = %#v", payment)
	}
	unused := moduleStatsByPath(report.Modules, "example.com/unused")
	if unused == nil || unused.Packages != 0 || unused.DependsOn != 0 || unused.DependedOnBy != 0 {
		t.Fatalf("unused module stats = %#v", unused)
	}
	if missing := moduleStatsByPath(report.Modules, "example.com/missing-module"); missing != nil {
		t.Fatalf("modules = %#v, missing placeholder module should not be reported as discovered module", report.Modules)
	}
}

func TestStatsReportPackageStatsIncludeIncompleteImportsAndBridgeCounts(t *testing.T) {
	report := buildStatsReport(statsGraphFixture(), compactReportOptions{Limit: 50})

	order := packageStatsByPath(report.Packages, "example.com/app/order")
	if order == nil {
		t.Fatalf("packages = %#v, missing order stats", report.Packages)
	}
	if order.Files != 1 || order.Functions != 1 || order.Methods != 1 || order.Types != 1 || order.Structs != 1 || order.Interfaces != 1 || order.Imports != 1 || order.IncompleteImports != 1 {
		t.Fatalf("order stats = %#v", *order)
	}
	if order.Observations != 4 || order.Candidates != 1 || order.BridgesOut != 1 || order.BridgesIn != 0 {
		t.Fatalf("order stats = %#v, DDD stats wrong", *order)
	}

	payment := packageStatsByPath(report.Packages, "example.com/app/payment")
	if payment == nil || payment.BridgesIn != 1 || payment.BridgesOut != 0 {
		t.Fatalf("payment stats = %#v", payment)
	}
}

func TestStatsReportFileStatsCountSourceScopedFacts(t *testing.T) {
	report := buildStatsReport(statsGraphFixture(), compactReportOptions{Limit: 50})

	file := fileStatsBySource(report.Files, "order/service.go")
	if file == nil {
		t.Fatalf("files = %#v, missing order/service.go stats", report.Files)
	}
	if file.PackagePath != "example.com/app/order" || file.Functions != 1 || file.Methods != 1 || file.Types != 1 || file.Structs != 1 || file.Interfaces != 1 || file.Observations != 1 {
		t.Fatalf("file stats = %#v", *file)
	}
}

func TestStatsReportScopePackageFiltersOutputOnly(t *testing.T) {
	report := buildStatsReport(statsGraphFixture(), compactReportOptions{Limit: 50, ScopePackage: "example.com/app/order"})

	if report.Summary.Packages != 1 || report.Summary.Files != 1 {
		t.Fatalf("summary = %#v, want scoped package/file counts", report.Summary)
	}
	if len(report.Packages) != 2 {
		t.Fatalf("packages = %#v, want scoped package plus bridge target package", report.Packages)
	}
	if order := packageStatsByPath(report.Packages, "example.com/app/order"); order == nil || order.BridgesOut != 1 {
		t.Fatalf("packages = %#v, missing scoped bridge out", report.Packages)
	}
	if payment := packageStatsByPath(report.Packages, "example.com/app/payment"); payment == nil || payment.BridgesIn != 1 {
		t.Fatalf("packages = %#v, missing scoped bridge in target", report.Packages)
	}
	if len(report.Files) != 1 || report.Files[0].SourceFile != "order/service.go" {
		t.Fatalf("files = %#v, want only scoped package file", report.Files)
	}
}

func TestStatsReportScopeModuleFiltersOutputOnly(t *testing.T) {
	report := buildStatsReport(statsGraphFixture(), compactReportOptions{Limit: 50, ScopeModule: "example.com/app"})

	if report.Summary.Modules != 1 || report.Summary.Packages != 1 || report.Summary.Files != 1 {
		t.Fatalf("summary = %#v, want scoped module counts", report.Summary)
	}
	app := moduleStatsByPath(report.Modules, "example.com/app")
	if app == nil || app.DependsOn != 1 || app.MissingDependencies != 1 || app.ExternalDependencies != 1 || app.CrossModuleImportsOut != 1 {
		t.Fatalf("modules = %#v, missing scoped app relationship counts", report.Modules)
	}
	payment := moduleStatsByPath(report.Modules, "example.com/payment")
	if payment == nil || payment.DependedOnBy != 1 || payment.CrossModuleImportsIn != 1 {
		t.Fatalf("modules = %#v, missing scoped adjacent payment relationship counts", report.Modules)
	}
	if len(report.Files) != 1 || report.Files[0].ModulePath != "example.com/app" {
		t.Fatalf("files = %#v, want only scoped module files", report.Files)
	}
}

func TestDDDReportScopePackageFiltersEvidenceSections(t *testing.T) {
	report := buildDDDReport(statsGraphFixture(), compactReportOptions{Limit: 50, ScopePackage: "example.com/app/order"})

	if len(report.Candidates) != 1 || report.Candidates[0].PackagePath != "example.com/app/order" {
		t.Fatalf("candidates = %#v", report.Candidates)
	}
	if len(report.Bridges) != 1 || report.Bridges[0].FromPackagePath != "example.com/app/order" {
		t.Fatalf("bridges = %#v", report.Bridges)
	}
	if len(report.IncompleteEvidence) != 1 || report.IncompleteEvidence[0].From != core.PackageID("example.com/app/order") {
		t.Fatalf("incomplete evidence = %#v", report.IncompleteEvidence)
	}
}

func TestDDDReportScopeModuleFiltersEvidenceSections(t *testing.T) {
	report := buildDDDReport(statsGraphFixture(), compactReportOptions{Limit: 50, ScopeModule: "example.com/app"})

	if len(report.Candidates) != 1 || report.Candidates[0].PackagePath != "example.com/app/order" {
		t.Fatalf("candidates = %#v", report.Candidates)
	}
	if len(report.Bridges) != 1 || report.Bridges[0].FromPackagePath != "example.com/app/order" {
		t.Fatalf("bridges = %#v", report.Bridges)
	}
	if len(report.IncompleteEvidence) != 1 || report.IncompleteEvidence[0].From != core.PackageID("example.com/app/order") {
		t.Fatalf("incomplete evidence = %#v", report.IncompleteEvidence)
	}
}

func statsGraphFixture() core.Graph {
	orderPackage := core.PackageID("example.com/app/order")
	paymentPackage := core.PackageID("example.com/app/payment")
	missingPackage := core.PlaceholderPackageID("example.com/app/missing")
	missingModule := core.PlaceholderModuleID("example.com/missing-module")
	incompleteEdge := core.EdgeID(core.EdgeKindImports, orderPackage, missingPackage)
	bridgeEdge := core.EdgeID(core.EdgeKindImports, orderPackage, paymentPackage)
	return core.Graph{
		SourceRecords: []core.SourceRecord{
			{Kind: "project_root", Path: "app"},
			{Kind: "workspace_root", Path: "."},
			{Kind: "go_work", Path: "go.work"},
			{Kind: "go_mod", Path: "app/go.mod"},
			{Kind: "go_mod", Path: "payment/go.mod"},
			{Kind: "module_discovery", Path: "app", Reason: "entry_module,go_mod_replace"},
			{Kind: "module_discovery", Path: "payment", Reason: "workspace_scan"},
			{Kind: "module_discovery", Path: "unused", Reason: "workspace_scan"},
		},
		Nodes: []core.Node{
			{ID: core.ModuleID("example.com/app"), Kind: core.NodeKindModule, Name: "example.com/app", SourceFile: "app/go.mod", ModulePath: "example.com/app"},
			{ID: core.ModuleID("example.com/payment"), Kind: core.NodeKindModule, Name: "example.com/payment", SourceFile: "payment/go.mod", ModulePath: "example.com/payment"},
			{ID: core.ModuleID("example.com/unused"), Kind: core.NodeKindModule, Name: "example.com/unused", SourceFile: "unused/go.mod", ModulePath: "example.com/unused"},
			{ID: orderPackage, Kind: core.NodeKindPackage, Name: "example.com/app/order", PackagePath: "example.com/app/order", ModulePath: "example.com/app"},
			{ID: paymentPackage, Kind: core.NodeKindPackage, Name: "example.com/app/payment", PackagePath: "example.com/app/payment", ModulePath: "example.com/payment"},
			{ID: "file:order/service.go", Kind: core.NodeKindFile, SourceFile: "order/service.go", PackagePath: "example.com/app/order", ModulePath: "example.com/app"},
			{ID: "file:payment/client.go", Kind: core.NodeKindFile, SourceFile: "payment/client.go", PackagePath: "example.com/app/payment", ModulePath: "example.com/payment"},
			{ID: "function:order.Place", Kind: core.NodeKindFunction, SourceFile: "order/service.go", PackagePath: "example.com/app/order", ModulePath: "example.com/app"},
			{ID: "method:order.Service.Place", Kind: core.NodeKindMethod, SourceFile: "order/service.go", PackagePath: "example.com/app/order", ModulePath: "example.com/app"},
			{ID: "type:order.ID", Kind: core.NodeKindType, SourceFile: "order/service.go", PackagePath: "example.com/app/order", ModulePath: "example.com/app"},
			{ID: "struct:order.Order", Kind: core.NodeKindStruct, SourceFile: "order/service.go", PackagePath: "example.com/app/order", ModulePath: "example.com/app"},
			{ID: "interface:order.Store", Kind: core.NodeKindInterface, SourceFile: "order/service.go", PackagePath: "example.com/app/order", ModulePath: "example.com/app"},
			{ID: "field:order.Order.ID", Kind: core.NodeKindField, SourceFile: "order/service.go", PackagePath: "example.com/app/order", ModulePath: "example.com/app"},
			{ID: "import:order:missing", Kind: core.NodeKindImport, SourceFile: "order/service.go", PackagePath: "example.com/app/order", ModulePath: "example.com/app"},
			{ID: missingPackage, Kind: core.NodeKindPlaceholder, Synthetic: true, PackagePath: "example.com/app/missing"},
			{ID: missingModule, Kind: core.NodeKindPlaceholder, Synthetic: true, ModulePath: "example.com/missing-module"},
		},
		Edges: []core.Edge{
			{ID: core.EdgeID(core.EdgeKindContains, orderPackage, "file:order/service.go"), Kind: core.EdgeKindContains, From: orderPackage, To: "file:order/service.go", Complete: true},
			{ID: incompleteEdge, Kind: core.EdgeKindImports, From: orderPackage, To: missingPackage, Complete: false, Synthetic: true, SourceFile: "order/service.go"},
			{ID: bridgeEdge, Kind: core.EdgeKindImports, From: orderPackage, To: paymentPackage, Complete: true, SourceFile: "order/service.go"},
			{ID: core.EdgeID(core.EdgeKindDependsOn, core.ModuleID("example.com/app"), core.ModuleID("example.com/payment")), Kind: core.EdgeKindDependsOn, From: core.ModuleID("example.com/app"), To: core.ModuleID("example.com/payment"), Complete: true},
			{ID: core.EdgeID(core.EdgeKindDependsOn, core.ModuleID("example.com/app"), missingModule), Kind: core.EdgeKindDependsOn, From: core.ModuleID("example.com/app"), To: missingModule, Complete: false, Synthetic: true},
		},
		Observations: []core.Observation{
			{ID: "obs:term:order", Name: core.ObservationNameVocabularyTerm, TargetID: "function:order.Place", SourceFile: "order/service.go", Attributes: map[string]string{"packagePath": "example.com/app/order"}},
			candidateObservationForReport("obs:candidate:order", "example.com/app/order", "order,place", "2", "1"),
			candidateObservationForReport("obs:candidate:payment", "example.com/app/payment", "payment,client", "2", "1"),
			bridgeObservationForReport("obs:bridge:order-payment", "example.com/app/order", "example.com/app/payment"),
			incompleteObservationForReport("obs:incomplete:order-missing", orderPackage, missingPackage),
		},
	}
}

func moduleStatsByPath(items []moduleStats, modulePath string) *moduleStats {
	for i := range items {
		if items[i].ModulePath == modulePath {
			return &items[i]
		}
	}
	return nil
}

func packageStatsByPath(items []packageStats, packagePath string) *packageStats {
	for i := range items {
		if items[i].PackagePath == packagePath {
			return &items[i]
		}
	}
	return nil
}

func fileStatsBySource(items []fileStats, sourceFile string) *fileStats {
	for i := range items {
		if items[i].SourceFile == sourceFile {
			return &items[i]
		}
	}
	return nil
}
