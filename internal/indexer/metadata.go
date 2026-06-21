package indexer

import "github.com/ravinsharma7/go-safedesign/internal/core"

func (b *graphBuilder) metadataForCurrentRun() core.FactMetadata {
	run := b.currentRun
	return core.FactMetadata{
		ProducerID:      run.AnalyzerID,
		ProducerVersion: run.AnalyzerVersion,
		RunID:           run.RunID,
		CreatedAt:       run.StartedAt,
	}
}

func metadataForRun(run core.RunRecord) core.FactMetadata {
	return core.FactMetadata{
		ProducerID:      run.AnalyzerID,
		ProducerVersion: run.AnalyzerVersion,
		RunID:           run.RunID,
		CreatedAt:       run.StartedAt,
	}
}

func (b *graphBuilder) applyStageMetadata(run core.RunRecord) {
	meta := metadataForRun(run)
	for id, node := range b.nodes {
		if node.RunID == "" {
			node.FactMetadata = meta
			b.nodes[id] = node
		}
	}
	for id, edge := range b.edges {
		if edge.RunID == "" {
			edge.FactMetadata = meta
			b.edges[id] = edge
		}
	}
	for i := range b.sourceRecords {
		if b.sourceRecords[i].RunID == "" {
			b.sourceRecords[i].FactMetadata = meta
		}
	}
	for i := range b.observations {
		if b.observations[i].RunID == "" {
			b.observations[i].FactMetadata = meta
		}
	}
	for i := range b.freshness {
		if b.freshness[i].RunID == "" {
			b.freshness[i].FactMetadata = meta
		}
	}
	for i := range b.diagnostics {
		if b.diagnostics[i].RunID == "" {
			b.diagnostics[i].FactMetadata = meta
		}
	}
	for i := range b.queries {
		if b.queries[i].RunID == "" {
			b.queries[i].FactMetadata = meta
		}
	}
	for i := range b.pathJobs {
		if b.pathJobs[i].RunID == "" {
			b.pathJobs[i].FactMetadata = meta
		}
	}
}

func (b *graphBuilder) applyAnalyzerMetadata(run core.RunRecord) {
	meta := metadataForRun(run)
	for i := range b.policyResults {
		if b.policyResults[i].RunID == "" && b.policyResults[i].AnalyzerID == run.AnalyzerID && b.policyResults[i].Stage == run.Stage {
			b.policyResults[i].FactMetadata = meta
		}
	}
	for i := range b.metrics {
		if b.metrics[i].RunID == "" && b.metrics[i].AnalyzerID == run.AnalyzerID && b.metrics[i].Stage == run.Stage {
			b.metrics[i].FactMetadata = meta
		}
	}
	for i := range b.observations {
		if b.observations[i].RunID == "" {
			b.observations[i].FactMetadata = meta
		}
	}
	for i := range b.labels {
		if b.labels[i].RunID == "" {
			b.labels[i].FactMetadata = meta
		}
	}
	for i := range b.warnings {
		if b.warnings[i].RunID == "" {
			b.warnings[i].FactMetadata = meta
		}
	}
	for id, edge := range b.edges {
		if edge.RunID == "" && edge.Kind == "violates" {
			edge.FactMetadata = meta
			b.edges[id] = edge
		}
	}
	for i := range b.diagnostics {
		if b.diagnostics[i].RunID == "" && b.diagnostics[i].AnalyzerID == run.AnalyzerID && b.diagnostics[i].Stage == run.Stage {
			b.diagnostics[i].FactMetadata = meta
		}
	}
}
