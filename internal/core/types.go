package core

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

const ExtractorVersion = "prototype-1"

type TrustLevel string

const (
	TrustSyntaxObserved TrustLevel = "syntax_observed"
	TrustPackageLoaded  TrustLevel = "package_loaded"
	TrustTypeResolved   TrustLevel = "type_resolved"
)

type Graph struct {
	Nodes         []Node         `json:"nodes"`
	Edges         []Edge         `json:"edges"`
	SourceRecords []SourceRecord `json:"sourceRecords"`
	Observations  []Observation  `json:"observations,omitempty"`
	Labels        []Label        `json:"labels"`
	Warnings      []Warning      `json:"warnings"`
	Queries       []QueryResult  `json:"queries"`
	PathJobs      []PathJob      `json:"pathJobs"`
	PolicyResults []PolicyResult `json:"policyResults,omitempty"`
	Metrics       []Metric       `json:"metrics,omitempty"`
	Freshness     []Freshness    `json:"freshness"`
	Diagnostics   []Diagnostic   `json:"diagnostics"`
	Runs          []RunRecord    `json:"runs,omitempty"`
}

type RunRecord struct {
	RunID             string   `json:"runId,omitempty"`
	AnalyzerID        string   `json:"analyzerId"`
	AnalyzerVersion   string   `json:"analyzerVersion"`
	Stage             string   `json:"stage"`
	InputGraphVersion string   `json:"inputGraphVersion"`
	ConfigurationHash string   `json:"configurationHash"`
	StartedAt         string   `json:"startedAt,omitempty"`
	FinishedAt        string   `json:"finishedAt,omitempty"`
	Status            string   `json:"status"`
	Diagnostics       []string `json:"diagnostics,omitempty"`
	EmittedFacts      int      `json:"emittedFacts"`
}

type FactMetadata struct {
	ProducerID      string `json:"producerId,omitempty"`
	ProducerVersion string `json:"producerVersion,omitempty"`
	RunID           string `json:"runId,omitempty"`
	CreatedAt       string `json:"createdAt,omitempty"`
}

type SourceRecord struct {
	ID         string     `json:"id"`
	Kind       string     `json:"kind"`
	Path       string     `json:"path"`
	AbsPath    string     `json:"absPath,omitempty"`
	SourceHash string     `json:"sourceHash,omitempty"`
	Freshness  string     `json:"freshness"`
	Reason     string     `json:"reason,omitempty"`
	TrustLevel TrustLevel `json:"trustLevel"`
	FactMetadata
}

type Observation struct {
	ID         string            `json:"id"`
	Kind       string            `json:"kind"`
	Name       string            `json:"name"`
	Value      string            `json:"value,omitempty"`
	TargetID   string            `json:"targetId,omitempty"`
	TargetKind string            `json:"targetKind,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Evidence   []string          `json:"evidence,omitempty"`
	Source     string            `json:"source"`
	Confidence float64           `json:"confidence,omitempty"`
	TrustLevel TrustLevel        `json:"trustLevel"`
	Freshness  string            `json:"freshness,omitempty"`
	SourceFile string            `json:"sourceFile,omitempty"`
	LineRange  string            `json:"lineRange,omitempty"`
	FactMetadata
}

type Label struct {
	ID         string     `json:"id"`
	Kind       string     `json:"kind"`
	Name       string     `json:"name"`
	Value      string     `json:"value"`
	TargetID   string     `json:"targetId"`
	TargetKind string     `json:"targetKind,omitempty"`
	Source     string     `json:"source"`
	Confidence float64    `json:"confidence,omitempty"`
	Evidence   []string   `json:"evidence,omitempty"`
	TrustLevel TrustLevel `json:"trustLevel"`
	Freshness  string     `json:"freshness,omitempty"`
	FactMetadata
}

type Warning struct {
	ID                  string     `json:"id"`
	Kind                string     `json:"kind"`
	Reason              string     `json:"reason"`
	SuggestedNextAction string     `json:"suggestedNextAction,omitempty"`
	AffectedNodeID      string     `json:"affectedNodeId,omitempty"`
	AffectedEdgeID      string     `json:"affectedEdgeId,omitempty"`
	Evidence            []string   `json:"evidence"`
	TrustLevel          TrustLevel `json:"trustLevel"`
	SourceFile          string     `json:"sourceFile,omitempty"`
	LineRange           string     `json:"lineRange,omitempty"`
	Freshness           string     `json:"freshness,omitempty"`
	FactMetadata
}

type Node struct {
	ID          string     `json:"id"`
	Kind        string     `json:"kind"`
	Name        string     `json:"name"`
	TrustLevel  TrustLevel `json:"trustLevel"`
	Synthetic   bool       `json:"synthetic"`
	Freshness   string     `json:"freshness"`
	Reason      string     `json:"reason,omitempty"`
	SourceFile  string     `json:"sourceFile,omitempty"`
	SourceHash  string     `json:"sourceHash,omitempty"`
	LineRange   string     `json:"lineRange,omitempty"`
	Extractor   string     `json:"extractorVersion,omitempty"`
	PackagePath string     `json:"packagePath,omitempty"`
	ModulePath  string     `json:"modulePath,omitempty"`
	FactMetadata
}

type Edge struct {
	ID         string     `json:"id"`
	From       string     `json:"from"`
	To         string     `json:"to"`
	Kind       string     `json:"kind"`
	TrustLevel TrustLevel `json:"trustLevel"`
	Synthetic  bool       `json:"synthetic"`
	Complete   bool       `json:"complete"`
	Reason     string     `json:"reason,omitempty"`
	SourceFile string     `json:"sourceFile,omitempty"`
	LineRange  string     `json:"lineRange,omitempty"`
	Extractor  string     `json:"extractorVersion,omitempty"`
	FactMetadata
}

type QueryResult struct {
	ID                 string     `json:"id"`
	Status             string     `json:"status"`
	Query              string     `json:"query"`
	Reason             string     `json:"reason,omitempty"`
	Scope              string     `json:"scope"`
	RequiredTrustLevel TrustLevel `json:"requiredTrustLevel"`
	ActualTrustLevel   TrustLevel `json:"actualTrustLevel"`
	ProofStatus        string     `json:"proofStatus,omitempty"`
	Evidence           []string   `json:"evidence"`
	FactMetadata
}

type PathJob struct {
	ID                 string     `json:"id"`
	Status             string     `json:"status"`
	StartNode          string     `json:"startNode"`
	TargetNode         string     `json:"targetNode"`
	KnownSegments      []string   `json:"knownSegments"`
	Missing            []string   `json:"missing"`
	BlockingReason     string     `json:"blockingReason"`
	RequiredTrustLevel TrustLevel `json:"requiredTrustLevel"`
	CurrentTrustLevel  TrustLevel `json:"currentTrustLevel"`
	FactMetadata
}

type Freshness struct {
	FactID     string `json:"factId"`
	SourceFile string `json:"sourceFile"`
	OldHash    string `json:"oldHash"`
	NewHash    string `json:"newHash"`
	Status     string `json:"status"`
	Reason     string `json:"reason"`
	Extractor  string `json:"extractorVersion"`
	FactMetadata
}

type PolicyResult struct {
	ID                string     `json:"id"`
	Kind              string     `json:"kind"`
	RuleID            string     `json:"ruleId"`
	AnalyzerID        string     `json:"analyzerId"`
	Stage             string     `json:"stage"`
	Status            string     `json:"status"`
	Scope             string     `json:"scope"`
	Subject           string     `json:"subject"`
	Reason            string     `json:"reason,omitempty"`
	Evidence          []string   `json:"evidence"`
	TrustLevel        TrustLevel `json:"trustLevel"`
	ConfigurationHash string     `json:"configurationHash,omitempty"`
	SourceFile        string     `json:"sourceFile,omitempty"`
	LineRange         string     `json:"lineRange,omitempty"`
	FactMetadata
}

type Metric struct {
	ID                string     `json:"id"`
	Kind              string     `json:"kind"`
	Name              string     `json:"name"`
	Value             int        `json:"value"`
	Unit              string     `json:"unit"`
	Scope             string     `json:"scope"`
	Subject           string     `json:"subject"`
	AnalyzerID        string     `json:"analyzerId"`
	Stage             string     `json:"stage"`
	Status            string     `json:"status"`
	Threshold         int        `json:"threshold"`
	Reason            string     `json:"reason,omitempty"`
	Evidence          []string   `json:"evidence"`
	TrustLevel        TrustLevel `json:"trustLevel"`
	ConfigurationHash string     `json:"configurationHash,omitempty"`
	SourceFile        string     `json:"sourceFile,omitempty"`
	LineRange         string     `json:"lineRange,omitempty"`
	FactMetadata
}

type Diagnostic struct {
	Level          string     `json:"level"`
	Source         string     `json:"source"`
	Reason         string     `json:"reason"`
	Stage          string     `json:"stage,omitempty"`
	AnalyzerID     string     `json:"analyzerId,omitempty"`
	Status         string     `json:"status,omitempty"`
	PolicyResultID string     `json:"policyResultId,omitempty"`
	NodeID         string     `json:"nodeId,omitempty"`
	EdgeID         string     `json:"edgeId,omitempty"`
	SourceFile     string     `json:"sourceFile,omitempty"`
	LineRange      string     `json:"lineRange,omitempty"`
	TrustLevel     TrustLevel `json:"trustLevel,omitempty"`
	Evidence       []string   `json:"evidence,omitempty"`
	FactMetadata
}

func TrustRank(level TrustLevel) int {
	switch level {
	case TrustSyntaxObserved:
		return 1
	case TrustPackageLoaded:
		return 2
	case TrustTypeResolved:
		return 3
	default:
		return 0
	}
}

func HashBytes(src []byte) string {
	sum := sha256.Sum256(src)
	return hex.EncodeToString(sum[:])
}

func SortGraph(graph Graph) Graph {
	sort.Slice(graph.Nodes, func(i, j int) bool { return graph.Nodes[i].ID < graph.Nodes[j].ID })
	sort.Slice(graph.Edges, func(i, j int) bool { return graph.Edges[i].ID < graph.Edges[j].ID })
	sort.Slice(graph.SourceRecords, func(i, j int) bool { return graph.SourceRecords[i].ID < graph.SourceRecords[j].ID })
	sort.Slice(graph.Observations, func(i, j int) bool { return graph.Observations[i].ID < graph.Observations[j].ID })
	sort.Slice(graph.Labels, func(i, j int) bool { return graph.Labels[i].ID < graph.Labels[j].ID })
	sort.Slice(graph.Warnings, func(i, j int) bool { return graph.Warnings[i].ID < graph.Warnings[j].ID })
	sort.Slice(graph.PolicyResults, func(i, j int) bool { return graph.PolicyResults[i].ID < graph.PolicyResults[j].ID })
	sort.Slice(graph.Metrics, func(i, j int) bool { return graph.Metrics[i].ID < graph.Metrics[j].ID })
	sort.Slice(graph.Diagnostics, func(i, j int) bool {
		if graph.Diagnostics[i].Source == graph.Diagnostics[j].Source {
			return graph.Diagnostics[i].Reason < graph.Diagnostics[j].Reason
		}
		return graph.Diagnostics[i].Source < graph.Diagnostics[j].Source
	})
	return graph
}
