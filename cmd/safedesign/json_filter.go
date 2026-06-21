package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/ravinsharma7/go-safedesign/internal/core"
)

type graphJSONFilterOptions struct {
	Sections         []string
	NodeKinds        []string
	EdgeKinds        []string
	ObservationNames []string
}

func filterGraphJSON(graph core.Graph, options graphJSONFilterOptions) (core.Graph, error) {
	if len(options.Sections) == 0 && len(options.NodeKinds) == 0 && len(options.EdgeKinds) == 0 && len(options.ObservationNames) == 0 {
		return graph, nil
	}

	sectionSet, err := graphJSONSectionSet(options.Sections)
	if err != nil {
		return core.Graph{}, err
	}

	filtered := core.Graph{}
	copyGraphSections(&filtered, graph, sectionSet)
	filtered.Nodes = filterByStringSet(filtered.Nodes, stringSet(options.NodeKinds), func(node core.Node) string {
		return node.Kind
	})
	filtered.Edges = filterByStringSet(filtered.Edges, stringSet(options.EdgeKinds), func(edge core.Edge) string {
		return edge.Kind
	})
	filtered.Observations = filterByStringSet(filtered.Observations, stringSet(options.ObservationNames), func(observation core.Observation) string {
		return observation.Name
	})
	return filtered, nil
}

func copyGraphSections(dst *core.Graph, src core.Graph, sections map[string]bool) {
	dstValue := reflect.ValueOf(dst).Elem()
	srcValue := reflect.ValueOf(src)
	graphType := srcValue.Type()
	for i := 0; i < graphType.NumField(); i++ {
		section := jsonTagName(graphType.Field(i))
		if section == "" || !includeGraphJSONSection(sections, section) {
			continue
		}
		dstValue.Field(i).Set(srcValue.Field(i))
	}
}

func graphJSONSectionNames() []string {
	return core.GraphJSONSections()
}

func graphJSONSectionSet(sections []string) (map[string]bool, error) {
	if len(sections) == 0 {
		return nil, nil
	}
	out := make(map[string]bool, len(sections))
	for _, section := range sections {
		canonical, ok := core.CanonicalGraphJSONSection(section)
		if !ok {
			return nil, fmt.Errorf("unknown --json-sections value %q", section)
		}
		out[canonical] = true
	}
	return out, nil
}

func includeGraphJSONSection(sections map[string]bool, section string) bool {
	return len(sections) == 0 || sections[section]
}

func jsonTagName(field reflect.StructField) string {
	name := strings.Split(field.Tag.Get("json"), ",")[0]
	if name == "-" {
		return ""
	}
	return name
}

func stringSet(values []string) map[string]bool {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]bool, len(values))
	for _, value := range values {
		out[value] = true
	}
	return out
}

func filterByStringSet[T any](items []T, allowed map[string]bool, value func(T) string) []T {
	if len(allowed) == 0 {
		return items
	}
	out := make([]T, 0, len(items))
	for _, item := range items {
		if allowed[value(item)] {
			out = append(out, item)
		}
	}
	return out
}
