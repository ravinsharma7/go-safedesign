package indexer

import (
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"time"

	"github.com/ravinsharma7/go-safedesign/internal/config"
	"github.com/ravinsharma7/go-safedesign/internal/core"
	"github.com/ravinsharma7/go-safedesign/internal/pipeline"
)

type Graph = core.Graph
type Node = core.Node
type Edge = core.Edge
type Diagnostic = core.Diagnostic
type Freshness = core.Freshness
type QueryResult = core.QueryResult
type PathJob = core.PathJob
type PolicyResult = core.PolicyResult
type Metric = core.Metric
type TrustLevel = core.TrustLevel

const (
	TrustSyntaxObserved = core.TrustSyntaxObserved
	TrustPackageLoaded  = core.TrustPackageLoaded
	TrustTypeResolved   = core.TrustTypeResolved
)

type Options struct {
	Path              string
	FixtureRoot       string
	WorkspaceRoot     string
	SourceBase        string
	ConfigPath        string
	PolicyConfigPath  string
	SimulateChange    bool
	DisablePolicyEval bool
	DisableComplexity bool
	AnalyzerExecution AnalyzerExecutionOptions
}

type moduleInfo struct {
	Path    string
	Dir     string
	Reasons []string
}

type graphBuilder struct {
	root            string
	entryPath       string
	workspaceRoot   string
	sourceBase      string
	modules         []moduleInfo
	fset            *token.FileSet
	nodes           map[string]Node
	edges           map[string]Edge
	sourceRecords   []core.SourceRecord
	observations    []core.Observation
	labels          []core.Label
	warnings        []core.Warning
	queries         []QueryResult
	pathJobs        []PathJob
	policyResults   []PolicyResult
	metrics         []Metric
	freshness       []Freshness
	diagnostics     []Diagnostic
	runs            []core.RunRecord
	config          config.Document
	configPath      string
	syntaxSnapshots map[string]pipeline.SyntaxSnapshot
	currentRun      core.RunRecord
}

type Project struct {
	Root          string
	EntryPath     string
	WorkspaceRoot string
	SourceBase    string
}

func ResolveProject(opts Options) (Project, error) {
	entry := opts.Path
	if entry == "" {
		entry = opts.FixtureRoot
	}
	if entry == "" {
		return Project{}, fmt.Errorf("missing --path")
	}
	entryAbs, err := filepath.Abs(entry)
	if err != nil {
		return Project{}, err
	}
	info, err := os.Stat(entryAbs)
	if err != nil {
		return Project{}, err
	}
	searchDir := entryAbs
	if !info.IsDir() {
		searchDir = filepath.Dir(entryAbs)
	}
	root, err := nearestModuleRoot(searchDir)
	if err != nil {
		return Project{}, err
	}
	workspaceRoot := root
	if opts.WorkspaceRoot != "" {
		workspaceRoot, err = filepath.Abs(opts.WorkspaceRoot)
		if err != nil {
			return Project{}, err
		}
	}
	sourceBase := workspaceRoot
	if opts.SourceBase != "" {
		sourceBase, err = filepath.Abs(opts.SourceBase)
		if err != nil {
			return Project{}, err
		}
	}
	return Project{Root: filepath.Clean(root), EntryPath: filepath.Clean(entryAbs), WorkspaceRoot: filepath.Clean(workspaceRoot), SourceBase: filepath.Clean(sourceBase)}, nil
}

func nearestModuleRoot(start string) (string, error) {
	dir := filepath.Clean(start)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod found at or above %s", start)
		}
		dir = parent
	}
}

func BuildGraph(opts Options) (Graph, error) {
	project, err := ResolveProject(opts)
	if err != nil {
		return Graph{}, err
	}

	b := &graphBuilder{
		root:            project.Root,
		entryPath:       project.EntryPath,
		workspaceRoot:   project.WorkspaceRoot,
		sourceBase:      project.SourceBase,
		fset:            token.NewFileSet(),
		nodes:           map[string]Node{},
		edges:           map[string]Edge{},
		syntaxSnapshots: map[string]pipeline.SyntaxSnapshot{},
	}
	configPath := config.ResolvePath(project.Root, opts.ConfigPath)
	if opts.PolicyConfigPath != "" {
		configPath = opts.PolicyConfigPath
	}
	b.configPath = configPath
	if err := b.runStage(pipeline.StageSourceDiscovery, b.discoverSources); err != nil {
		return b.graph(), err
	}
	b.config, _, err = config.Load(configPath)
	if err != nil {
		return Graph{}, err
	}

	if err := b.runStage(pipeline.StageModuleExtraction, b.indexModules); err != nil {
		return b.graph(), err
	}
	if err := b.runStage(pipeline.StageSyntaxExtraction, b.indexGoFiles); err != nil {
		return b.graph(), err
	}
	if err := b.runStage(pipeline.StageBaseGraphAssembly, func() error {
		b.reconcilePlaceholders()
		return nil
	}); err != nil {
		return b.graph(), err
	}
	if err := b.runStage(pipeline.StagePackageLoading, func() error {
		b.loadPackages()
		return nil
	}); err != nil {
		return b.graph(), err
	}
	execution := opts.AnalyzerExecution
	if opts.DisableComplexity {
		execution.Skip = append(execution.Skip, AnalyzerIDComplexity)
	}
	if opts.DisablePolicyEval {
		execution.Skip = append(execution.Skip, AnalyzerIDDependencyPolicy)
	}
	analyzers, err := planAnalyzers(execution)
	if err != nil {
		return b.graph(), err
	}
	for _, analyzer := range analyzers {
		b.runAnalyzer(analyzer)
	}
	if err := b.runStage(pipeline.StageQueryMaterialization, func() error {
		b.addQueriesAndPathJobs()
		if opts.SimulateChange {
			b.simulateFreshnessChange()
		}
		return nil
	}); err != nil {
		return b.graph(), err
	}

	return b.graph(), nil
}

func (b *graphBuilder) runStage(stage pipeline.Stage, fn func() error) error {
	beforeFacts := b.factCount()
	beforeDiagnostics := len(b.diagnostics)
	started := time.Now().UTC()
	run := core.RunRecord{
		RunID:           pipeline.RunID(string(stage), "indexer."+string(stage), started),
		AnalyzerID:      "indexer." + string(stage),
		AnalyzerVersion: core.ExtractorVersion,
		Stage:           string(stage),
		StartedAt:       started.Format(time.RFC3339Nano),
	}
	b.currentRun = run
	err := fn()
	finished := time.Now().UTC()
	status := core.StatusCompleted
	if err != nil {
		status = core.StatusAnalysisError
	} else if len(b.diagnostics) > beforeDiagnostics {
		status = core.StatusPartial
	}
	var diagnostics []string
	for _, diagnostic := range b.diagnostics[beforeDiagnostics:] {
		diagnostics = append(diagnostics, diagnostic.Source+": "+diagnostic.Reason)
	}
	emittedFacts := b.factCount() - beforeFacts
	if emittedFacts < 0 {
		emittedFacts = 0
	}
	run.FinishedAt = finished.Format(time.RFC3339Nano)
	run.Status = status
	run.Diagnostics = diagnostics
	run.EmittedFacts = emittedFacts
	b.runs = append(b.runs, run)
	b.applyStageMetadata(run)
	b.currentRun = core.RunRecord{}
	return err
}

func (b *graphBuilder) runAnalyzer(analyzer pipeline.Analyzer) {
	metadata := analyzer.Metadata()
	if diagnostics := pipeline.ValidateAnalyzerMetadata(metadata); len(diagnostics) > 0 {
		started := time.Now().UTC()
		run := core.RunRecord{
			RunID:           pipeline.RunID(string(metadata.Stage), metadata.ID, started),
			AnalyzerID:      metadata.ID,
			AnalyzerVersion: metadata.Version,
			Stage:           string(metadata.Stage),
			StartedAt:       started.Format(time.RFC3339Nano),
			FinishedAt:      started.Format(time.RFC3339Nano),
			Status:          core.StatusAnalysisError,
			Diagnostics:     diagnosticMessages(diagnostics),
			EmittedFacts:    0,
		}
		for i := range diagnostics {
			diagnostics[i].FactMetadata = metadataForRun(run)
		}
		b.diagnostics = append(b.diagnostics, diagnostics...)
		b.runs = append(b.runs, run)
		return
	}
	if diagnostics := b.readinessDiagnostics(metadata); len(diagnostics) > 0 {
		started := time.Now().UTC()
		run := core.RunRecord{
			RunID:           pipeline.RunID(string(metadata.Stage), metadata.ID, started),
			AnalyzerID:      metadata.ID,
			AnalyzerVersion: metadata.Version,
			Stage:           string(metadata.Stage),
			StartedAt:       started.Format(time.RFC3339Nano),
			FinishedAt:      started.Format(time.RFC3339Nano),
			Status:          core.StatusPartial,
			Diagnostics:     diagnosticMessages(diagnostics),
			EmittedFacts:    len(diagnostics),
		}
		for i := range diagnostics {
			diagnostics[i].FactMetadata = metadataForRun(run)
		}
		b.diagnostics = append(b.diagnostics, diagnostics...)
		b.runs = append(b.runs, run)
		return
	}
	configuration := b.config.Section(metadata.ConfigurationSection)
	result, run, err := pipeline.RunAnalyzer(analyzer, pipeline.GraphContext{
		Graph:           b.graph(),
		SourceBase:      b.sourceBase,
		SyntaxSnapshots: b.syntaxSnapshots,
		Configuration:   configuration,
	})
	validationDiagnostics := pipeline.ValidateAnalyzerResult(b.graph(), metadata, result)
	if len(validationDiagnostics) > 0 {
		run.Status = core.StatusAnalysisError
		run.Diagnostics = append(run.Diagnostics, diagnosticMessages(validationDiagnostics)...)
		run.EmittedFacts = 0
		for i := range validationDiagnostics {
			validationDiagnostics[i].FactMetadata = metadataForRun(run)
		}
		b.diagnostics = append(b.diagnostics, validationDiagnostics...)
		b.runs = append(b.runs, run)
		return
	}
	b.diagnostics = append(b.diagnostics, result.Diagnostics...)
	b.policyResults = append(b.policyResults, result.PolicyResults...)
	b.metrics = append(b.metrics, result.Metrics...)
	b.observations = append(b.observations, result.Observations...)
	b.labels = append(b.labels, result.Labels...)
	b.warnings = append(b.warnings, result.Warnings...)
	for _, edge := range result.Edges {
		b.addEdge(edge)
	}
	if err != nil {
		b.diagnostics = append(b.diagnostics, Diagnostic{
			Level:      "error",
			Source:     "analyzer:" + metadata.ID,
			Reason:     err.Error(),
			Stage:      string(metadata.Stage),
			AnalyzerID: metadata.ID,
			Status:     core.StatusAnalysisError,
		})
	}
	b.runs = append(b.runs, run)
	b.applyAnalyzerMetadata(run)
}

func (b *graphBuilder) factCount() int {
	return len(b.nodes) + len(b.edges) + len(b.sourceRecords) + len(b.observations) + len(b.labels) + len(b.warnings) + len(b.queries) + len(b.pathJobs) + len(b.policyResults) + len(b.metrics) + len(b.freshness) + len(b.diagnostics)
}

func (b *graphBuilder) graph() Graph {
	nodes := make([]Node, 0, len(b.nodes))
	for _, n := range b.nodes {
		nodes = append(nodes, n)
	}
	edges := make([]Edge, 0, len(b.edges))
	for _, e := range b.edges {
		edges = append(edges, e)
	}
	sourceRecords := b.sourceRecords
	if sourceRecords == nil {
		sourceRecords = []core.SourceRecord{}
	}
	labels := b.labels
	if labels == nil {
		labels = []core.Label{}
	}
	observations := b.observations
	if observations == nil {
		observations = []core.Observation{}
	}
	warnings := b.warnings
	if warnings == nil {
		warnings = []core.Warning{}
	}
	return core.SortGraph(Graph{
		Nodes:         nodes,
		Edges:         edges,
		SourceRecords: sourceRecords,
		Observations:  observations,
		Labels:        labels,
		Warnings:      warnings,
		Queries:       b.queries,
		PathJobs:      b.pathJobs,
		PolicyResults: b.policyResults,
		Metrics:       b.metrics,
		Freshness:     b.freshness,
		Diagnostics:   b.diagnostics,
		Runs:          b.runs,
	})
}

func (b *graphBuilder) addNode(n Node) {
	if n.Extractor == "" {
		n.Extractor = core.ExtractorVersion
	}
	if n.RunID == "" {
		n.FactMetadata = b.metadataForCurrentRun()
	}
	if existing, ok := b.nodes[n.ID]; ok {
		if core.TrustRank(n.TrustLevel) > core.TrustRank(existing.TrustLevel) {
			existing.TrustLevel = n.TrustLevel
		}
		if existing.Reason == "" {
			existing.Reason = n.Reason
		}
		b.nodes[n.ID] = existing
		return
	}
	b.nodes[n.ID] = n
}

func (b *graphBuilder) addEdge(e Edge) {
	if e.Extractor == "" {
		e.Extractor = core.ExtractorVersion
	}
	if e.RunID == "" && b.currentRun.RunID != "" {
		e.FactMetadata = b.metadataForCurrentRun()
	}
	b.edges[e.ID] = e
}

func (b *graphBuilder) hasNode(id string) bool {
	_, ok := b.nodes[id]
	return ok
}
