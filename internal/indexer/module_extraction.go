package indexer

import (
	"os"
	"path/filepath"
	"strings"

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
	id := "module:" + info.Path
	b.addNode(Node{
		ID:         id,
		Kind:       "module",
		Name:       info.Path,
		TrustLevel: TrustSyntaxObserved,
		Freshness:  "fresh",
		SourceFile: b.rel(path),
		SourceHash: srcutil.HashBytes(src),
		ModulePath: info.Path,
	})
	for _, req := range mf.Require {
		to := "module:" + req.Mod.Path
		if !b.hasNode(to) {
			b.addNode(Node{
				ID:         "placeholder:module:" + req.Mod.Path,
				Kind:       "placeholder",
				Name:       req.Mod.Path,
				TrustLevel: TrustSyntaxObserved,
				Synthetic:  true,
				Freshness:  "fresh",
				Reason:     "module_required_but_not_discovered",
				ModulePath: req.Mod.Path,
			})
			to = "placeholder:module:" + req.Mod.Path
		}
		b.addEdge(Edge{
			ID:         "edge:depends_on:" + id + "->" + to,
			From:       id,
			To:         to,
			Kind:       "depends_on",
			TrustLevel: TrustSyntaxObserved,
			Complete:   !strings.HasPrefix(to, "placeholder:"),
			Synthetic:  strings.HasPrefix(to, "placeholder:"),
			Reason:     reasonIf(strings.HasPrefix(to, "placeholder:"), "required_module_not_present_in_workspace"),
			SourceFile: b.rel(path),
		})
	}
	return nil
}
