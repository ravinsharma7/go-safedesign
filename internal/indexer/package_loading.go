package indexer

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ravinsharma7/go-safedesign/internal/core"
	"github.com/ravinsharma7/go-safedesign/internal/pipeline"

	"golang.org/x/tools/go/packages"
)

func (b *graphBuilder) loadPackages() {
	restore := preferWorkingGoToolchain()
	defer restore()

	cfg := &packages.Config{Mode: packages.LoadAllSyntax, Dir: b.root, Env: append(os.Environ(), "PATH="+filepath.Join(runtime.GOROOT(), "bin")+string(os.PathListSeparator)+os.Getenv("PATH"), "GOROOT="+runtime.GOROOT(), "GOWORK=off", "GONOSUMDB=*", "GONOPROXY=*")}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		b.diagnostics = append(b.diagnostics, packageLoadingDiagnostic("go/packages", err.Error()))
		return
	}
	for _, pkg := range pkgs {
		id := core.PackageID(pkg.PkgPath)
		n := b.nodes[id]
		if n.ID == "" {
			n = Node{ID: id, Kind: core.NodeKindPackage, Name: pkg.PkgPath, Freshness: core.FreshnessFresh, PackagePath: pkg.PkgPath}
		}
		if len(pkg.Errors) == 0 {
			n.TrustLevel = TrustTypeResolved
			n.Reason = ""
		} else if n.TrustLevel == "" {
			n.TrustLevel = TrustSyntaxObserved
		}
		b.nodes[id] = n
		for _, e := range pkg.Errors {
			b.diagnostics = append(b.diagnostics, packageLoadingDiagnostic("go/packages:"+pkg.PkgPath, e.Msg))
		}
		for importPath, imported := range pkg.Imports {
			if imported == nil {
				continue
			}
			target := core.PackageID(importPath)
			n := b.nodes[target]
			if n.ID == "" {
				continue
			}
			if len(imported.Errors) == 0 {
				if core.TrustRank(n.TrustLevel) < core.TrustRank(TrustPackageLoaded) {
					n.TrustLevel = TrustPackageLoaded
				}
				b.nodes[target] = n
			}
		}
	}
}

func packageLoadingDiagnostic(source, reason string) Diagnostic {
	diagnostic := Diagnostic{
		Level:      "warning",
		Source:     source,
		Reason:     reason,
		Stage:      string(pipeline.StagePackageLoading),
		AnalyzerID: "indexer." + string(pipeline.StagePackageLoading),
		Status:     classifyPackageLoadingDiagnostic(reason),
	}
	if diagnostic.Status == core.DiagnosticStatusImportCycle {
		diagnostic.Level = "error"
	}
	return diagnostic
}

func classifyPackageLoadingDiagnostic(reason string) string {
	lower := strings.ToLower(reason)
	switch {
	case strings.Contains(lower, "import cycle"):
		return core.DiagnosticStatusImportCycle
	case strings.Contains(lower, "no required module provides package"),
		strings.Contains(lower, "cannot find module providing package"),
		strings.Contains(lower, "module found, but does not contain package"),
		strings.Contains(lower, "missing go.sum entry"):
		return core.DiagnosticStatusMissingDependency
	default:
		return core.DiagnosticStatusPackageLoadingDiagnostic
	}
}

func preferWorkingGoToolchain() func() {
	goroot := runtime.GOROOT()
	if _, err := os.Stat(filepath.Join(goroot, "bin", "go")); err != nil {
		return func() {}
	}
	oldPath, oldGOROOT := os.Getenv("PATH"), os.Getenv("GOROOT")
	_ = os.Setenv("PATH", filepath.Join(goroot, "bin")+string(os.PathListSeparator)+oldPath)
	_ = os.Setenv("GOROOT", goroot)
	return func() {
		_ = os.Setenv("PATH", oldPath)
		if oldGOROOT == "" {
			_ = os.Unsetenv("GOROOT")
			return
		}
		_ = os.Setenv("GOROOT", oldGOROOT)
	}
}
