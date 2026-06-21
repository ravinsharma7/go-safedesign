package main

import (
	"flag"
	"fmt"
	"os"

	"go-safedesign/internal/indexer"
	"go-safedesign/internal/viewer"
)

func main() {
	path := flag.String("path", "", "project path, Go file, or directory to index")
	fixture := flag.String("fixture", "", "deprecated alias for --path")
	workspaceRoot := flag.String("workspace-root", "", "optional module discovery root; defaults to resolved project root")
	listValues := flag.Bool("list-values", false, "emit discoverable CLI values such as JSON sections, node kinds, edge kinds, observation names, and analyzer IDs")
	jsonOut := flag.Bool("json", false, "emit canonical JSON")
	jsonSections := flag.String("json-sections", "", "comma-separated graph JSON sections to emit, for example nodes,edges,observations")
	jsonNodeKinds := flag.String("json-node-kinds", "", "comma-separated node kinds to include in --json output")
	jsonEdgeKinds := flag.String("json-edge-kinds", "", "comma-separated edge kinds to include in --json output")
	jsonObservationNames := flag.String("json-observation-names", "", "comma-separated observation names to include in --json output")
	problemsOut := flag.Bool("problems", false, "emit compact JSON containing only non-pass facts")
	dddReportOut := flag.Bool("ddd-report", false, "emit compact JSON summarizing DDD evidence observations")
	statsOut := flag.Bool("stats", false, "emit compact JSON summarizing graph stats")
	outputDir := flag.String("output-dir", defaultOutputDir, "directory for JSON output files when --stdout is not set")
	stdout := flag.Bool("stdout", false, "write JSON output to stdout instead of --output-dir")
	problemsLimit := flag.Int("limit", 50, "maximum number of items per compact report list")
	scopeModule := flag.String("scope-module", "", "limit compact report output to a module path")
	scopePackage := flag.String("scope-package", "", "limit compact report output to a package path")
	scopeFile := flag.String("scope-file", "", "limit compact report output to a source file")
	analyzers := flag.String("analyzers", "", "comma-separated analyzer IDs to run with required prerequisites")
	skipAnalyzers := flag.String("skip-analyzers", "", "comma-separated analyzer IDs to skip")
	serveAddr := flag.String("serve", "", "serve browser viewer at address, for example :8080")
	configPath := flag.String("config", "", "optional shared safedesign config path; defaults to safedesign.config.json under project root when present")
	policyConfig := flag.String("policy-config", "", "compatibility override for dependency policy config path")
	disablePolicy := flag.Bool("disable-policy", false, "disable policy_evaluation analyzer stage")
	disableComplexity := flag.Bool("disable-complexity", false, "disable complexity_metrics analyzer stage")
	simulateChange := flag.Bool("simulate-change", false, "include a simulated changed-file freshness record")
	flag.Parse()

	if *listValues {
		if err := emitJSONOutput("values.json", buildValueDiscoveryReport(), *stdout, *outputDir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	entryPath := *path
	if entryPath == "" {
		entryPath = *fixture
	}
	if entryPath == "" {
		fmt.Fprintln(os.Stderr, "missing --path")
		os.Exit(2)
	}
	if !*jsonOut && !*problemsOut && !*dddReportOut && !*statsOut && *serveAddr == "" {
		fmt.Fprintln(os.Stderr, "choose --json, --problems, --ddd-report, --stats, --list-values, or --serve")
		os.Exit(2)
	}

	opts := indexer.Options{
		Path:              entryPath,
		WorkspaceRoot:     *workspaceRoot,
		ConfigPath:        *configPath,
		PolicyConfigPath:  *policyConfig,
		DisablePolicyEval: *disablePolicy,
		DisableComplexity: *disableComplexity,
		SimulateChange:    *simulateChange,
		AnalyzerExecution: indexer.AnalyzerExecutionOptions{
			Include: splitCommaList(*analyzers),
			Skip:    splitCommaList(*skipAnalyzers),
		},
	}
	graph, err := indexer.BuildGraph(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *serveAddr != "" {
		project, err := indexer.ResolveProject(opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "viewer listening on http://localhost%s\n", *serveAddr)
		if err := viewer.Serve(*serveAddr, graph, project.SourceBase); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	reportOptions := compactReportOptions{Limit: *problemsLimit, ScopeModule: *scopeModule, ScopePackage: *scopePackage, ScopeFile: *scopeFile}
	if *statsOut {
		if err := emitJSONOutput("stats.json", buildStatsReport(graph, reportOptions), *stdout, *outputDir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *dddReportOut {
		if err := emitJSONOutput("ddd-report.json", buildDDDReport(graph, reportOptions), *stdout, *outputDir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *problemsOut {
		if err := emitJSONOutput("problems.json", buildProblemsReport(graph, *problemsLimit), *stdout, *outputDir); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	jsonGraph, err := filterGraphJSON(graph, graphJSONFilterOptions{
		Sections:         splitCommaList(*jsonSections),
		NodeKinds:        splitCommaList(*jsonNodeKinds),
		EdgeKinds:        splitCommaList(*jsonEdgeKinds),
		ObservationNames: splitCommaList(*jsonObservationNames),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := emitJSONOutput("graph.json", jsonGraph, *stdout, *outputDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
