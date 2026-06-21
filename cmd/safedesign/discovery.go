package main

import (
	"github.com/ravinsharma7/go-safedesign/internal/core"
	"github.com/ravinsharma7/go-safedesign/internal/indexer"
)

type valueDiscoveryReport struct {
	JSONSections       []string              `json:"jsonSections"`
	NodeKinds          []string              `json:"nodeKinds"`
	EdgeKinds          []string              `json:"edgeKinds"`
	FactKinds          []string              `json:"factKinds"`
	Statuses           []string              `json:"statuses"`
	Freshness          []string              `json:"freshness"`
	ObservationNames   []string              `json:"observationNames"`
	ObservationSources []string              `json:"observationSources"`
	TrustLevels        []core.TrustLevelInfo `json:"trustLevels"`
	AnalyzerIDs        []string              `json:"analyzerIds"`
}

func buildValueDiscoveryReport() valueDiscoveryReport {
	return valueDiscoveryReport{
		JSONSections:       graphJSONSectionNames(),
		NodeKinds:          core.NodeKinds(),
		EdgeKinds:          core.EdgeKinds(),
		FactKinds:          core.FactKinds(),
		Statuses:           core.Statuses(),
		Freshness:          core.FreshnessStatuses(),
		ObservationNames:   core.ObservationNames(),
		ObservationSources: core.ObservationSources(),
		TrustLevels:        core.TrustLevelInfos(),
		AnalyzerIDs:        indexer.KnownAnalyzerIDs(),
	}
}
