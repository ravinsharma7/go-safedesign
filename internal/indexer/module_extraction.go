package indexer

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ravinsharma7/go-safedesign/internal/core"
	"github.com/ravinsharma7/go-safedesign/internal/pipeline"
	srcutil "github.com/ravinsharma7/go-safedesign/internal/source"

	"golang.org/x/mod/modfile"
)

const (
	moduleDiscoveryEntryModule     = "entry_module"
	moduleDiscoveryWorkspaceScan   = "workspace_scan"
	moduleDiscoveryGoWorkUse       = "go_work_use"
	moduleDiscoveryGoModReplace    = "go_mod_replace"
	moduleDiscoveryRequiredMissing = "required_missing"
	moduleDiscoveryExternalModule  = "external_module"
)

type discoveredModule struct {
	goModPath string
	reasons   map[string]bool
}

func (b *graphBuilder) indexModules() error {
	discovered, err := b.discoverModuleFiles()
	if err != nil {
		return err
	}

	moduleByPath := map[string]moduleInfo{}
	type parsedModule struct {
		info moduleInfo
		file *modfile.File
	}
	var parsed []parsedModule
	for _, module := range sortedDiscoveredModules(discovered) {
		src, err := os.ReadFile(module.goModPath)
		if err != nil {
			return err
		}
		mf, err := modfile.Parse(module.goModPath, src, nil)
		if err != nil {
			return err
		}
		if mf.Module == nil {
			continue
		}
		info := moduleInfo{
			Path:    mf.Module.Mod.Path,
			Dir:     filepath.Dir(module.goModPath),
			Reasons: sortedReasonKeys(module.reasons),
		}
		if existing, ok := moduleByPath[info.Path]; ok {
			info.Reasons = mergeReasons(existing.Reasons, info.Reasons)
		}
		moduleByPath[info.Path] = info
		parsed = append(parsed, parsedModule{info: info, file: mf})
	}

	b.modules = make([]moduleInfo, 0, len(moduleByPath))
	for _, info := range moduleByPath {
		b.modules = append(b.modules, info)
		b.addModuleNode(info)
	}
	sort.Slice(b.modules, func(i, j int) bool {
		if b.modules[i].Path != b.modules[j].Path {
			return b.modules[i].Path < b.modules[j].Path
		}
		return b.modules[i].Dir < b.modules[j].Dir
	})

	for _, module := range parsed {
		info := moduleByPath[module.info.Path]
		b.addModuleDependencies(info, module.file, moduleByPath)
	}
	return nil
}

func (b *graphBuilder) discoverModuleFiles() (map[string]*discoveredModule, error) {
	discovered := map[string]*discoveredModule{}
	addModule := func(path, reason string) {
		goModPath := filepath.Clean(path)
		module := discovered[goModPath]
		if module == nil {
			module = &discoveredModule{goModPath: goModPath, reasons: map[string]bool{}}
			discovered[goModPath] = module
		}
		module.reasons[reason] = true
	}

	addModule(filepath.Join(b.root, "go.mod"), moduleDiscoveryEntryModule)
	if filepath.Clean(b.workspaceRoot) != filepath.Clean(b.root) {
		if err := filepath.WalkDir(b.workspaceRoot, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Name() == "go.mod" {
				addModule(path, moduleDiscoveryWorkspaceScan)
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}

	if err := b.discoverGoWorkModules(addModule); err != nil {
		return nil, err
	}
	if err := b.discoverReplaceModules(discovered, addModule); err != nil {
		return nil, err
	}
	return discovered, nil
}

func (b *graphBuilder) discoverGoWorkModules(addModule func(path, reason string)) error {
	goWorkPath := filepath.Join(b.workspaceRoot, "go.work")
	src, err := os.ReadFile(goWorkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	wf, err := modfile.ParseWork(goWorkPath, src, nil)
	if err != nil {
		return err
	}
	for _, use := range wf.Use {
		dir := use.Path
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(filepath.Dir(goWorkPath), dir)
		}
		goModPath := filepath.Join(filepath.Clean(dir), "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			addModule(goModPath, moduleDiscoveryGoWorkUse)
		} else if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (b *graphBuilder) discoverReplaceModules(discovered map[string]*discoveredModule, addModule func(path, reason string)) error {
	visited := map[string]bool{}
	for {
		changed := false
		for _, module := range sortedDiscoveredModules(discovered) {
			if visited[module.goModPath] {
				continue
			}
			visited[module.goModPath] = true
			src, err := os.ReadFile(module.goModPath)
			if err != nil {
				return err
			}
			mf, err := modfile.Parse(module.goModPath, src, nil)
			if err != nil {
				return err
			}
			for _, replacement := range mf.Replace {
				if replacement.New.Path == "" || replacement.New.Version != "" || !localModulePath(replacement.New.Path) {
					continue
				}
				dir := replacement.New.Path
				if !filepath.IsAbs(dir) {
					dir = filepath.Join(filepath.Dir(module.goModPath), dir)
				}
				goModPath := filepath.Join(filepath.Clean(dir), "go.mod")
				if _, err := os.Stat(goModPath); err == nil {
					before := len(discovered)
					addModule(goModPath, moduleDiscoveryGoModReplace)
					if len(discovered) != before {
						changed = true
					}
					continue
				} else if err != nil && !os.IsNotExist(err) {
					return err
				}
				if info, err := os.Stat(filepath.Clean(dir)); err == nil && info.IsDir() {
					b.diagnostics = append(b.diagnostics, Diagnostic{
						Level:      "warning",
						Source:     b.rel(module.goModPath),
						Reason:     "local replace path has no go.mod: " + b.rel(dir),
						Stage:      string(pipeline.StageModuleExtraction),
						Status:     core.StatusWarning,
						TrustLevel: TrustSyntaxObserved,
					})
				}
			}
		}
		if !changed {
			break
		}
	}
	return nil
}

func (b *graphBuilder) addModuleNode(info moduleInfo) {
	src, _ := os.ReadFile(filepath.Join(info.Dir, "go.mod"))
	id := core.ModuleID(info.Path)
	b.addNode(Node{
		ID:         id,
		Kind:       core.NodeKindModule,
		Name:       info.Path,
		TrustLevel: TrustSyntaxObserved,
		Freshness:  core.FreshnessFresh,
		SourceFile: b.rel(filepath.Join(info.Dir, "go.mod")),
		SourceHash: srcutil.HashBytes(src),
		Reason:     strings.Join(info.Reasons, ","),
		ModulePath: info.Path,
	})
	b.addSourceRecord(b.sourceRecord("module_discovery", info.Dir, strings.Join(info.Reasons, ",")))
}

func (b *graphBuilder) addModuleDependencies(info moduleInfo, mf *modfile.File, moduleByPath map[string]moduleInfo) {
	id := core.ModuleID(info.Path)
	for _, req := range mf.Require {
		to := core.ModuleID(req.Mod.Path)
		if _, ok := moduleByPath[req.Mod.Path]; !ok {
			placeholderID := core.PlaceholderModuleID(req.Mod.Path)
			b.addNode(Node{
				ID:         placeholderID,
				Kind:       core.NodeKindPlaceholder,
				Name:       req.Mod.Path,
				TrustLevel: TrustSyntaxObserved,
				Synthetic:  true,
				Freshness:  core.FreshnessFresh,
				Reason:     moduleDiscoveryRequiredMissing,
				ModulePath: req.Mod.Path,
			})
			to = placeholderID
		}
		incomplete := core.IsPlaceholderID(to)
		b.addEdge(Edge{
			ID:         core.EdgeID(core.EdgeKindDependsOn, id, to),
			From:       id,
			To:         to,
			Kind:       core.EdgeKindDependsOn,
			TrustLevel: TrustSyntaxObserved,
			Complete:   !incomplete,
			Synthetic:  incomplete,
			Reason:     reasonIf(incomplete, "required_module_not_present_in_workspace"),
			SourceFile: b.rel(filepath.Join(info.Dir, "go.mod")),
		})
	}
}

func sortedDiscoveredModules(discovered map[string]*discoveredModule) []*discoveredModule {
	modules := make([]*discoveredModule, 0, len(discovered))
	for _, module := range discovered {
		modules = append(modules, module)
	}
	sort.Slice(modules, func(i, j int) bool { return modules[i].goModPath < modules[j].goModPath })
	return modules
}

func sortedReasonKeys(reasons map[string]bool) []string {
	keys := make([]string, 0, len(reasons))
	for reason := range reasons {
		keys = append(keys, reason)
	}
	sort.Strings(keys)
	return keys
}

func mergeReasons(left, right []string) []string {
	values := map[string]bool{}
	for _, reason := range left {
		values[reason] = true
	}
	for _, reason := range right {
		values[reason] = true
	}
	return sortedReasonKeys(values)
}

func localModulePath(path string) bool {
	return path == "." || strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") || filepath.IsAbs(path)
}
