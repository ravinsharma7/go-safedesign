package indexer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ravinsharma7/go-safedesign/internal/core"
	"github.com/ravinsharma7/go-safedesign/internal/pipeline"
	srcutil "github.com/ravinsharma7/go-safedesign/internal/source"
)

func (b *graphBuilder) indexGoFiles() error {
	for _, mod := range b.modules {
		if err := filepath.WalkDir(mod.Dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".go") {
				return nil
			}
			return b.indexGoFile(mod, path)
		}); err != nil {
			return err
		}
	}
	return nil
}

func (b *graphBuilder) indexGoFile(mod moduleInfo, path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	file, err := parser.ParseFile(b.fset, path, src, parser.ParseComments|parser.AllErrors)
	if err != nil {
		b.diagnostics = append(b.diagnostics, Diagnostic{Level: "error", Source: b.rel(path), Reason: err.Error()})
		return nil
	}

	pkgPath := packagePath(mod, filepath.Dir(path))
	pkgID := core.PackageID(pkgPath)
	sourceFile := b.rel(path)
	fileID := core.NodeKindFile + ":" + sourceFile
	fileHash := srcutil.HashBytes(src)
	b.syntaxSnapshots[sourceFile] = pipeline.SyntaxSnapshot{File: file, SourceHash: fileHash}

	b.addNode(Node{ID: pkgID, Kind: core.NodeKindPackage, Name: pkgPath, TrustLevel: TrustSyntaxObserved, Freshness: core.FreshnessFresh, PackagePath: pkgPath, ModulePath: mod.Path})
	b.addNode(Node{ID: fileID, Kind: core.NodeKindFile, Name: filepath.Base(path), TrustLevel: TrustSyntaxObserved, Freshness: core.FreshnessFresh, SourceFile: sourceFile, SourceHash: fileHash, LineRange: fileLineRange(b.fset, file), PackagePath: pkgPath, ModulePath: mod.Path})
	b.addEdge(Edge{ID: core.EdgeID(core.EdgeKindContains, pkgID, fileID), From: pkgID, To: fileID, Kind: core.EdgeKindContains, TrustLevel: TrustSyntaxObserved, Complete: true, SourceFile: sourceFile})

	for _, imp := range file.Imports {
		importPath, _ := strconv.Unquote(imp.Path.Value)
		importID := core.NodeKindImport + ":" + pkgPath + ":" + importPath
		targetID := b.importTargetID(importPath)
		incomplete := core.IsPlaceholderID(targetID)
		line := lineRange(b.fset, imp.Pos(), imp.End())
		b.addNode(Node{ID: importID, Kind: core.NodeKindImport, Name: importPath, TrustLevel: TrustSyntaxObserved, Freshness: core.FreshnessFresh, SourceFile: b.rel(path), SourceHash: fileHash, LineRange: line, PackagePath: pkgPath, ModulePath: mod.Path})
		b.addEdge(Edge{ID: core.EdgeID(core.EdgeKindImports, pkgID, targetID), From: pkgID, To: targetID, Kind: core.EdgeKindImports, TrustLevel: TrustSyntaxObserved, Synthetic: incomplete, Complete: !incomplete, Reason: reasonIf(incomplete, "import_target_not_parsed_or_loaded"), SourceFile: b.rel(path), LineRange: line})
	}

	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			b.indexFunc(path, fileHash, pkgPath, mod.Path, x)
		case *ast.TypeSpec:
			b.indexType(path, fileHash, pkgPath, mod.Path, x)
		case *ast.GoStmt:
			b.indexRuntimeMarker(path, fileHash, pkgPath, "go_statement", x.Pos(), x.End())
		case *ast.DeferStmt:
			b.indexRuntimeMarker(path, fileHash, pkgPath, "defer_statement", x.Pos(), x.End())
		case *ast.SendStmt:
			b.indexRuntimeMarker(path, fileHash, pkgPath, "channel_send", x.Pos(), x.End())
		case *ast.UnaryExpr:
			if x.Op == token.ARROW {
				b.indexRuntimeMarker(path, fileHash, pkgPath, "channel_receive", x.Pos(), x.End())
			}
		case *ast.SelectStmt:
			b.indexRuntimeMarker(path, fileHash, pkgPath, "select_statement", x.Pos(), x.End())
		case *ast.CallExpr:
			b.indexCall(path, pkgPath, x)
		}
		return true
	})
	return nil
}

func (b *graphBuilder) indexType(path, fileHash, pkgPath, modPath string, spec *ast.TypeSpec) {
	kind := core.NodeKindType
	switch spec.Type.(type) {
	case *ast.InterfaceType:
		kind = core.NodeKindInterface
	case *ast.StructType:
		kind = core.NodeKindStruct
	}
	id := kind + ":" + pkgPath + "." + spec.Name.Name
	line := lineRange(b.fset, spec.Pos(), spec.End())
	b.addNode(Node{ID: id, Kind: kind, Name: spec.Name.Name, TrustLevel: TrustSyntaxObserved, Freshness: core.FreshnessFresh, SourceFile: b.rel(path), SourceHash: fileHash, LineRange: line, PackagePath: pkgPath, ModulePath: modPath})
	b.addEdge(Edge{ID: core.EdgeID(core.EdgeKindDeclares, core.PackageID(pkgPath), id), From: core.PackageID(pkgPath), To: id, Kind: core.EdgeKindDeclares, TrustLevel: TrustSyntaxObserved, Complete: true, SourceFile: b.rel(path), LineRange: line})

	structType, ok := spec.Type.(*ast.StructType)
	if !ok || structType.Fields == nil {
		return
	}
	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			fieldLine := lineRange(b.fset, name.Pos(), name.End())
			fieldID := core.NodeKindField + ":" + pkgPath + "." + spec.Name.Name + "." + name.Name
			b.addNode(Node{ID: fieldID, Kind: core.NodeKindField, Name: name.Name, TrustLevel: TrustSyntaxObserved, Freshness: core.FreshnessFresh, SourceFile: b.rel(path), SourceHash: fileHash, LineRange: fieldLine, PackagePath: pkgPath, ModulePath: modPath})
			b.addEdge(Edge{ID: core.EdgeID(core.EdgeKindContains, id, fieldID), From: id, To: fieldID, Kind: core.EdgeKindContains, TrustLevel: TrustSyntaxObserved, Complete: true, SourceFile: b.rel(path), LineRange: fieldLine})
		}
	}
}

func (b *graphBuilder) indexFunc(path, fileHash, pkgPath, modPath string, fn *ast.FuncDecl) {
	id := core.NodeKindFunction + ":" + pkgPath + "." + fn.Name.Name
	kind := core.NodeKindFunction
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		id = core.NodeKindMethod + ":" + pkgPath + "." + receiverName(fn.Recv.List[0].Type) + "." + fn.Name.Name
		kind = core.NodeKindMethod
	}
	b.addNode(Node{ID: id, Kind: kind, Name: fn.Name.Name, TrustLevel: TrustSyntaxObserved, Freshness: core.FreshnessFresh, SourceFile: b.rel(path), SourceHash: fileHash, LineRange: lineRange(b.fset, fn.Pos(), fn.End()), PackagePath: pkgPath, ModulePath: modPath})
	b.addEdge(Edge{ID: core.EdgeID(core.EdgeKindDeclares, core.PackageID(pkgPath), id), From: core.PackageID(pkgPath), To: id, Kind: core.EdgeKindDeclares, TrustLevel: TrustSyntaxObserved, Complete: true, SourceFile: b.rel(path), LineRange: lineRange(b.fset, fn.Pos(), fn.End())})
}

func (b *graphBuilder) indexRuntimeMarker(path, fileHash, pkgPath, name string, start, end token.Pos) {
	line := lineRange(b.fset, start, end)
	id := core.NodeKindRuntimeMarker + ":" + b.rel(path) + ":" + line + ":" + name
	fileID := core.NodeKindFile + ":" + b.rel(path)
	b.addNode(Node{ID: id, Kind: core.NodeKindRuntimeMarker, Name: name, TrustLevel: TrustSyntaxObserved, Freshness: core.FreshnessFresh, SourceFile: b.rel(path), SourceHash: fileHash, LineRange: line, PackagePath: pkgPath})
	b.addEdge(Edge{ID: core.EdgeID(core.EdgeKindContainsRuntimeMarker, fileID, id), From: fileID, To: id, Kind: core.EdgeKindContainsRuntimeMarker, TrustLevel: TrustSyntaxObserved, Complete: true, SourceFile: b.rel(path), LineRange: line})
}

func (b *graphBuilder) indexCall(path, pkgPath string, call *ast.CallExpr) {
	name := callName(call.Fun)
	if name == "" || isBuiltin(name) {
		return
	}
	line := lineRange(b.fset, call.Pos(), call.End())
	id := core.NodeKindUnresolvedCall + ":" + b.rel(path) + ":" + line + ":" + name
	b.addNode(Node{ID: id, Kind: core.NodeKindUnresolvedCall, Name: name, TrustLevel: TrustSyntaxObserved, Synthetic: true, Freshness: core.FreshnessFresh, Reason: "call_target_not_type_resolved", SourceFile: b.rel(path), LineRange: line, PackagePath: pkgPath})
	b.addEdge(Edge{ID: core.EdgeID(core.EdgeKindCalls, core.PackageID(pkgPath), id), From: core.PackageID(pkgPath), To: id, Kind: core.EdgeKindCalls, TrustLevel: TrustSyntaxObserved, Synthetic: true, Complete: false, Reason: "target_symbol_not_type_resolved", SourceFile: b.rel(path), LineRange: line})
}

func (b *graphBuilder) importTargetID(importPath string) string {
	if isStdlibImport(importPath) {
		return core.PackageID(importPath)
	}
	for _, mod := range b.modules {
		if importPath == mod.Path || strings.HasPrefix(importPath, mod.Path+"/") {
			id := core.PackageID(importPath)
			if !b.hasNode(id) {
				placeholderID := core.PlaceholderPackageID(importPath)
				b.addNode(Node{ID: placeholderID, Kind: core.NodeKindPlaceholder, Name: importPath, TrustLevel: TrustSyntaxObserved, Synthetic: true, Freshness: core.FreshnessFresh, Reason: "package_imported_but_no_go_files_discovered", PackagePath: importPath, ModulePath: mod.Path})
				return placeholderID
			}
			return id
		}
	}
	placeholderID := core.PlaceholderPackageID(importPath)
	b.addNode(Node{ID: placeholderID, Kind: core.NodeKindPlaceholder, Name: importPath, TrustLevel: TrustSyntaxObserved, Synthetic: true, Freshness: core.FreshnessFresh, Reason: "imported_package_outside_known_workspace", PackagePath: importPath})
	return placeholderID
}
