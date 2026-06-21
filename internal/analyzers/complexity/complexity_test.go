package complexity

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"go-safedesign/internal/core"
	"go-safedesign/internal/pipeline"
)

func TestComplexityCountsBaseline(t *testing.T) {
	got := complexityForSource(t, `package p
func Simple() {}
`)
	if got != 1 {
		t.Fatalf("complexity = %d, want 1", got)
	}
}

func TestComplexityCountsBranchIncrements(t *testing.T) {
	got := complexityForSource(t, `package p
func Branchy(xs []int, ch <-chan int) {
	if len(xs) > 0 {}
	for i := 0; i < 2; i++ {}
	for range xs {}
	switch len(xs) {
	case 0:
	default:
	}
	select {
	case <-ch:
	default:
	}
}
`)
	if got != 8 {
		t.Fatalf("complexity = %d, want 8", got)
	}
}

func TestComplexityCountsBooleanOperators(t *testing.T) {
	got := complexityForSource(t, `package p
func Boolean(a, b, c bool) {
	if a && b || c {}
}
`)
	if got != 4 {
		t.Fatalf("complexity = %d, want 4", got)
	}
}

func TestCountingRulesCanDisableBooleanOperators(t *testing.T) {
	got := complexityForSourceWithConfig(t, `package p
func Boolean(a, b, c bool) {
	if a && b || c {}
}
`, `{"cyclomatic":{"countBooleanOperators":false}}`)
	if got != 2 {
		t.Fatalf("complexity = %d, want 2", got)
	}
}

func TestCountingRulesCanDisableDefaultClauses(t *testing.T) {
	got := complexityForSourceWithConfig(t, `package p
func Defaults(ch <-chan int) {
	switch {
	case true:
	default:
	}
	select {
	case <-ch:
	default:
	}
}
`, `{"cyclomatic":{"countSwitchDefaultClauses":false,"countSelectDefaultClauses":false}}`)
	if got != 3 {
		t.Fatalf("complexity = %d, want 3", got)
	}
}

func TestCountingRulesCanEnableGotoStatements(t *testing.T) {
	got := complexityForSourceWithConfig(t, `package p
func Jump(ok bool) {
	if ok {
		goto done
	}
done:
}
`, `{"cyclomatic":{"countGotoStatements":true}}`)
	if got != 3 {
		t.Fatalf("complexity = %d, want 3", got)
	}
}

func TestThresholdViolationEmitsDiagnostic(t *testing.T) {
	sourceFile := "sample.go"
	src := `package p
func Branchy(a, b bool) {
	if a && b {}
}
`
	graph := core.Graph{Nodes: []core.Node{{
		ID:          "function:example.com/p.Branchy",
		Kind:        "function",
		Name:        "Branchy",
		TrustLevel:  core.TrustSyntaxObserved,
		SourceFile:  sourceFile,
		LineRange:   "2:1-4:2",
		PackagePath: "example.com/p",
	}}}

	result, err := Analyzer{}.Run(pipeline.GraphContext{
		Graph:             graph,
		SyntaxSnapshots:   syntaxSnapshots(t, sourceFile, src),
		Configuration:     []byte(`{"cyclomatic":{"warningThreshold":2}}`),
		ConfigurationHash: "config-hash",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Metrics) != 3 {
		t.Fatalf("metrics = %#v, want cyclomatic, decision count, and runtime marker count metrics", result.Metrics)
	}
	cyclomatic := metricNamed(result.Metrics, CyclomaticMetricName)
	if cyclomatic == nil || cyclomatic.Value != 3 || cyclomatic.Status != "warning" {
		t.Fatalf("metrics = %#v, want warning cyclomatic metric with value 3", result.Metrics)
	}
	decisions := metricNamed(result.Metrics, DecisionCountMetricName)
	if decisions == nil || decisions.Value != 2 || decisions.Status != "pass" {
		t.Fatalf("metrics = %#v, want pass decision_count metric with value 2", result.Metrics)
	}
	runtimeMarkers := metricNamed(result.Metrics, RuntimeMarkerCountMetricName)
	if runtimeMarkers == nil || runtimeMarkers.Value != 0 || runtimeMarkers.Status != "pass" {
		t.Fatalf("metrics = %#v, want pass runtime_marker_count metric with value 0", result.Metrics)
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Evidence[0] != cyclomatic.ID {
		t.Fatalf("diagnostics = %#v, want diagnostic with metric evidence", result.Diagnostics)
	}
}

func TestAnalyzerEmitsMetricForMethodNode(t *testing.T) {
	sourceFile := "sample.go"
	src := `package p
type Service struct{}

func (s *Service) Handle(ok bool) {
	if ok {}
}
`
	graph := core.Graph{Nodes: []core.Node{{
		ID:          "method:example.com/p.*Service.Handle",
		Kind:        "method",
		Name:        "Handle",
		TrustLevel:  core.TrustSyntaxObserved,
		SourceFile:  sourceFile,
		LineRange:   "4:1-6:2",
		PackagePath: "example.com/p",
	}}}

	result, err := Analyzer{}.Run(pipeline.GraphContext{
		Graph:           graph,
		SyntaxSnapshots: syntaxSnapshots(t, sourceFile, src),
	})
	if err != nil {
		t.Fatal(err)
	}
	cyclomatic := metricNamed(result.Metrics, CyclomaticMetricName)
	decisions := metricNamed(result.Metrics, DecisionCountMetricName)
	runtimeMarkers := metricNamed(result.Metrics, RuntimeMarkerCountMetricName)
	if len(result.Metrics) != 3 || cyclomatic == nil || decisions == nil || runtimeMarkers == nil || cyclomatic.Subject != "method:example.com/p.*Service.Handle" || cyclomatic.Value != 2 || decisions.Value != 1 || runtimeMarkers.Value != 0 {
		t.Fatalf("metrics = %#v, want method cyclomatic=2, decision_count=1, runtime_marker_count=0", result.Metrics)
	}
}

func TestAnalyzerEmitsRuntimeMarkerCountMetric(t *testing.T) {
	sourceFile := "sample.go"
	src := `package p
func Runtime(ch chan int) {
	go func() {}()
	defer func() {}()
	ch <- 1
	<-ch
	select {
	case <-ch:
	default:
	}
}
`
	graph := core.Graph{Nodes: []core.Node{{
		ID:          "function:example.com/p.Runtime",
		Kind:        "function",
		Name:        "Runtime",
		TrustLevel:  core.TrustSyntaxObserved,
		SourceFile:  sourceFile,
		LineRange:   "2:1-11:2",
		PackagePath: "example.com/p",
	}}}

	result, err := Analyzer{}.Run(pipeline.GraphContext{
		Graph:           graph,
		SyntaxSnapshots: syntaxSnapshots(t, sourceFile, src),
	})
	if err != nil {
		t.Fatal(err)
	}
	runtimeMarkers := metricNamed(result.Metrics, RuntimeMarkerCountMetricName)
	if runtimeMarkers == nil || runtimeMarkers.Value != 5 || runtimeMarkers.Status != "pass" || runtimeMarkers.Subject != "function:example.com/p.Runtime" {
		t.Fatalf("metrics = %#v, want runtime_marker_count=5 for function", result.Metrics)
	}
}

func TestAnalyzerRequiresSyntaxSnapshot(t *testing.T) {
	graph := core.Graph{Nodes: []core.Node{{
		ID:          "function:example.com/p.Branchy",
		Kind:        "function",
		Name:        "Branchy",
		TrustLevel:  core.TrustSyntaxObserved,
		SourceFile:  "sample.go",
		LineRange:   "2:1-4:2",
		PackagePath: "example.com/p",
	}}}

	result, err := Analyzer{}.Run(pipeline.GraphContext{Graph: graph})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Metrics) != 0 {
		t.Fatalf("metrics = %#v, want none without syntax snapshot", result.Metrics)
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Status != "analysis_error" {
		t.Fatalf("diagnostics = %#v, want analysis_error for missing snapshot", result.Diagnostics)
	}
}

func TestMetadataUsesComplexityMetricsStage(t *testing.T) {
	metadata := Metadata()
	if metadata.ID != ID || metadata.Stage != pipeline.StageComplexityMetrics || metadata.IncompleteInputPolicy != pipeline.IncompleteInputRequireComplete {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func TestCyclomaticMetricDefinitionMatchesEmittedMetric(t *testing.T) {
	definition := CyclomaticMetricDefinition()
	rules := defaultCountingRules()
	if definition.Name != MetricName || definition.Unit != MetricUnit {
		t.Fatalf("definition = %#v, want exported metric name and unit", definition)
	}
	if definition.Scope != "function_or_method" {
		t.Fatalf("definition scope = %q, want function_or_method", definition.Scope)
	}
	if definition.DefaultRules != rules {
		t.Fatalf("definition default rules = %#v, want %#v", definition.DefaultRules, rules)
	}
	if len(definition.Limitations) == 0 {
		t.Fatal("definition should document current prototype limitations")
	}

	metric := metricFor(core.Node{
		ID:          "function:example.com/p.F",
		Kind:        "function",
		PackagePath: "example.com/p",
	}, 1, rules.WarningThreshold, "config-hash")
	if metric.Name != definition.Name || metric.Unit != definition.Unit {
		t.Fatalf("metric = %#v, want name and unit from definition %#v", metric, definition)
	}
}

func TestDecisionCountMetricDefinitionMatchesEmittedMetric(t *testing.T) {
	definition := DecisionCountMetricDefinition()
	rules := defaultCountingRules()
	if definition.Name != DecisionCountMetricName || definition.Unit != MetricUnit {
		t.Fatalf("definition = %#v, want exported decision count name and unit", definition)
	}
	if definition.Scope != "function_or_method" {
		t.Fatalf("definition scope = %q, want function_or_method", definition.Scope)
	}
	if definition.DefaultRules != rules {
		t.Fatalf("definition default rules = %#v, want %#v", definition.DefaultRules, rules)
	}

	metric := decisionCountMetricFor(core.Node{
		ID:          "function:example.com/p.F",
		Kind:        "function",
		PackagePath: "example.com/p",
	}, 2, "config-hash")
	if metric.Name != definition.Name || metric.Unit != definition.Unit || metric.Value != 2 {
		t.Fatalf("metric = %#v, want name and unit from definition %#v", metric, definition)
	}
}

func TestRuntimeMarkerCountMetricDefinitionMatchesEmittedMetric(t *testing.T) {
	definition := RuntimeMarkerCountMetricDefinition()
	if definition.Name != RuntimeMarkerCountMetricName || definition.Unit != MetricUnit {
		t.Fatalf("definition = %#v, want exported runtime marker count name and unit", definition)
	}
	if definition.Scope != "function_or_method" {
		t.Fatalf("definition scope = %q, want function_or_method", definition.Scope)
	}

	metric := runtimeMarkerCountMetricFor(core.Node{
		ID:          "function:example.com/p.F",
		Kind:        "function",
		PackagePath: "example.com/p",
	}, 3, "config-hash")
	if metric.Name != definition.Name || metric.Unit != definition.Unit || metric.Value != 3 || metric.Status != "pass" {
		t.Fatalf("metric = %#v, want name and unit from definition %#v", metric, definition)
	}
}

func complexityForSource(t *testing.T, src string) int {
	t.Helper()
	return complexityForSourceWithRules(t, src, defaultCountingRules())
}

func complexityForSourceWithConfig(t *testing.T, src string, config string) int {
	t.Helper()
	rules, err := parseCountingRules([]byte(config))
	if err != nil {
		t.Fatal(err)
	}
	return complexityForSourceWithRules(t, src, rules)
}

func complexityForSourceWithRules(t *testing.T, src string, rules countingRules) int {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments|parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			return complexity(fn, rules)
		}
	}
	t.Fatal("no function declaration")
	return 0
}

func TestDecisionCountUsesCyclomaticTraversalRules(t *testing.T) {
	rules, err := parseCountingRules([]byte(`{"cyclomatic":{"countBooleanOperators":false}}`))
	if err != nil {
		t.Fatal(err)
	}
	fn := firstFunc(t, `package p
func Boolean(a, b, c bool) {
	if a && b || c {}
}
`)
	if got := decisionCount(fn, rules); got != 1 {
		t.Fatalf("decision count = %d, want 1", got)
	}
	if got := complexity(fn, rules); got != 2 {
		t.Fatalf("complexity = %d, want 2", got)
	}
}

func syntaxSnapshots(t *testing.T, sourceFile, src string) map[string]pipeline.SyntaxSnapshot {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, sourceFile, src, parser.ParseComments|parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	return map[string]pipeline.SyntaxSnapshot{
		sourceFile: {File: file, SourceHash: core.HashBytes([]byte(src))},
	}
}

func firstFunc(t *testing.T, src string) *ast.FuncDecl {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments|parser.AllErrors)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			return fn
		}
	}
	t.Fatal("no function declaration")
	return nil
}

func metricNamed(metrics []core.Metric, name string) *core.Metric {
	for i := range metrics {
		if metrics[i].Name == name {
			return &metrics[i]
		}
	}
	return nil
}
