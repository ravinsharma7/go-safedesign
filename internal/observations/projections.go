package observations

import (
	"sort"
	"strings"

	"go-safedesign/internal/core"
)

type TermSummary struct {
	Term           string
	Count          int
	Packages       []string
	NodeKinds      []string
	ObservationIDs []string
}

type SpellingSummary struct {
	Original       string
	Term           string
	Count          int
	ObservationIDs []string
}

func TermsByPackage(graph core.Graph) map[string][]TermSummary {
	byPackage := map[string]map[string]*termAccumulator{}
	for _, observation := range graph.Observations {
		if !isVocabularyTerm(observation) {
			continue
		}
		pkg := observation.Attributes["packagePath"]
		if pkg == "" {
			continue
		}
		if byPackage[pkg] == nil {
			byPackage[pkg] = map[string]*termAccumulator{}
		}
		acc := byPackage[pkg][observation.Value]
		if acc == nil {
			acc = newTermAccumulator(observation.Value)
			byPackage[pkg][observation.Value] = acc
		}
		acc.add(observation)
	}

	out := map[string][]TermSummary{}
	for pkg, terms := range byPackage {
		out[pkg] = sortedTermSummaries(terms)
	}
	return out
}

func TopTermsByPackage(graph core.Graph, limit int) map[string][]TermSummary {
	terms := TermsByPackage(graph)
	if limit <= 0 {
		return terms
	}
	out := map[string][]TermSummary{}
	for pkg, summaries := range terms {
		if len(summaries) > limit {
			summaries = summaries[:limit]
		}
		out[pkg] = summaries
	}
	return out
}

func SpellingsByTerm(graph core.Graph) map[string][]SpellingSummary {
	byTerm := map[string]map[string]*spellingAccumulator{}
	for _, observation := range graph.Observations {
		if !isVocabularyTerm(observation) {
			continue
		}
		original := observation.Attributes["original"]
		if original == "" {
			continue
		}
		if byTerm[observation.Value] == nil {
			byTerm[observation.Value] = map[string]*spellingAccumulator{}
		}
		acc := byTerm[observation.Value][original]
		if acc == nil {
			acc = &spellingAccumulator{Original: original, Term: observation.Value}
			byTerm[observation.Value][original] = acc
		}
		acc.Count++
		acc.ObservationIDs = append(acc.ObservationIDs, observation.ID)
	}

	out := map[string][]SpellingSummary{}
	for term, spellings := range byTerm {
		summaries := make([]SpellingSummary, 0, len(spellings))
		for _, acc := range spellings {
			sort.Strings(acc.ObservationIDs)
			summaries = append(summaries, SpellingSummary{
				Original:       acc.Original,
				Term:           acc.Term,
				Count:          acc.Count,
				ObservationIDs: acc.ObservationIDs,
			})
		}
		sort.Slice(summaries, func(i, j int) bool {
			if summaries[i].Count != summaries[j].Count {
				return summaries[i].Count > summaries[j].Count
			}
			return summaries[i].Original < summaries[j].Original
		})
		out[term] = summaries
	}
	return out
}

func ObservationsByTarget(graph core.Graph) map[string][]core.Observation {
	out := map[string][]core.Observation{}
	for _, observation := range graph.Observations {
		if observation.TargetID == "" {
			continue
		}
		out[observation.TargetID] = append(out[observation.TargetID], observation)
	}
	sortObservationGroups(out)
	return out
}

func IncompleteDependenciesByPackage(graph core.Graph) map[string][]core.Observation {
	out := map[string][]core.Observation{}
	for _, observation := range graph.Observations {
		if observation.Name != core.ObservationNameVocabularyIncompleteDependency {
			continue
		}
		pkg := packageFromNodeID(observation.Attributes["from"])
		if pkg == "" {
			continue
		}
		out[pkg] = append(out[pkg], observation)
	}
	sortObservationGroups(out)
	return out
}

type termAccumulator struct {
	Term           string
	Count          int
	Packages       map[string]bool
	NodeKinds      map[string]bool
	ObservationIDs []string
}

func newTermAccumulator(term string) *termAccumulator {
	return &termAccumulator{
		Term:      term,
		Packages:  map[string]bool{},
		NodeKinds: map[string]bool{},
	}
}

func (acc *termAccumulator) add(observation core.Observation) {
	acc.Count++
	acc.ObservationIDs = append(acc.ObservationIDs, observation.ID)
	if pkg := observation.Attributes["packagePath"]; pkg != "" {
		acc.Packages[pkg] = true
	}
	if kind := observation.Attributes["nodeKind"]; kind != "" {
		acc.NodeKinds[kind] = true
	}
}

type spellingAccumulator struct {
	Original       string
	Term           string
	Count          int
	ObservationIDs []string
}

func sortedTermSummaries(terms map[string]*termAccumulator) []TermSummary {
	summaries := make([]TermSummary, 0, len(terms))
	for _, acc := range terms {
		sort.Strings(acc.ObservationIDs)
		summaries = append(summaries, TermSummary{
			Term:           acc.Term,
			Count:          acc.Count,
			Packages:       sortedKeys(acc.Packages),
			NodeKinds:      sortedKeys(acc.NodeKinds),
			ObservationIDs: acc.ObservationIDs,
		})
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Count != summaries[j].Count {
			return summaries[i].Count > summaries[j].Count
		}
		return summaries[i].Term < summaries[j].Term
	})
	return summaries
}

func isVocabularyTerm(observation core.Observation) bool {
	return observation.Name == core.ObservationNameVocabularyTerm && observation.Value != ""
}

func packageFromNodeID(id string) string {
	value, ok := strings.CutPrefix(id, core.IDPrefixPackage)
	if !ok {
		return ""
	}
	return value
}

func sortObservationGroups(groups map[string][]core.Observation) {
	for key := range groups {
		sort.Slice(groups[key], func(i, j int) bool {
			return groups[key][i].ID < groups[key][j].ID
		})
	}
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
