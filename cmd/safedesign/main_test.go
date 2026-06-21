package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/ravinsharma7/go-safedesign/internal/core"
	"github.com/ravinsharma7/go-safedesign/internal/indexer"
	"github.com/ravinsharma7/go-safedesign/internal/pipeline"
)

func TestPrototypeFixtureGraph(t *testing.T) {
	graph := buildFixtureGraph(t)

	if got := countKind(graph.Nodes, "file"); got < 5 {
		t.Fatalf("file count = %d, want >= 5", got)
	}
	if got := countKind(graph.Nodes, "package"); got < 3 {
		t.Fatalf("package count = %d, want >= 3", got)
	}
	if got := countKind(graph.Nodes, "module"); got < 3 {
		t.Fatalf("module count = %d, want >= 3", got)
	}

	missing := nodeByID(graph, "placeholder:package:example.com/missing-notification/notify")
	if missing == nil {
		t.Fatal("missing notification package placeholder was not created")
	}
	if !missing.Synthetic || missing.TrustLevel != core.TrustSyntaxObserved {
		t.Fatalf("missing placeholder = %#v, want synthetic syntax_observed", *missing)
	}

	if edgeByID(graph, "edge:imports:package:example.com/shop/order->placeholder:package:example.com/missing-notification/notify") == nil {
		t.Fatal("missing notification import edge was not preserved as a partial placeholder edge")
	}
	if edgeByID(graph, "edge:imports:package:example.com/shop/order->package:example.com/shop/paymentclient") == nil {
		t.Fatal("same-module import placeholder was not reconciled to the real package")
	}

	call := nodeByID(graph, "unresolved_call:shop/order/service.go:23:2-23:25:notification.Notify")
	if call == nil {
		t.Fatal("unresolved notification call was not captured")
	}
	if !call.Synthetic || call.Kind != "unresolved_call" {
		t.Fatalf("notification call = %#v, want synthetic unresolved_call", *call)
	}

	runtimeMarkers := namesByKind(graph.Nodes, "runtime_marker")
	for _, want := range []string{"go_statement", "defer_statement", "channel_send", "channel_receive", "select_statement"} {
		if !contains(runtimeMarkers, want) {
			t.Fatalf("runtime markers = %v, missing %s", runtimeMarkers, want)
		}
	}

	payments := nodeByID(graph, "package:example.com/payments/gateway")
	if payments == nil || core.TrustRank(payments.TrustLevel) < core.TrustRank(core.TrustPackageLoaded) {
		t.Fatalf("payments package = %#v, want package_loaded or better", payments)
	}

	if !hasQuery(graph, "query:policy:example.com/shop/order", "fail", "policy_result_failed") {
		t.Fatalf("query result = %#v, missing policy failure query", graph.Queries)
	}
	if !hasQuery(graph, "query:complexity:package:example.com/shop/order", "pass", "all_complexity_metrics_passed") {
		t.Fatalf("query result = %#v, missing package complexity pass query", graph.Queries)
	}
	if !hasQuery(graph, "query:complexity:domain:example.com/shop/order", "unknown", "complexity_analysis_incomplete") {
		t.Fatalf("query result = %#v, missing domain complexity incomplete query", graph.Queries)
	}
	if !hasFreshness(graph, "file:shop/order/service.go", "superseded") {
		t.Fatalf("freshness = %#v, want simulated superseded record", graph.Freshness)
	}
	if !hasSourceRecord(graph, "go_mod", "shop/go.mod") || !hasSourceRecord(graph, "config", "shop/safedesign.config.json") {
		t.Fatalf("source records = %#v, missing module/config discovery", graph.SourceRecords)
	}
	if len(graph.Labels) < 8 || len(graph.Warnings) < 2 {
		t.Fatalf("labels/warnings = %#v %#v, want module dependency and ubiquitous language labels plus policy warnings", graph.Labels, graph.Warnings)
	}
	if !hasLabel(graph, "module.dependency", "direct", "edge:depends_on:module:example.com/shop->module:example.com/missing-notification") {
		t.Fatalf("labels = %#v, missing direct dependency label", graph.Labels)
	}
	if !hasLabel(graph, "ddd.context", "ordering", "package:example.com/shop/order") {
		t.Fatalf("labels = %#v, missing ordering context label", graph.Labels)
	}
	if !hasLabel(graph, "ddd.context", "payments", "package:example.com/payments/gateway") {
		t.Fatalf("labels = %#v, missing payments context label", graph.Labels)
	}
	if !hasLabel(graph, "ul.term", "order", "package:example.com/shop/order") || !hasLabel(graph, "ul.term", "payment", "package:example.com/shop/order") || !hasLabel(graph, "ul.term", "gateway", "package:example.com/payments/gateway") {
		t.Fatalf("labels = %#v, missing ubiquitous language term labels", graph.Labels)
	}
	if !hasWarning(graph, "fail", "policy_violation: example.com/shop/order imports package outside allow list example.com/payments/gateway", "edge:imports:package:example.com/shop/order->package:example.com/payments/gateway") {
		t.Fatalf("warnings = %#v, missing policy violation warning", graph.Warnings)
	}
	if !hasWarning(graph, "unknown", "policy_unknown: import target incomplete for example.com/missing-notification/notify", "edge:imports:package:example.com/shop/order->placeholder:package:example.com/missing-notification/notify") {
		t.Fatalf("warnings = %#v, missing policy unknown warning", graph.Warnings)
	}
	violation := edgeByID(graph, "edge:violates:policy_result:order-package-imports:edge:imports:package:example.com/shop/order->package:example.com/payments/gateway")
	if violation == nil || violation.From != "package:example.com/shop/order" || violation.To != "package:example.com/payments/gateway" || violation.RunID == "" {
		t.Fatalf("violation edge = %#v, want policy violation edge with provenance", violation)
	}
	graphJSON, err := json.Marshal(graph)
	if err != nil {
		t.Fatal(err)
	}
	if !containsJSONFragment(graphJSON, `"sourceRecords":[`) || !containsJSONFragment(graphJSON, `"observations":[`) || !containsJSONFragment(graphJSON, `"labels":[`) || !containsJSONFragment(graphJSON, `"warnings":[`) {
		t.Fatalf("graph JSON missing additive collections: %s", graphJSON[:min(len(graphJSON), 400)])
	}
	if len(graph.PathJobs) != 0 {
		t.Fatalf("path jobs = %#v, want none without explicit query config", graph.PathJobs)
	}
	wantStages := []string{
		string(pipeline.StageSourceDiscovery),
		string(pipeline.StageModuleExtraction),
		string(pipeline.StageSyntaxExtraction),
		string(pipeline.StageBaseGraphAssembly),
		string(pipeline.StagePackageLoading),
		string(pipeline.StageModuleDependencyEnrichment),
		string(pipeline.StageDDDClassification),
		string(pipeline.StageComplexityMetrics),
		string(pipeline.StagePolicyEvaluation),
		string(pipeline.StageQueryMaterialization),
	}
	for _, stage := range wantStages {
		if !hasRun(graph, stage) {
			t.Fatalf("runs = %#v, missing stage %s", graph.Runs, stage)
		}
	}
	policyRun := runByStage(graph, string(pipeline.StagePolicyEvaluation))
	if policyRun == nil || policyRun.AnalyzerID != "dependency_policy" || policyRun.Status != "partial" {
		t.Fatalf("policy run = %#v, want dependency_policy partial", policyRun)
	}
	complexityRun := runByStage(graph, string(pipeline.StageComplexityMetrics))
	if complexityRun == nil || complexityRun.AnalyzerID != "complexity" || complexityRun.Status != "partial" || complexityRun.ConfigurationHash == "" {
		t.Fatalf("complexity run = %#v, want complexity partial with config hash", complexityRun)
	}
	moddepRun := runByStage(graph, string(pipeline.StageModuleDependencyEnrichment))
	if moddepRun == nil || moddepRun.AnalyzerID != "module_dependency_enrichment" || moddepRun.Status != "completed" {
		t.Fatalf("module dependency enrichment run = %#v, want completed run", moddepRun)
	}
	vocabRun := runByAnalyzer(graph, "vocabulary_extraction")
	if vocabRun == nil || vocabRun.Status != "completed" || vocabRun.EmittedFacts == 0 {
		t.Fatalf("vocabulary extraction run = %#v, want completed run with observations", vocabRun)
	}
	vocabCoRun := runByAnalyzer(graph, "vocabulary_cooccurrence")
	if vocabCoRun == nil || vocabCoRun.Status != "completed" || vocabCoRun.EmittedFacts == 0 {
		t.Fatalf("vocabulary cooccurrence run = %#v, want completed run with observations", vocabCoRun)
	}
	langZoneRun := runByAnalyzer(graph, "language_zone_candidate")
	if langZoneRun == nil || langZoneRun.Status != "completed" || langZoneRun.EmittedFacts == 0 {
		t.Fatalf("language zone candidate run = %#v, want completed run with observations", langZoneRun)
	}
	bridgeRun := runByAnalyzer(graph, "bridge_symbol")
	if bridgeRun == nil || bridgeRun.Status != "completed" || bridgeRun.EmittedFacts == 0 {
		t.Fatalf("bridge symbol run = %#v, want completed run with observations", bridgeRun)
	}
	dddRun := runByAnalyzer(graph, "ubiquitous_language")
	if dddRun == nil || dddRun.AnalyzerID != "ubiquitous_language" || dddRun.Status != "completed" || dddRun.ConfigurationHash == "" {
		t.Fatalf("ddd classification run = %#v, want ubiquitous language completed run with config hash", dddRun)
	}
	for _, run := range graph.Runs {
		if run.RunID == "" || run.StartedAt == "" || run.FinishedAt == "" {
			t.Fatalf("run = %#v, want enriched run metadata", run)
		}
	}
	if len(graph.Metrics) != 20 {
		t.Fatalf("metrics = %#v, want complexity metrics for 6 functions plus ubiquitous language alignment metrics", graph.Metrics)
	}
	if !hasMetric(graph, "cyclomatic_complexity", "warning", "function:example.com/shop/paymentclient.Await") {
		t.Fatalf("metrics = %#v, missing paymentclient Await warning", graph.Metrics)
	}
	if !hasMetricValue(graph, "runtime_marker_count", "function:example.com/shop/order.PlaceOrder", 4) {
		t.Fatalf("metrics = %#v, missing order PlaceOrder runtime marker count", graph.Metrics)
	}
	if !hasMetric(graph, "ubiquitous_language_alignment", "pass", "package:example.com/shop/order") {
		t.Fatalf("metrics = %#v, missing ubiquitous language alignment metric", graph.Metrics)
	}
	if !hasObservation(graph, core.ObservationNameVocabularyCooccurrence) {
		t.Fatalf("observations = %#v, missing vocabulary cooccurrence observation", graph.Observations)
	}
	if !hasObservation(graph, core.ObservationNameLanguageZoneCandidate) {
		t.Fatalf("observations = %#v, missing language zone candidate observation", graph.Observations)
	}
	if !hasObservation(graph, core.ObservationNameBridgeSymbol) {
		t.Fatalf("observations = %#v, missing bridge symbol observation", graph.Observations)
	}
	if len(graph.PolicyResults) != 4 {
		t.Fatalf("policy results = %#v, want 4", graph.PolicyResults)
	}
	if !hasPolicyResult(graph, "order-package-imports", "fail", "example.com/payments/gateway") {
		t.Fatalf("policy results = %#v, missing order gateway failure", graph.PolicyResults)
	}
	if !hasPolicyResult(graph, "order-package-imports", "unknown", "example.com/missing-notification/notify") {
		t.Fatalf("policy results = %#v, missing order missing-notification unknown", graph.PolicyResults)
	}
	if !hasPolicyResult(graph, "order-package-imports", "pass", "example.com/shop/paymentclient") {
		t.Fatalf("policy results = %#v, missing order paymentclient pass", graph.PolicyResults)
	}
	if !hasPolicyResult(graph, "paymentclient-package-imports", "pass", "example.com/payments/gateway") {
		t.Fatalf("policy results = %#v, missing paymentclient gateway pass", graph.PolicyResults)
	}
	if !hasDiagnostic(graph, "fail", "policy_violation: example.com/shop/order imports package outside allow list example.com/payments/gateway") {
		t.Fatalf("diagnostics = %#v, missing disallowed import failure", graph.Diagnostics)
	}
	if !hasDiagnostic(graph, "unknown", "policy_unknown: import target incomplete for example.com/missing-notification/notify") {
		t.Fatalf("diagnostics = %#v, missing incomplete import unknown", graph.Diagnostics)
	}
	if hasDiagnostic(graph, "fail", "example.com/shop/paymentclient imports package outside allow list example.com/payments/gateway") {
		t.Fatalf("diagnostics = %#v, allowed paymentclient import should not fail", graph.Diagnostics)
	}
}

func TestDisableComplexityRemovesMetricsAndUnknownsQuery(t *testing.T) {
	graph, err := indexer.BuildGraph(indexer.Options{
		Path:              filepath.Join("..", "..", "testdata", "workspace", "shop"),
		WorkspaceRoot:     filepath.Join("..", "..", "testdata", "workspace"),
		SimulateChange:    true,
		DisableComplexity: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if hasMetricName(graph, "cyclomatic_complexity") || hasMetricName(graph, "decision_count") || hasMetricName(graph, "runtime_marker_count") {
		t.Fatalf("metrics = %#v, want no complexity metrics", graph.Metrics)
	}
	if !hasQuery(graph, "query:complexity:package:example.com/shop/order", "unknown", "no_complexity_metrics_for_scope") {
		t.Fatalf("query result = %#v, missing disabled package complexity unknown", graph.Queries)
	}
	if !hasQuery(graph, "query:complexity:domain:example.com/shop/order", "unknown", "no_complexity_metrics_for_scope") {
		t.Fatalf("query result = %#v, missing disabled domain complexity unknown", graph.Queries)
	}
	if !hasQuery(graph, "query:policy:example.com/shop/order", "fail", "policy_result_failed") {
		t.Fatalf("query result = %#v, policy behavior changed", graph.Queries)
	}
}

func TestFixtureProblemsReportIncludesExpectedProblems(t *testing.T) {
	graph := buildFixtureGraph(t)
	report := buildProblemsReport(graph, 50)

	if report.Summary.PolicyResults != 2 {
		t.Fatalf("summary = %#v, want two non-pass policy results", report.Summary)
	}
	if report.Summary.Metrics != 1 {
		t.Fatalf("summary = %#v, want one warning metric", report.Summary)
	}
	if report.Summary.Warnings != 2 || len(report.Warnings) != 2 {
		t.Fatalf("summary/report warnings = %#v %#v, want two policy warnings", report.Summary, report.Warnings)
	}
	if report.Summary.Queries != 4 {
		t.Fatalf("summary = %#v, want four non-pass queries", report.Summary)
	}
	if report.Summary.Runs < 1 {
		t.Fatalf("summary = %#v, want partial runs", report.Summary)
	}
	if !reportHasPolicyResult(report, "fail", "example.com/payments/gateway") {
		t.Fatalf("policy results = %#v, missing fixture policy failure", report.PolicyResults)
	}
	if !reportHasPolicyResult(report, "unknown", "example.com/missing-notification/notify") {
		t.Fatalf("policy results = %#v, missing fixture policy unknown", report.PolicyResults)
	}
	if !reportHasMetric(report, "warning", "function:example.com/shop/paymentclient.Await") {
		t.Fatalf("metrics = %#v, missing fixture complexity warning", report.Metrics)
	}
	if !reportHasWarning(report, "policy_violation: example.com/shop/order imports package outside allow list example.com/payments/gateway") {
		t.Fatalf("warnings = %#v, missing fixture policy violation warning", report.Warnings)
	}
	if !reportHasQuery(report, "query:complexity:domain:example.com/shop/order", "unknown") {
		t.Fatalf("queries = %#v, missing fixture domain complexity incomplete query", report.Queries)
	}
}

func TestSelfDogfoodGraphUsesBoundedWorkspaceRoot(t *testing.T) {
	graph, err := indexer.BuildGraph(indexer.Options{
		Path:           filepath.Join("..", ".."),
		WorkspaceRoot:  filepath.Join("..", ".."),
		SimulateChange: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := countKind(graph.Nodes, "module"); got != 1 {
		t.Fatalf("module count = %d, want 1 for bounded self-analysis", got)
	}
	if nodeByID(graph, "module:github.com/ravinsharma7/go-safedesign") == nil {
		t.Fatalf("modules = %#v, missing go-safedesign module", nodesByKind(graph.Nodes, "module"))
	}
	for _, node := range graph.Nodes {
		if node.Kind == "module" && node.ID != "module:github.com/ravinsharma7/go-safedesign" {
			t.Fatalf("unexpected module from outside bounded workspace: %#v", node)
		}
	}
}

func TestSelfDogfoodGraphEmitsPolicyAndComplexityFacts(t *testing.T) {
	graph, err := indexer.BuildGraph(indexer.Options{
		Path:           filepath.Join("..", ".."),
		WorkspaceRoot:  filepath.Join("..", ".."),
		SimulateChange: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Metrics) == 0 {
		t.Fatal("self dogfood graph emitted no complexity metrics")
	}
	if len(graph.PolicyResults) == 0 {
		t.Fatal("self dogfood graph emitted no policy results from root safedesign.config.json")
	}
	for _, result := range graph.PolicyResults {
		if result.Status != "pass" {
			t.Fatalf("self dogfood policy result = %#v, want all pass", result)
		}
	}
	if run := runByStage(graph, string(pipeline.StagePolicyEvaluation)); run == nil || run.ConfigurationHash == "" {
		t.Fatalf("policy run = %#v, want dogfood config hash", run)
	}
	if run := runByStage(graph, string(pipeline.StageComplexityMetrics)); run == nil || run.ConfigurationHash == "" {
		t.Fatalf("complexity run = %#v, want dogfood config hash", run)
	}
}

func TestFixtureLanguageZoneCandidatesSatisfyQualityInvariants(t *testing.T) {
	graph := buildFixtureGraph(t)
	candidates := languageZoneCandidates(graph)
	if len(candidates) == 0 {
		t.Fatal("fixture emitted no language-zone candidates")
	}
	assertLanguageZoneCandidateQuality(t, graph, candidates)
	if !hasCandidateForPackage(candidates, "example.com/shop/order") {
		t.Fatalf("candidates = %#v, missing order package candidate", candidates)
	}
	if !hasCandidateForPackage(candidates, "example.com/shop/paymentclient") {
		t.Fatalf("candidates = %#v, missing paymentclient package candidate", candidates)
	}
}

func TestSelfDogfoodLanguageZoneCandidatesSatisfyQualityInvariants(t *testing.T) {
	graph, err := indexer.BuildGraph(indexer.Options{
		Path:           filepath.Join("..", ".."),
		WorkspaceRoot:  filepath.Join("..", ".."),
		SimulateChange: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	candidates := languageZoneCandidates(graph)
	if len(candidates) == 0 {
		return
	}
	assertLanguageZoneCandidateQuality(t, graph, candidates)
}

func TestSelfDogfoodProblemsReportHasNoPolicyFailures(t *testing.T) {
	graph, err := indexer.BuildGraph(indexer.Options{
		Path:           filepath.Join("..", ".."),
		WorkspaceRoot:  filepath.Join("..", ".."),
		SimulateChange: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	report := buildProblemsReport(graph, 50)
	if len(report.PolicyResults) != 0 {
		t.Fatalf("policy results = %#v, want no self dogfood policy problems", report.PolicyResults)
	}
	if len(report.Diagnostics) > report.Summary.Diagnostics || len(report.Metrics) > report.Summary.Metrics || len(report.Queries) > report.Summary.Queries || len(report.Runs) > report.Summary.Runs {
		t.Fatalf("report = %#v, list lengths should not exceed summaries", report)
	}
}

func TestPrototypeGoldenSummary(t *testing.T) {
	graph := buildFixtureGraph(t)
	got := summarizeGraph(graph)
	wantPath := filepath.Join("..", "..", "test", "golden", "prototype_summary.json.golden")
	wantBytes, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	var want graphSummary
	if err := json.Unmarshal(wantBytes, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		gotBytes, _ := json.MarshalIndent(got, "", "  ")
		wantPretty, _ := json.MarshalIndent(want, "", "  ")
		t.Fatalf("summary mismatch\nwant:\n%s\n\ngot:\n%s", wantPretty, gotBytes)
	}
}

func buildFixtureGraph(t *testing.T) core.Graph {
	t.Helper()
	graph, err := indexer.BuildGraph(indexer.Options{
		Path:           filepath.Join("..", "..", "testdata", "workspace", "shop"),
		WorkspaceRoot:  filepath.Join("..", "..", "testdata", "workspace"),
		SimulateChange: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return graph
}

type graphSummary struct {
	NodeKinds           map[string]int    `json:"nodeKinds"`
	EdgeKinds           map[string]int    `json:"edgeKinds"`
	SourceRecordKinds   map[string]int    `json:"sourceRecordKinds"`
	Placeholders        []string          `json:"placeholders"`
	RuntimeMarkers      []string          `json:"runtimeMarkers"`
	PackageTrust        map[string]string `json:"packageTrust"`
	QueryStatuses       map[string]string `json:"queryStatuses"`
	PathStatuses        map[string]string `json:"pathStatuses"`
	FreshnessStatuses   map[string]string `json:"freshnessStatuses"`
	ReconciledImportIDs []string          `json:"reconciledImportIds"`
}

func summarizeGraph(graph core.Graph) graphSummary {
	summary := graphSummary{
		NodeKinds:         map[string]int{},
		EdgeKinds:         map[string]int{},
		SourceRecordKinds: map[string]int{},
		PackageTrust:      map[string]string{},
		QueryStatuses:     map[string]string{},
		PathStatuses:      map[string]string{},
		FreshnessStatuses: map[string]string{},
	}
	for _, node := range graph.Nodes {
		summary.NodeKinds[node.Kind]++
		switch node.Kind {
		case "placeholder":
			summary.Placeholders = append(summary.Placeholders, node.ID)
		case "runtime_marker":
			summary.RuntimeMarkers = append(summary.RuntimeMarkers, node.Name)
		case "package":
			summary.PackageTrust[node.ID] = string(node.TrustLevel)
		}
	}
	for _, edge := range graph.Edges {
		summary.EdgeKinds[edge.Kind]++
		if edge.Reason == "placeholder_reconciled_to_real_package" {
			summary.ReconciledImportIDs = append(summary.ReconciledImportIDs, edge.ID)
		}
	}
	for _, record := range graph.SourceRecords {
		summary.SourceRecordKinds[record.Kind]++
	}
	for _, query := range graph.Queries {
		summary.QueryStatuses[query.ID] = query.Status
	}
	for _, path := range graph.PathJobs {
		summary.PathStatuses[path.ID] = path.Status
	}
	for _, freshness := range graph.Freshness {
		summary.FreshnessStatuses[freshness.FactID] = freshness.Status
	}
	sort.Strings(summary.Placeholders)
	sort.Strings(summary.RuntimeMarkers)
	sort.Strings(summary.ReconciledImportIDs)
	return summary
}

func countKind(nodes []core.Node, kind string) int {
	count := 0
	for _, node := range nodes {
		if node.Kind == kind {
			count++
		}
	}
	return count
}

func nodeByID(graph core.Graph, id string) *core.Node {
	for i := range graph.Nodes {
		if graph.Nodes[i].ID == id {
			return &graph.Nodes[i]
		}
	}
	return nil
}

func edgeByID(graph core.Graph, id string) *core.Edge {
	for i := range graph.Edges {
		if graph.Edges[i].ID == id {
			return &graph.Edges[i]
		}
	}
	return nil
}

func namesByKind(nodes []core.Node, kind string) []string {
	var names []string
	for _, node := range nodes {
		if node.Kind == kind {
			names = append(names, node.Name)
		}
	}
	sort.Strings(names)
	return names
}

func nodesByKind(nodes []core.Node, kind string) []core.Node {
	var out []core.Node
	for _, node := range nodes {
		if node.Kind == kind {
			out = append(out, node)
		}
	}
	return out
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func hasRun(graph core.Graph, stage string) bool {
	for _, run := range graph.Runs {
		if run.Stage == stage {
			return true
		}
	}
	return false
}

func runByStage(graph core.Graph, stage string) *core.RunRecord {
	for i := range graph.Runs {
		if graph.Runs[i].Stage == stage {
			return &graph.Runs[i]
		}
	}
	return nil
}

func runByAnalyzer(graph core.Graph, analyzerID string) *core.RunRecord {
	for i := range graph.Runs {
		if graph.Runs[i].AnalyzerID == analyzerID {
			return &graph.Runs[i]
		}
	}
	return nil
}

func hasObservation(graph core.Graph, name string) bool {
	for _, observation := range graph.Observations {
		if observation.Name == name && len(observation.Evidence) > 0 {
			return true
		}
	}
	return false
}

func languageZoneCandidates(graph core.Graph) []core.Observation {
	var out []core.Observation
	for _, observation := range graph.Observations {
		if observation.Name == core.ObservationNameLanguageZoneCandidate {
			out = append(out, observation)
		}
	}
	return out
}

func observationByID(graph core.Graph, id string) *core.Observation {
	for i := range graph.Observations {
		if graph.Observations[i].ID == id {
			return &graph.Observations[i]
		}
	}
	return nil
}

func hasCandidateForPackage(candidates []core.Observation, packagePath string) bool {
	for _, candidate := range candidates {
		if candidate.Attributes["packagePath"] == packagePath {
			return true
		}
	}
	return false
}

func candidateTerms(candidate core.Observation) []string {
	raw := candidate.Attributes["terms"]
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var terms []string
	for _, part := range parts {
		term := strings.TrimSpace(part)
		if term != "" {
			terms = append(terms, term)
		}
	}
	return terms
}

func assertLanguageZoneCandidateQuality(t *testing.T, graph core.Graph, candidates []core.Observation) {
	t.Helper()
	stopTerms := map[string]bool{
		"com": true, "org": true, "net": true, "io": true, "github": true, "gitlab": true,
		"example": true, "internal": true, "testdata": true, "workspace": true, "go": true,
	}
	for _, candidate := range candidates {
		if candidate.Attributes["packagePath"] == "" {
			t.Fatalf("candidate = %#v, missing packagePath", candidate)
		}
		terms := candidateTerms(candidate)
		if len(terms) == 0 {
			t.Fatalf("candidate = %#v, missing terms", candidate)
		}
		termCount, err := strconv.Atoi(candidate.Attributes["termCount"])
		if err != nil || termCount != len(terms) {
			t.Fatalf("candidate = %#v, termCount should match terms", candidate)
		}
		cooccurrenceCount, err := strconv.Atoi(candidate.Attributes["cooccurrenceCount"])
		if err != nil || cooccurrenceCount <= 0 {
			t.Fatalf("candidate = %#v, cooccurrenceCount should be positive", candidate)
		}
		if len(candidate.Evidence) == 0 {
			t.Fatalf("candidate = %#v, missing evidence", candidate)
		}
		if core.IsPlaceholderID(candidate.TargetID) {
			t.Fatalf("candidate = %#v, target must not be placeholder-backed", candidate)
		}
		for _, term := range terms {
			if stopTerms[term] {
				t.Fatalf("candidate = %#v, contains stop term %q", candidate, term)
			}
		}
		for _, evidenceID := range candidate.Evidence {
			evidence := observationByID(graph, evidenceID)
			if evidence == nil {
				t.Fatalf("candidate = %#v, missing evidence observation %s", candidate, evidenceID)
			}
			if evidence.Name == core.ObservationNameVocabularyIncompleteDependency {
				t.Fatalf("candidate = %#v, uses incomplete dependency evidence %s", candidate, evidenceID)
			}
		}
	}
}

func hasDiagnostic(graph core.Graph, status, reason string) bool {
	for _, diagnostic := range graph.Diagnostics {
		if diagnostic.Status == status && diagnostic.Reason == reason {
			return true
		}
	}
	return false
}

func hasPolicyResult(graph core.Graph, ruleID, status, subject string) bool {
	for _, result := range graph.PolicyResults {
		if result.RuleID == ruleID && result.Status == status && result.Subject == subject && len(result.Evidence) > 0 {
			return true
		}
	}
	return false
}

func hasMetric(graph core.Graph, name, status, subject string) bool {
	for _, metric := range graph.Metrics {
		if metric.Name == name && metric.Status == status && metric.Subject == subject && len(metric.Evidence) > 0 {
			return true
		}
	}
	return false
}

func hasMetricName(graph core.Graph, name string) bool {
	for _, metric := range graph.Metrics {
		if metric.Name == name {
			return true
		}
	}
	return false
}

func hasLabel(graph core.Graph, name, value, targetID string) bool {
	for _, label := range graph.Labels {
		if label.Name == name && label.Value == value && label.TargetID == targetID && label.RunID != "" {
			return true
		}
	}
	return false
}

func hasMetricValue(graph core.Graph, name, subject string, value int) bool {
	for _, metric := range graph.Metrics {
		if metric.Name == name && metric.Subject == subject && metric.Value == value && len(metric.Evidence) > 0 {
			return true
		}
	}
	return false
}

func reportHasPolicyResult(report problemsReport, status, subject string) bool {
	for _, result := range report.PolicyResults {
		if result.Status == status && result.Subject == subject {
			return true
		}
	}
	return false
}

func reportHasMetric(report problemsReport, status, subject string) bool {
	for _, metric := range report.Metrics {
		if metric.Status == status && metric.Subject == subject {
			return true
		}
	}
	return false
}

func reportHasQuery(report problemsReport, id, status string) bool {
	for _, query := range report.Queries {
		if query.ID == id && query.Status == status {
			return true
		}
	}
	return false
}

func reportHasWarning(report problemsReport, reason string) bool {
	for _, warning := range report.Warnings {
		if warning.Reason == reason && len(warning.Evidence) > 0 {
			return true
		}
	}
	return false
}

func hasQuery(graph core.Graph, id, status, reason string) bool {
	for _, query := range graph.Queries {
		if query.ID == id && query.Status == status && query.Reason == reason && query.ProofStatus == proofStatusForTest(status) && len(query.Evidence) > 0 {
			return true
		}
	}
	return false
}

func hasWarning(graph core.Graph, status, reason, edgeID string) bool {
	for _, warning := range graph.Warnings {
		if warning.Reason != reason || warning.AffectedEdgeID != edgeID || warning.RunID == "" {
			continue
		}
		policy := policyResultByID(graph, warning.Evidence[0])
		if policy != nil && policy.Status == status && contains(warning.Evidence, edgeID) {
			return true
		}
	}
	return false
}

func policyResultByID(graph core.Graph, id string) *core.PolicyResult {
	for i := range graph.PolicyResults {
		if graph.PolicyResults[i].ID == id {
			return &graph.PolicyResults[i]
		}
	}
	return nil
}

func proofStatusForTest(status string) string {
	if status == "unknown" || status == "analysis_error" {
		return status
	}
	return "exists"
}

func hasFreshness(graph core.Graph, factID, status string) bool {
	for _, freshness := range graph.Freshness {
		if freshness.FactID == factID && freshness.Status == status {
			return true
		}
	}
	return false
}

func hasSourceRecord(graph core.Graph, kind, path string) bool {
	for _, record := range graph.SourceRecords {
		if record.Kind == kind && record.Path == path && record.RunID != "" {
			return true
		}
	}
	return false
}

func containsJSONFragment(data []byte, fragment string) bool {
	return strings.Contains(string(data), fragment)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
