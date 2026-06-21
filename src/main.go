package main

import (
	"encoding/json"
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
	jsonOut := flag.Bool("json", false, "emit canonical JSON")
	problemsOut := flag.Bool("problems", false, "emit compact JSON containing only non-pass facts")
	problemsLimit := flag.Int("limit", 50, "maximum number of items per problems list")
	serveAddr := flag.String("serve", "", "serve browser viewer at address, for example :8080")
	configPath := flag.String("config", "", "optional shared safedesign config path; defaults to safedesign.json under project root when present")
	policyConfig := flag.String("policy-config", "", "compatibility override for dependency policy config path")
	disablePolicy := flag.Bool("disable-policy", false, "disable policy_evaluation analyzer stage")
	disableComplexity := flag.Bool("disable-complexity", false, "disable complexity_metrics analyzer stage")
	simulateChange := flag.Bool("simulate-change", false, "include a simulated changed-file freshness record")
	flag.Parse()

	entryPath := *path
	if entryPath == "" {
		entryPath = *fixture
	}
	if entryPath == "" {
		fmt.Fprintln(os.Stderr, "missing --path")
		os.Exit(2)
	}
	if !*jsonOut && !*problemsOut && *serveAddr == "" {
		fmt.Fprintln(os.Stderr, "choose --json, --problems, or --serve")
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

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if *problemsOut {
		if err := enc.Encode(buildProblemsReport(graph, *problemsLimit)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := enc.Encode(graph); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
