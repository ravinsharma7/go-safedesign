package indexer

import (
	"os"
	"path/filepath"
	"strings"

	"go-safedesign/internal/core"
	srcutil "go-safedesign/internal/source"

	"golang.org/x/mod/modfile"
)

func (b *graphBuilder) indexModules() error {
	if filepath.Clean(b.workspaceRoot) == filepath.Clean(b.root) {
		return b.indexModuleFile(filepath.Join(b.root, "go.mod"))
	}
	return filepath.WalkDir(b.workspaceRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "go.mod" {
			return nil
		}
		return b.indexModuleFile(path)
	})
}

func (b *graphBuilder) indexModuleFile(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	mf, err := modfile.Parse(path, src, nil)
	if err != nil {
		return err
	}
	if mf.Module == nil {
		return nil
	}
	dir := filepath.Dir(path)
	info := moduleInfo{Path: mf.Module.Mod.Path, Dir: dir}
	b.modules = append(b.modules, info)
	id := core.ModuleID(info.Path)
	b.addNode(Node{
		ID:         id,
		Kind:       core.NodeKindModule,
		Name:       info.Path,
		TrustLevel: TrustSyntaxObserved,
		Freshness:  core.FreshnessFresh,
		SourceFile: b.rel(path),
		SourceHash: srcutil.HashBytes(src),
		ModulePath: info.Path,
	})
	for _, req := range mf.Require {
		to := core.ModuleID(req.Mod.Path)
		if !b.hasNode(to) {
			placeholderID := core.PlaceholderModuleID(req.Mod.Path)
			b.addNode(Node{
				ID:         placeholderID,
				Kind:       core.NodeKindPlaceholder,
				Name:       req.Mod.Path,
				TrustLevel: TrustSyntaxObserved,
				Synthetic:  true,
				Freshness:  core.FreshnessFresh,
				Reason:     "module_required_but_not_discovered",
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
			SourceFile: b.rel(path),
		})
	}
	return nil
}
