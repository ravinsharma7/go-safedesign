package complexity

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"sort"

	"go-safedesign/internal/core"
	"go-safedesign/internal/pipeline"
)

const (
	ID      = "complexity"
	Version = "prototype-1"
)

const (
	CyclomaticMetricName         = "cyclomatic_complexity"
	DecisionCountMetricName      = "decision_count"
	RuntimeMarkerCountMetricName = "runtime_marker_count"
	MetricUnit                   = "count"
	defaultWarningThreshold      = 10
)

const MetricName = CyclomaticMetricName

type MetricDefinition struct {
	Name         string
	Unit         string
	Description  string
	Scope        string
	DefaultRules countingRules
	Limitations  []string
}

func CyclomaticMetricDefinition() MetricDefinition {
	return MetricDefinition{
		Name:         CyclomaticMetricName,
		Unit:         MetricUnit,
		Description:  "Function-level cyclomatic complexity with a baseline of 1 plus configured decision increments.",
		Scope:        "function_or_method",
		DefaultRules: defaultCountingRules(),
		Limitations: []string{
			"does not account for unsaved overlays",
			"does not emit cognitive complexity",
			"domain query scope is package plus direct imports",
		},
	}
}

func DecisionCountMetricDefinition() MetricDefinition {
	return MetricDefinition{
		Name:         DecisionCountMetricName,
		Unit:         MetricUnit,
		Description:  "Function-level decision count using the same configured decision increments as cyclomatic complexity.",
		Scope:        "function_or_method",
		DefaultRules: defaultCountingRules(),
		Limitations: []string{
			"does not account for unsaved overlays",
			"does not emit package or module aggregate metrics",
		},
	}
}

func RuntimeMarkerCountMetricDefinition() MetricDefinition {
	return MetricDefinition{
		Name:        RuntimeMarkerCountMetricName,
		Unit:        MetricUnit,
		Description: "Function-level runtime marker count for go, defer, channel send, and channel receive operations.",
		Scope:       "function_or_method",
		Limitations: []string{
			"does not emit package or module aggregate metrics",
			"does not currently apply thresholds or warnings",
		},
	}
}

type Config struct {
	Cyclomatic CyclomaticConfig `json:"cyclomatic"`
}

type CyclomaticConfig struct {
	WarningThreshold          int   `json:"warningThreshold"`
	CountBooleanOperators     *bool `json:"countBooleanOperators,omitempty"`
	CountSwitchDefaultClauses *bool `json:"countSwitchDefaultClauses,omitempty"`
	CountSelectDefaultClauses *bool `json:"countSelectDefaultClauses,omitempty"`
	CountGotoStatements       *bool `json:"countGotoStatements,omitempty"`
}

type countingRules struct {
	WarningThreshold          int
	CountBooleanOperators     bool
	CountSwitchDefaultClauses bool
	CountSelectDefaultClauses bool
	CountGotoStatements       bool
}

type Analyzer struct{}

func (Analyzer) Metadata() pipeline.AnalyzerMetadata {
	return pipeline.AnalyzerMetadata{
		ID:                    ID,
		Version:               Version,
		Stage:                 pipeline.StageComplexityMetrics,
		InputFactKinds:        []string{core.NodeKindFunction, core.NodeKindMethod},
		MinimumRequiredTrust:  core.TrustSyntaxObserved,
		MaximumEmittedTrust:   core.TrustSyntaxObserved,
		EmittedFactKinds:      []string{core.FactKindMetric, core.FactKindDiagnostic},
		ConfigurationSection:  "complexity",
		FailureMode:           pipeline.FailureModePartial,
		IncompleteInputPolicy: pipeline.IncompleteInputRequireComplete,
	}
}

func Metadata() pipeline.AnalyzerMetadata {
	return Analyzer{}.Metadata()
}

func (a Analyzer) Run(context pipeline.GraphContext) (pipeline.AnalyzerResult, error) {
	rules, err := parseCountingRules(context.Configuration)
	if err != nil {
		return pipeline.AnalyzerResult{}, err
	}
	metrics, diagnostics := a.evaluate(context.Graph, context.SyntaxSnapshots, rules, context.ConfigurationHash)
	return pipeline.AnalyzerResult{Metrics: metrics, Diagnostics: diagnostics}, nil
}

func parseCountingRules(raw []byte) (countingRules, error) {
	rules := defaultCountingRules()
	if len(raw) == 0 {
		return rules, nil
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return countingRules{}, err
	}
	if cfg.Cyclomatic.WarningThreshold > 0 {
		rules.WarningThreshold = cfg.Cyclomatic.WarningThreshold
	}
	if cfg.Cyclomatic.CountBooleanOperators != nil {
		rules.CountBooleanOperators = *cfg.Cyclomatic.CountBooleanOperators
	}
	if cfg.Cyclomatic.CountSwitchDefaultClauses != nil {
		rules.CountSwitchDefaultClauses = *cfg.Cyclomatic.CountSwitchDefaultClauses
	}
	if cfg.Cyclomatic.CountSelectDefaultClauses != nil {
		rules.CountSelectDefaultClauses = *cfg.Cyclomatic.CountSelectDefaultClauses
	}
	if cfg.Cyclomatic.CountGotoStatements != nil {
		rules.CountGotoStatements = *cfg.Cyclomatic.CountGotoStatements
	}
	return rules, nil
}

func defaultCountingRules() countingRules {
	return countingRules{
		WarningThreshold:          defaultWarningThreshold,
		CountBooleanOperators:     true,
		CountSwitchDefaultClauses: true,
		CountSelectDefaultClauses: true,
		CountGotoStatements:       false,
	}
}

func (Analyzer) evaluate(graph core.Graph, snapshots map[string]pipeline.SyntaxSnapshot, rules countingRules, configHash string) ([]core.Metric, []core.Diagnostic) {
	functions := functionNodes(graph)
	byFile := map[string][]core.Node{}
	for _, node := range functions {
		if node.SourceFile != "" {
			byFile[node.SourceFile] = append(byFile[node.SourceFile], node)
		}
	}

	var metrics []core.Metric
	var diagnostics []core.Diagnostic
	for sourceFile, nodes := range byFile {
		snapshot, ok := snapshots[sourceFile]
		if !ok || snapshot.File == nil {
			diagnostics = append(diagnostics, parseDiagnostic(sourceFile, "", "syntax snapshot missing for complexity analysis"))
			continue
		}
		known := map[string]core.Node{}
		for _, node := range nodes {
			known[node.ID] = node
		}
		for _, decl := range snapshot.File.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			id := functionID(packagePath(nodes), fn)
			node, ok := known[id]
			if !ok {
				continue
			}
			decisions := decisionCount(fn, rules)
			cyclomatic := cyclomaticComplexity(decisions)
			cyclomaticMetric := cyclomaticMetricFor(node, cyclomatic, rules.WarningThreshold, configHash)
			metrics = append(metrics, cyclomaticMetric, decisionCountMetricFor(node, decisions, configHash), runtimeMarkerCountMetricFor(node, runtimeMarkerCount(fn), configHash))
			if cyclomaticMetric.Status == core.StatusWarning {
				diagnostics = append(diagnostics, diagnosticFor(cyclomaticMetric))
			}
		}
	}

	sort.Slice(metrics, func(i, j int) bool { return metrics[i].ID < metrics[j].ID })
	sort.Slice(diagnostics, func(i, j int) bool {
		if diagnostics[i].Source == diagnostics[j].Source {
			return diagnostics[i].Reason < diagnostics[j].Reason
		}
		return diagnostics[i].Source < diagnostics[j].Source
	})
	return metrics, diagnostics
}

func functionNodes(graph core.Graph) []core.Node {
	var nodes []core.Node
	for _, node := range graph.Nodes {
		if (node.Kind == core.NodeKindFunction || node.Kind == core.NodeKindMethod) && node.PackagePath != "" {
			nodes = append(nodes, node)
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	return nodes
}

func packagePath(nodes []core.Node) string {
	if len(nodes) == 0 {
		return ""
	}
	return nodes[0].PackagePath
}

func functionID(pkgPath string, fn *ast.FuncDecl) string {
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		return "method:" + pkgPath + "." + receiverName(fn.Recv.List[0].Type) + "." + fn.Name.Name
	}
	return "function:" + pkgPath + "." + fn.Name.Name
}

func receiverName(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.StarExpr:
		return "*" + receiverName(x.X)
	default:
		return "receiver"
	}
}

func complexity(fn *ast.FuncDecl, rules countingRules) int {
	return cyclomaticComplexity(decisionCount(fn, rules))
}

func cyclomaticComplexity(decisions int) int {
	return 1 + decisions
}

func decisionCount(fn *ast.FuncDecl, rules countingRules) int {
	value := 0
	if fn.Body == nil {
		return value
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt:
			value++
		case *ast.CaseClause:
			if len(x.List) > 0 || rules.CountSwitchDefaultClauses {
				value++
			}
		case *ast.CommClause:
			if x.Comm != nil || rules.CountSelectDefaultClauses {
				value++
			}
		case *ast.BranchStmt:
			if x.Tok == token.GOTO && rules.CountGotoStatements {
				value++
			}
		case *ast.BinaryExpr:
			if rules.CountBooleanOperators && (x.Op == token.LAND || x.Op == token.LOR) {
				value++
			}
		}
		return true
	})
	return value
}

func runtimeMarkerCount(fn *ast.FuncDecl) int {
	value := 0
	if fn.Body == nil {
		return value
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GoStmt, *ast.DeferStmt, *ast.SendStmt:
			value++
		case *ast.UnaryExpr:
			if x.Op == token.ARROW {
				value++
			}
		}
		return true
	})
	return value
}

func cyclomaticMetricFor(node core.Node, value, threshold int, configHash string) core.Metric {
	status := core.StatusPass
	reason := fmt.Sprintf("cyclomatic complexity %d is within threshold %d", value, threshold)
	if value > threshold {
		status = core.StatusWarning
		reason = fmt.Sprintf("cyclomatic complexity %d exceeds threshold %d", value, threshold)
	}
	return core.Metric{
		ID:                "metric:" + CyclomaticMetricName + ":" + node.ID,
		Kind:              core.FactKindMetric,
		Name:              CyclomaticMetricName,
		Value:             value,
		Unit:              MetricUnit,
		Scope:             node.PackagePath,
		Subject:           node.ID,
		AnalyzerID:        ID,
		Stage:             string(pipeline.StageComplexityMetrics),
		Status:            status,
		Threshold:         threshold,
		Reason:            reason,
		Evidence:          []string{node.ID},
		TrustLevel:        core.TrustSyntaxObserved,
		ConfigurationHash: configHash,
		SourceFile:        node.SourceFile,
		LineRange:         node.LineRange,
	}
}

func metricFor(node core.Node, value, threshold int, configHash string) core.Metric {
	return cyclomaticMetricFor(node, value, threshold, configHash)
}

func decisionCountMetricFor(node core.Node, value int, configHash string) core.Metric {
	return core.Metric{
		ID:                "metric:" + DecisionCountMetricName + ":" + node.ID,
		Kind:              core.FactKindMetric,
		Name:              DecisionCountMetricName,
		Value:             value,
		Unit:              MetricUnit,
		Scope:             node.PackagePath,
		Subject:           node.ID,
		AnalyzerID:        ID,
		Stage:             string(pipeline.StageComplexityMetrics),
		Status:            core.StatusPass,
		Reason:            fmt.Sprintf("decision count %d", value),
		Evidence:          []string{node.ID},
		TrustLevel:        core.TrustSyntaxObserved,
		ConfigurationHash: configHash,
		SourceFile:        node.SourceFile,
		LineRange:         node.LineRange,
	}
}

func runtimeMarkerCountMetricFor(node core.Node, value int, configHash string) core.Metric {
	return core.Metric{
		ID:                "metric:" + RuntimeMarkerCountMetricName + ":" + node.ID,
		Kind:              core.FactKindMetric,
		Name:              RuntimeMarkerCountMetricName,
		Value:             value,
		Unit:              MetricUnit,
		Scope:             node.PackagePath,
		Subject:           node.ID,
		AnalyzerID:        ID,
		Stage:             string(pipeline.StageComplexityMetrics),
		Status:            core.StatusPass,
		Reason:            fmt.Sprintf("runtime marker count %d", value),
		Evidence:          []string{node.ID},
		TrustLevel:        core.TrustSyntaxObserved,
		ConfigurationHash: configHash,
		SourceFile:        node.SourceFile,
		LineRange:         node.LineRange,
	}
}

func diagnosticFor(metric core.Metric) core.Diagnostic {
	return core.Diagnostic{
		Level:      core.StatusWarning,
		Source:     "metric:" + metric.Scope,
		Reason:     metric.Reason,
		Stage:      metric.Stage,
		AnalyzerID: metric.AnalyzerID,
		Status:     metric.Status,
		NodeID:     metric.Subject,
		SourceFile: metric.SourceFile,
		LineRange:  metric.LineRange,
		TrustLevel: metric.TrustLevel,
		Evidence:   []string{metric.ID},
	}
}

func parseDiagnostic(sourceFile, lineRange, reason string) core.Diagnostic {
	return core.Diagnostic{
		Level:      "error",
		Source:     sourceFile,
		Reason:     reason,
		Stage:      string(pipeline.StageComplexityMetrics),
		AnalyzerID: ID,
		Status:     core.StatusAnalysisError,
		SourceFile: sourceFile,
		LineRange:  lineRange,
		TrustLevel: core.TrustSyntaxObserved,
	}
}
