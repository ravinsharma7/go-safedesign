package main

import (
	"sort"
	"strconv"
	"strings"

	"go-safedesign/internal/core"
)

type dddReport struct {
	Summary            dddReportSummary               `json:"summary"`
	Stats              statsReport                    `json:"stats"`
	Candidates         []dddCandidateSummary          `json:"candidates"`
	Bridges            []dddBridgeSummary             `json:"bridges"`
	IncompleteEvidence []dddIncompleteEvidenceSummary `json:"incompleteEvidence"`
}

type dddReportSummary struct {
	Candidates         int  `json:"candidates"`
	Bridges            int  `json:"bridges"`
	IncompleteEvidence int  `json:"incompleteEvidence"`
	Truncated          bool `json:"truncated"`
}

type dddCandidateSummary struct {
	PackagePath       string          `json:"packagePath"`
	TargetID          string          `json:"targetId,omitempty"`
	Terms             []string        `json:"terms"`
	TermCount         int             `json:"termCount"`
	CooccurrenceCount int             `json:"cooccurrenceCount"`
	EvidenceCount     int             `json:"evidenceCount"`
	TrustLevel        core.TrustLevel `json:"trustLevel"`
	ObservationID     string          `json:"observationId"`
}

type dddBridgeSummary struct {
	FromPackagePath string          `json:"fromPackagePath"`
	ToPackagePath   string          `json:"toPackagePath"`
	EdgeKind        string          `json:"edgeKind"`
	TargetID        string          `json:"targetId"`
	EvidenceCount   int             `json:"evidenceCount"`
	TrustLevel      core.TrustLevel `json:"trustLevel"`
	ObservationID   string          `json:"observationId"`
}

type dddIncompleteEvidenceSummary struct {
	From          string          `json:"from,omitempty"`
	To            string          `json:"to,omitempty"`
	EdgeKind      string          `json:"edgeKind,omitempty"`
	Reason        string          `json:"reason,omitempty"`
	TargetID      string          `json:"targetId,omitempty"`
	EvidenceCount int             `json:"evidenceCount"`
	TrustLevel    core.TrustLevel `json:"trustLevel"`
	ObservationID string          `json:"observationId"`
}

func buildDDDReport(graph core.Graph, options compactReportOptions) dddReport {
	options = options.normalized()
	scope := newReportScope(graph, options)
	report := dddReport{Stats: buildStatsReport(graph, options)}
	for _, observation := range graph.Observations {
		if !scope.matchesObservation(observation) {
			continue
		}
		switch observation.Name {
		case core.ObservationNameLanguageZoneCandidate:
			report.Candidates = append(report.Candidates, dddCandidateSummary{
				PackagePath:       observation.Attributes["packagePath"],
				TargetID:          observation.TargetID,
				Terms:             splitTerms(observation.Attributes["terms"]),
				TermCount:         intAttribute(observation.Attributes, "termCount"),
				CooccurrenceCount: intAttribute(observation.Attributes, "cooccurrenceCount"),
				EvidenceCount:     len(observation.Evidence),
				TrustLevel:        observation.TrustLevel,
				ObservationID:     observation.ID,
			})
		case core.ObservationNameBridgeSymbol:
			report.Bridges = append(report.Bridges, dddBridgeSummary{
				FromPackagePath: observation.Attributes["fromPackagePath"],
				ToPackagePath:   observation.Attributes["toPackagePath"],
				EdgeKind:        observation.Attributes["edgeKind"],
				TargetID:        observation.TargetID,
				EvidenceCount:   len(observation.Evidence),
				TrustLevel:      observation.TrustLevel,
				ObservationID:   observation.ID,
			})
		case core.ObservationNameVocabularyIncompleteDependency:
			report.IncompleteEvidence = append(report.IncompleteEvidence, dddIncompleteEvidenceSummary{
				From:          observation.Attributes["from"],
				To:            observation.Attributes["to"],
				EdgeKind:      observation.Attributes["edgeKind"],
				Reason:        observation.Attributes["reason"],
				TargetID:      observation.TargetID,
				EvidenceCount: len(observation.Evidence),
				TrustLevel:    observation.TrustLevel,
				ObservationID: observation.ID,
			})
		}
	}

	sort.Slice(report.Candidates, func(i, j int) bool {
		if report.Candidates[i].PackagePath != report.Candidates[j].PackagePath {
			return report.Candidates[i].PackagePath < report.Candidates[j].PackagePath
		}
		return report.Candidates[i].ObservationID < report.Candidates[j].ObservationID
	})
	sort.Slice(report.Bridges, func(i, j int) bool {
		if report.Bridges[i].FromPackagePath != report.Bridges[j].FromPackagePath {
			return report.Bridges[i].FromPackagePath < report.Bridges[j].FromPackagePath
		}
		if report.Bridges[i].ToPackagePath != report.Bridges[j].ToPackagePath {
			return report.Bridges[i].ToPackagePath < report.Bridges[j].ToPackagePath
		}
		if report.Bridges[i].EdgeKind != report.Bridges[j].EdgeKind {
			return report.Bridges[i].EdgeKind < report.Bridges[j].EdgeKind
		}
		return report.Bridges[i].ObservationID < report.Bridges[j].ObservationID
	})
	sort.Slice(report.IncompleteEvidence, func(i, j int) bool {
		if report.IncompleteEvidence[i].From != report.IncompleteEvidence[j].From {
			return report.IncompleteEvidence[i].From < report.IncompleteEvidence[j].From
		}
		if report.IncompleteEvidence[i].To != report.IncompleteEvidence[j].To {
			return report.IncompleteEvidence[i].To < report.IncompleteEvidence[j].To
		}
		if report.IncompleteEvidence[i].EdgeKind != report.IncompleteEvidence[j].EdgeKind {
			return report.IncompleteEvidence[i].EdgeKind < report.IncompleteEvidence[j].EdgeKind
		}
		return report.IncompleteEvidence[i].ObservationID < report.IncompleteEvidence[j].ObservationID
	})

	report.Summary.Candidates = len(report.Candidates)
	report.Candidates, report.Summary.Truncated = limited(report.Candidates, options.Limit, report.Summary.Truncated)
	report.Summary.Bridges = len(report.Bridges)
	report.Bridges, report.Summary.Truncated = limited(report.Bridges, options.Limit, report.Summary.Truncated)
	report.Summary.IncompleteEvidence = len(report.IncompleteEvidence)
	report.IncompleteEvidence, report.Summary.Truncated = limited(report.IncompleteEvidence, options.Limit, report.Summary.Truncated)

	return report
}

func splitTerms(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	terms := make([]string, 0, len(parts))
	for _, part := range parts {
		term := strings.TrimSpace(part)
		if term != "" {
			terms = append(terms, term)
		}
	}
	return terms
}

func intAttribute(attributes map[string]string, key string) int {
	value, err := strconv.Atoi(attributes[key])
	if err != nil {
		return 0
	}
	return value
}
