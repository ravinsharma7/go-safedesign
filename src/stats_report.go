package main

import (
	"path/filepath"
	"sort"
	"strings"

	"go-safedesign/internal/core"
)

type statsReport struct {
	Workspace statsWorkspace    `json:"workspace"`
	Summary   graphStatsSummary `json:"summary"`
	Modules   []moduleStats     `json:"modules"`
	Packages  []packageStats    `json:"packages"`
	Files     []fileStats       `json:"files"`
}

type statsWorkspace struct {
	ProjectRoot         string   `json:"projectRoot,omitempty"`
	WorkspaceRoot       string   `json:"workspaceRoot,omitempty"`
	GoWorkPath          string   `json:"goWorkPath,omitempty"`
	GoModFiles          []string `json:"goModFiles"`
	IndexedModules      int      `json:"indexedModules"`
	IgnoredNestedGoMods int      `json:"ignoredNestedGoMods"`
	SingleModuleScan    bool     `json:"singleModuleScan"`
	Warning             string   `json:"warning,omitempty"`
}

type graphStatsSummary struct {
	Modules         int `json:"modules"`
	Packages        int `json:"packages"`
	Files           int `json:"files"`
	Functions       int `json:"functions"`
	Methods         int `json:"methods"`
	Types           int `json:"types"`
	Structs         int `json:"structs"`
	Interfaces      int `json:"interfaces"`
	Fields          int `json:"fields"`
	Imports         int `json:"imports"`
	Edges           int `json:"edges"`
	Observations    int `json:"observations"`
	Placeholders    int `json:"placeholders"`
	IncompleteEdges int `json:"incompleteEdges"`
}

type moduleStats struct {
	ModulePath            string   `json:"modulePath"`
	Dir                   string   `json:"dir,omitempty"`
	SourceFile            string   `json:"sourceFile,omitempty"`
	DiscoveryReasons      []string `json:"discoveryReasons"`
	Packages              int      `json:"packages"`
	Files                 int      `json:"files"`
	Imports               int      `json:"imports"`
	IncompleteImports     int      `json:"incompleteImports"`
	DependsOn             int      `json:"dependsOn"`
	DependedOnBy          int      `json:"dependedOnBy"`
	ExternalDependencies  int      `json:"externalDependencies"`
	MissingDependencies   int      `json:"missingDependencies"`
	CrossModuleImportsOut int      `json:"crossModuleImportsOut"`
	CrossModuleImportsIn  int      `json:"crossModuleImportsIn"`
	Observations          int      `json:"observations"`
	Candidates            int      `json:"candidates"`
	BridgesIn             int      `json:"bridgesIn"`
	BridgesOut            int      `json:"bridgesOut"`
}

type packageStats struct {
	PackagePath       string `json:"packagePath"`
	ModulePath        string `json:"modulePath,omitempty"`
	Files             int    `json:"files"`
	Functions         int    `json:"functions"`
	Methods           int    `json:"methods"`
	Types             int    `json:"types"`
	Structs           int    `json:"structs"`
	Interfaces        int    `json:"interfaces"`
	Imports           int    `json:"imports"`
	IncompleteImports int    `json:"incompleteImports"`
	Observations      int    `json:"observations"`
	Candidates        int    `json:"candidates"`
	BridgesIn         int    `json:"bridgesIn"`
	BridgesOut        int    `json:"bridgesOut"`
}

type fileStats struct {
	SourceFile   string `json:"sourceFile"`
	PackagePath  string `json:"packagePath,omitempty"`
	ModulePath   string `json:"modulePath,omitempty"`
	Functions    int    `json:"functions"`
	Methods      int    `json:"methods"`
	Types        int    `json:"types"`
	Structs      int    `json:"structs"`
	Interfaces   int    `json:"interfaces"`
	Observations int    `json:"observations"`
}

type reportScope struct {
	modulePath      string
	packagePath     string
	sourceFile      string
	packageToModule map[string]string
	nodeToModule    map[string]string
}

func buildStatsReport(graph core.Graph, options compactReportOptions) statsReport {
	options = options.normalized()
	scope := newReportScope(graph, options)
	moduleMap := map[string]*moduleStats{}
	packageMap := map[string]*packageStats{}
	fileMap := map[string]*fileStats{}
	var summary graphStatsSummary

	moduleDiscovery := moduleDiscoveryRecords(graph)
	for _, node := range graph.Nodes {
		if !scope.matchesNode(node) {
			continue
		}
		switch node.Kind {
		case core.NodeKindModule:
			if !core.IsPlaceholderNode(node) {
				summary.Modules++
				stats := ensureModuleStats(moduleMap, node.ModulePath)
				stats.SourceFile = firstNonEmpty(stats.SourceFile, node.SourceFile)
				if discovery := moduleDiscovery[node.ModulePath]; discovery != nil {
					stats.Dir = firstNonEmpty(stats.Dir, discovery.dir)
					stats.DiscoveryReasons = mergeStringSet(stats.DiscoveryReasons, discovery.reasons)
				}
			}
		case core.NodeKindPackage:
			if !core.IsPlaceholderNode(node) {
				summary.Packages++
				stats := ensurePackageStats(packageMap, node.PackagePath)
				stats.ModulePath = firstNonEmpty(stats.ModulePath, node.ModulePath)
				ensureModuleStats(moduleMap, node.ModulePath).Packages++
			}
		case core.NodeKindFile:
			summary.Files++
			stats := ensurePackageStats(packageMap, node.PackagePath)
			stats.ModulePath = firstNonEmpty(stats.ModulePath, node.ModulePath)
			stats.Files++
			ensureModuleStats(moduleMap, node.ModulePath).Files++
			file := ensureFileStats(fileMap, node.SourceFile)
			file.PackagePath = firstNonEmpty(file.PackagePath, node.PackagePath)
			file.ModulePath = firstNonEmpty(file.ModulePath, node.ModulePath)
		case core.NodeKindFunction:
			summary.Functions++
			ensurePackageStats(packageMap, node.PackagePath).Functions++
			ensureFileStats(fileMap, node.SourceFile).Functions++
		case core.NodeKindMethod:
			summary.Methods++
			ensurePackageStats(packageMap, node.PackagePath).Methods++
			ensureFileStats(fileMap, node.SourceFile).Methods++
		case core.NodeKindType:
			summary.Types++
			ensurePackageStats(packageMap, node.PackagePath).Types++
			ensureFileStats(fileMap, node.SourceFile).Types++
		case core.NodeKindStruct:
			summary.Structs++
			ensurePackageStats(packageMap, node.PackagePath).Structs++
			ensureFileStats(fileMap, node.SourceFile).Structs++
		case core.NodeKindInterface:
			summary.Interfaces++
			ensurePackageStats(packageMap, node.PackagePath).Interfaces++
			ensureFileStats(fileMap, node.SourceFile).Interfaces++
		case core.NodeKindField:
			summary.Fields++
		case core.NodeKindImport:
			summary.Imports++
			ensurePackageStats(packageMap, node.PackagePath).Imports++
			ensureModuleStats(moduleMap, node.ModulePath).Imports++
		case core.NodeKindPlaceholder:
			summary.Placeholders++
		}
		if core.IsPlaceholderNode(node) && node.Kind != core.NodeKindPlaceholder {
			summary.Placeholders++
		}
	}

	for _, edge := range graph.Edges {
		if !scope.matchesEdge(edge) {
			continue
		}
		summary.Edges++
		if core.IsIncompleteEdge(edge) {
			summary.IncompleteEdges++
		}
		switch edge.Kind {
		case core.EdgeKindDependsOn:
			fromModule := modulePathFromFactID(edge.From)
			toModule := modulePathFromFactID(edge.To)
			if fromModule == "" {
				continue
			}
			if core.IsPlaceholderID(edge.To) {
				ensureModuleStats(moduleMap, fromModule).MissingDependencies++
				continue
			}
			if toModule != "" && fromModule != toModule {
				ensureModuleStats(moduleMap, fromModule).DependsOn++
				ensureModuleStats(moduleMap, toModule).DependedOnBy++
			}
		case core.EdgeKindImports:
			fromPackage := packagePathFromFactID(edge.From)
			fromModule := scope.moduleForPackage(fromPackage)
			toPackage := packagePathFromFactID(edge.To)
			toModule := scope.moduleForPackage(toPackage)
			if core.IsIncompleteEdge(edge) {
				if fromPackage != "" {
					ensurePackageStats(packageMap, fromPackage).IncompleteImports++
				}
				if fromModule != "" {
					ensureModuleStats(moduleMap, fromModule).IncompleteImports++
					if core.IsPlaceholderID(edge.To) && toModule == "" {
						ensureModuleStats(moduleMap, fromModule).ExternalDependencies++
					}
				}
			}
			if fromModule != "" && toModule != "" && fromModule != toModule {
				ensureModuleStats(moduleMap, fromModule).CrossModuleImportsOut++
				ensureModuleStats(moduleMap, toModule).CrossModuleImportsIn++
			}
		}
	}

	for _, observation := range graph.Observations {
		if !scope.matchesObservation(observation) {
			continue
		}
		summary.Observations++
		if pkg := packagePathForObservation(observation); pkg != "" {
			stats := ensurePackageStats(packageMap, pkg)
			stats.Observations++
			modulePath := scope.moduleForPackage(pkg)
			if modulePath != "" {
				ensureModuleStats(moduleMap, modulePath).Observations++
			}
			if observation.Name == core.ObservationNameLanguageZoneCandidate {
				stats.Candidates++
				if modulePath != "" {
					ensureModuleStats(moduleMap, modulePath).Candidates++
				}
			}
		}
		if observation.SourceFile != "" {
			ensureFileStats(fileMap, observation.SourceFile).Observations++
		}
		if observation.Name == core.ObservationNameBridgeSymbol {
			fromPackage := observation.Attributes["fromPackagePath"]
			toPackage := observation.Attributes["toPackagePath"]
			fromModule := scope.moduleForPackage(fromPackage)
			toModule := scope.moduleForPackage(toPackage)
			if fromPackage != "" {
				ensurePackageStats(packageMap, fromPackage).BridgesOut++
			}
			if toPackage != "" {
				ensurePackageStats(packageMap, toPackage).BridgesIn++
			}
			if fromModule != "" {
				ensureModuleStats(moduleMap, fromModule).BridgesOut++
			}
			if toModule != "" {
				ensureModuleStats(moduleMap, toModule).BridgesIn++
			}
		}
	}

	return statsReport{
		Workspace: workspaceStats(graph, indexedModuleCount(graph)),
		Summary:   summary,
		Modules:   sortedModuleStats(moduleMap),
		Packages:  sortedPackageStats(packageMap),
		Files:     sortedFileStats(fileMap),
	}
}

type moduleDiscoverySummary struct {
	dir     string
	reasons []string
}

func moduleDiscoveryRecords(graph core.Graph) map[string]*moduleDiscoverySummary {
	out := map[string]*moduleDiscoverySummary{}
	for _, record := range graph.SourceRecords {
		if record.Kind != "module_discovery" {
			continue
		}
		// module_discovery records use the module directory as Path and comma-separated reasons in Reason.
		var modulePath string
		for _, node := range graph.Nodes {
			if node.Kind == core.NodeKindModule && !core.IsPlaceholderNode(node) && moduleDirFromSourceFile(node.SourceFile) == record.Path {
				modulePath = node.ModulePath
				break
			}
		}
		if modulePath == "" {
			continue
		}
		summary := out[modulePath]
		if summary == nil {
			summary = &moduleDiscoverySummary{dir: record.Path}
			out[modulePath] = summary
		}
		summary.reasons = mergeStringSet(summary.reasons, splitCommaList(record.Reason))
	}
	return out
}

func moduleDirFromSourceFile(sourceFile string) string {
	if sourceFile == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Dir(sourceFile))
}

func workspaceStats(graph core.Graph, indexedModules int) statsWorkspace {
	stats := statsWorkspace{IndexedModules: indexedModules}
	for _, record := range graph.SourceRecords {
		switch record.Kind {
		case "project_root":
			stats.ProjectRoot = record.Path
		case "workspace_root":
			stats.WorkspaceRoot = record.Path
		case "go_work":
			stats.GoWorkPath = record.Path
		case "go_mod":
			stats.GoModFiles = append(stats.GoModFiles, record.Path)
		}
	}
	sort.Strings(stats.GoModFiles)
	if stats.GoModFiles == nil {
		stats.GoModFiles = []string{}
	}
	stats.SingleModuleScan = stats.ProjectRoot != "" && stats.ProjectRoot == stats.WorkspaceRoot
	if stats.SingleModuleScan && len(stats.GoModFiles) > indexedModules {
		stats.IgnoredNestedGoMods = len(stats.GoModFiles) - indexedModules
		stats.Warning = "single module scan: additional go.mod files were discovered under the project root but not indexed; pass --workspace-root to analyze a multi-module workspace"
	}
	return stats
}

func indexedModuleCount(graph core.Graph) int {
	count := 0
	for _, node := range graph.Nodes {
		if node.Kind == core.NodeKindModule && !core.IsPlaceholderNode(node) {
			count++
		}
	}
	return count
}

func newReportScope(graph core.Graph, options compactReportOptions) reportScope {
	scope := reportScope{
		modulePath:      options.ScopeModule,
		packagePath:     options.ScopePackage,
		sourceFile:      options.ScopeFile,
		packageToModule: map[string]string{},
		nodeToModule:    map[string]string{},
	}
	for _, node := range graph.Nodes {
		if node.ID != "" && node.ModulePath != "" {
			scope.nodeToModule[node.ID] = node.ModulePath
		}
		if node.PackagePath != "" && node.ModulePath != "" && !core.IsPlaceholderNode(node) {
			scope.packageToModule[node.PackagePath] = node.ModulePath
			scope.packageToModule[core.PackageID(node.PackagePath)] = node.ModulePath
		}
	}
	return scope
}

func (scope reportScope) active() bool {
	return scope.modulePath != "" || scope.packagePath != "" || scope.sourceFile != ""
}

func (scope reportScope) matchesNode(node core.Node) bool {
	if !scope.active() {
		return true
	}
	if scope.modulePath != "" {
		return node.ModulePath == scope.modulePath
	}
	if scope.packagePath != "" {
		return node.PackagePath == scope.packagePath
	}
	return node.SourceFile == scope.sourceFile
}

func (scope reportScope) matchesEdge(edge core.Edge) bool {
	if !scope.active() {
		return true
	}
	if scope.modulePath != "" {
		return scope.factInModule(edge.From, scope.modulePath) ||
			scope.factInModule(edge.To, scope.modulePath) ||
			edge.SourceFile != "" && scope.sourceFile != "" && edge.SourceFile == scope.sourceFile
	}
	if scope.packagePath != "" {
		packageID := core.PackageID(scope.packagePath)
		return edge.From == packageID || edge.To == packageID || packagePathFromFactID(edge.From) == scope.packagePath || packagePathFromFactID(edge.To) == scope.packagePath
	}
	return edge.SourceFile == scope.sourceFile
}

func (scope reportScope) matchesObservation(observation core.Observation) bool {
	if !scope.active() {
		return true
	}
	if scope.modulePath != "" {
		return scope.observationReferencesModule(observation, scope.modulePath)
	}
	if scope.packagePath != "" {
		return observationReferencesPackage(observation, scope.packagePath)
	}
	return observation.SourceFile == scope.sourceFile
}

func (scope reportScope) factInModule(id, modulePath string) bool {
	if id == "" || modulePath == "" {
		return false
	}
	if modulePathFromFactID(id) == modulePath {
		return true
	}
	if scope.nodeToModule[id] == modulePath {
		return true
	}
	if scope.packageToModule[id] == modulePath {
		return true
	}
	if pkg := packagePathFromFactID(id); pkg != "" {
		return scope.moduleForPackage(pkg) == modulePath
	}
	return false
}

func (scope reportScope) observationReferencesModule(observation core.Observation, modulePath string) bool {
	if observation.Attributes["modulePath"] == modulePath || scope.factInModule(observation.TargetID, modulePath) {
		return true
	}
	for _, key := range []string{"packagePath", "fromPackagePath", "toPackagePath"} {
		if scope.moduleForPackage(observation.Attributes[key]) == modulePath {
			return true
		}
	}
	for _, key := range []string{"from", "to"} {
		if scope.factInModule(observation.Attributes[key], modulePath) {
			return true
		}
	}
	return false
}

func (scope reportScope) moduleForPackage(packagePath string) string {
	if packagePath == "" {
		return ""
	}
	if modulePath := scope.packageToModule[packagePath]; modulePath != "" {
		return modulePath
	}
	if modulePath := scope.packageToModule[core.PackageID(packagePath)]; modulePath != "" {
		return modulePath
	}
	return ""
}

func observationReferencesPackage(observation core.Observation, packagePath string) bool {
	if packagePath == "" {
		return false
	}
	if observation.Attributes["packagePath"] == packagePath ||
		observation.Attributes["fromPackagePath"] == packagePath ||
		observation.Attributes["toPackagePath"] == packagePath {
		return true
	}
	packageID := core.PackageID(packagePath)
	return observation.TargetID == packageID ||
		observation.Attributes["from"] == packageID ||
		observation.Attributes["to"] == packageID ||
		strings.Contains(observation.TargetID, packageID) ||
		strings.Contains(observation.Attributes["from"], packageID) ||
		strings.Contains(observation.Attributes["to"], packageID)
}

func packagePathForObservation(observation core.Observation) string {
	if pkg := observation.Attributes["packagePath"]; pkg != "" {
		return pkg
	}
	if pkg := observation.Attributes["fromPackagePath"]; pkg != "" {
		return pkg
	}
	if pkg := packagePathFromFactID(observation.Attributes["from"]); pkg != "" {
		return pkg
	}
	return packagePathFromFactID(observation.TargetID)
}

func packagePathFromFactID(id string) string {
	if value, ok := strings.CutPrefix(id, core.IDPrefixPackage); ok {
		return value
	}
	return ""
}

func modulePathFromFactID(id string) string {
	if value, ok := strings.CutPrefix(id, core.IDPrefixModule); ok {
		return value
	}
	if value, ok := strings.CutPrefix(id, core.IDPrefixPlaceholderModule); ok {
		return value
	}
	return ""
}

func ensureModuleStats(values map[string]*moduleStats, modulePath string) *moduleStats {
	if modulePath == "" {
		modulePath = "(unknown)"
	}
	stats := values[modulePath]
	if stats == nil {
		stats = &moduleStats{ModulePath: modulePath, DiscoveryReasons: []string{}}
		values[modulePath] = stats
	}
	return stats
}

func ensurePackageStats(values map[string]*packageStats, packagePath string) *packageStats {
	if packagePath == "" {
		packagePath = "(unknown)"
	}
	stats := values[packagePath]
	if stats == nil {
		stats = &packageStats{PackagePath: packagePath}
		values[packagePath] = stats
	}
	return stats
}

func ensureFileStats(values map[string]*fileStats, sourceFile string) *fileStats {
	if sourceFile == "" {
		sourceFile = "(unknown)"
	}
	stats := values[sourceFile]
	if stats == nil {
		stats = &fileStats{SourceFile: sourceFile}
		values[sourceFile] = stats
	}
	return stats
}

func sortedModuleStats(values map[string]*moduleStats) []moduleStats {
	out := make([]moduleStats, 0, len(values))
	for _, stats := range values {
		out = append(out, *stats)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ModulePath < out[j].ModulePath })
	return out
}

func sortedPackageStats(values map[string]*packageStats) []packageStats {
	out := make([]packageStats, 0, len(values))
	for _, stats := range values {
		out = append(out, *stats)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ModulePath != out[j].ModulePath {
			return out[i].ModulePath < out[j].ModulePath
		}
		return out[i].PackagePath < out[j].PackagePath
	})
	return out
}

func sortedFileStats(values map[string]*fileStats) []fileStats {
	out := make([]fileStats, 0, len(values))
	for _, stats := range values {
		out = append(out, *stats)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ModulePath != out[j].ModulePath {
			return out[i].ModulePath < out[j].ModulePath
		}
		if out[i].PackagePath != out[j].PackagePath {
			return out[i].PackagePath < out[j].PackagePath
		}
		return out[i].SourceFile < out[j].SourceFile
	})
	return out
}

func firstNonEmpty(current, next string) string {
	if current != "" {
		return current
	}
	return next
}

func mergeStringSet(left, right []string) []string {
	values := map[string]bool{}
	for _, value := range left {
		if value != "" {
			values[value] = true
		}
	}
	for _, value := range right {
		if value != "" {
			values[value] = true
		}
	}
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
