package pipeline

import (
	"fmt"
	"go/ast"
	"time"

	"github.com/ravinsharma7/go-safedesign/internal/core"
)

type Stage string

const (
	StageSourceDiscovery            Stage = "source_discovery"
	StageModuleExtraction           Stage = "module_extraction"
	StageSyntaxExtraction           Stage = "syntax_extraction"
	StageBaseGraphAssembly          Stage = "base_graph_assembly"
	StagePackageLoading             Stage = "package_loading"
	StageTypeResolution             Stage = "type_resolution"
	StageModuleDependencyEnrichment Stage = "module_dependency_enrichment"
	StageThirdPartyBehaviorLabeling Stage = "third_party_behavior_labeling"
	StageFrameworkExtraction        Stage = "framework_extraction"
	StageDDDClassification          Stage = "ddd_classification"
	StageComplexityMetrics          Stage = "complexity_metrics"
	StagePolicyEvaluation           Stage = "policy_evaluation"
	StageQueryMaterialization       Stage = "query_materialization"
	StageRendering                  Stage = "rendering"
)

type AnalyzerMetadata struct {
	ID                    string
	Version               string
	Stage                 Stage
	InputFactKinds        []string
	MinimumRequiredTrust  core.TrustLevel
	MaximumEmittedTrust   core.TrustLevel
	EmittedFactKinds      []string
	ConfigurationSection  string
	FailureMode           string
	IncompleteInputPolicy IncompleteInputPolicy
}

const (
	FailureModePartial = "partial"
)

type IncompleteInputPolicy string

const (
	IncompleteInputRequireComplete IncompleteInputPolicy = "require_complete"
	IncompleteInputSkipScope       IncompleteInputPolicy = "skip_incomplete_scope"
	IncompleteInputAllow           IncompleteInputPolicy = "allow_incomplete"
)

type RunRecord struct {
	RunID             string   `json:"runId,omitempty"`
	AnalyzerID        string   `json:"analyzerId"`
	AnalyzerVersion   string   `json:"analyzerVersion"`
	Stage             Stage    `json:"stage"`
	InputGraphVersion string   `json:"inputGraphVersion"`
	ConfigurationHash string   `json:"configurationHash"`
	StartedAt         string   `json:"startedAt,omitempty"`
	FinishedAt        string   `json:"finishedAt,omitempty"`
	Status            string   `json:"status"`
	Diagnostics       []string `json:"diagnostics"`
	EmittedFacts      int      `json:"emittedFacts"`
}

type Analyzer interface {
	Metadata() AnalyzerMetadata
	Run(GraphContext) (AnalyzerResult, error)
}

type GraphContext struct {
	Graph             core.Graph
	SourceBase        string
	SyntaxSnapshots   map[string]SyntaxSnapshot
	Configuration     []byte
	ConfigurationHash string
}

type SyntaxSnapshot struct {
	File       *ast.File
	SourceHash string
}

type AnalyzerResult struct {
	Diagnostics   []core.Diagnostic
	PolicyResults []core.PolicyResult
	Metrics       []core.Metric
	Observations  []core.Observation
	Labels        []core.Label
	Warnings      []core.Warning
	Edges         []core.Edge
}

func RunAnalyzer(analyzer Analyzer, context GraphContext) (AnalyzerResult, core.RunRecord, error) {
	metadata := analyzer.Metadata()
	started := time.Now().UTC()
	configHash := ""
	if len(context.Configuration) > 0 {
		configHash = core.HashBytes(context.Configuration)
	}
	context.ConfigurationHash = configHash
	result, err := analyzer.Run(context)
	finished := time.Now().UTC()
	status := core.StatusCompleted
	if err != nil {
		status = core.StatusAnalysisError
	} else if len(result.Diagnostics) > 0 {
		status = core.StatusPartial
	}
	var diagnostics []string
	for _, diagnostic := range result.Diagnostics {
		diagnostics = append(diagnostics, diagnostic.Source+": "+diagnostic.Reason)
	}
	run := core.RunRecord{
		RunID:             RunID(string(metadata.Stage), metadata.ID, started),
		AnalyzerID:        metadata.ID,
		AnalyzerVersion:   metadata.Version,
		Stage:             string(metadata.Stage),
		ConfigurationHash: configHash,
		StartedAt:         started.Format(time.RFC3339Nano),
		FinishedAt:        finished.Format(time.RFC3339Nano),
		Status:            status,
		Diagnostics:       diagnostics,
		EmittedFacts:      len(result.Diagnostics) + len(result.PolicyResults) + len(result.Metrics) + len(result.Observations) + len(result.Labels) + len(result.Warnings) + len(result.Edges),
	}
	if err != nil {
		run.Diagnostics = append(run.Diagnostics, fmt.Sprintf("%s: %v", metadata.ID, err))
	}
	return result, run, err
}

func RunID(stage, analyzerID string, started time.Time) string {
	return "run:" + stage + ":" + analyzerID + ":" + started.Format("20060102T150405.000000000Z")
}

type Registry struct {
	analyzers []AnalyzerMetadata
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) Register(metadata AnalyzerMetadata) {
	r.analyzers = append(r.analyzers, metadata)
}

func (r *Registry) Analyzers() []AnalyzerMetadata {
	out := make([]AnalyzerMetadata, len(r.analyzers))
	copy(out, r.analyzers)
	return out
}
