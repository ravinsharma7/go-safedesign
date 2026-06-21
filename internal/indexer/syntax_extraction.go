package indexer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go-safedesign/internal/pipeline"
	srcutil "go-safedesign/internal/source"
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
	pkgID := "package:" + pkgPath
	sourceFile := b.rel(path)
	fileID := "file:" + sourceFile
	fileHash := srcutil.HashBytes(src)
	b.syntaxSnapshots[sourceFile] = pipeline.SyntaxSnapshot{File: file, SourceHash: fileHash}

	b.addNode(Node{ID: pkgID, Kind: "package", Name: pkgPath, TrustLevel: TrustSyntaxObserved, Freshness: "fresh", PackagePath: pkgPath, ModulePath: mod.Path})
	b.addNode(Node{ID: fileID, Kind: "file", Name: filepath.Base(path), TrustLevel: TrustSyntaxObserved, Freshness: "fresh", SourceFile: sourceFile, SourceHash: fileHash, LineRange: fileLineRange(b.fset, file), PackagePath: pkgPath, ModulePath: mod.Path})
	b.addEdge(Edge{ID: "edge:contains:" + pkgID + "->" + fileID, From: pkgID, To: fileID, Kind: "contains", TrustLevel: TrustSyntaxObserved, Complete: true, SourceFile: sourceFile})

	for _, imp := range file.Imports {
		importPath, _ := strconv.Unquote(imp.Path.Value)
		importID := "import:" + pkgPath + ":" + importPath
		targetID := b.importTargetID(importPath)
		line := lineRange(b.fset, imp.Pos(), imp.End())
		b.addNode(Node{ID: importID, Kind: "import", Name: importPath, TrustLevel: TrustSyntaxObserved, Freshness: "fresh", SourceFile: b.rel(path), SourceHash: fileHash, LineRange: line, PackagePath: pkgPath, ModulePath: mod.Path})
		b.addEdge(Edge{ID: "edge:imports:" + pkgID + "->" + targetID, From: pkgID, To: targetID, Kind: "imports", TrustLevel: TrustSyntaxObserved, Synthetic: strings.HasPrefix(targetID, "placeholder:"), Complete: !strings.HasPrefix(targetID, "placeholder:"), Reason: reasonIf(strings.HasPrefix(targetID, "placeholder:"), "import_target_not_parsed_or_loaded"), SourceFile: b.rel(path), LineRange: line})
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
	kind := "type"
	switch spec.Type.(type) {
	case *ast.InterfaceType:
		kind = "interface"
	case *ast.StructType:
		kind = "struct"
	}
	id := kind + ":" + pkgPath + "." + spec.Name.Name
	line := lineRange(b.fset, spec.Pos(), spec.End())
	b.addNode(Node{ID: id, Kind: kind, Name: spec.Name.Name, TrustLevel: TrustSyntaxObserved, Freshness: "fresh", SourceFile: b.rel(path), SourceHash: fileHash, LineRange: line, PackagePath: pkgPath, ModulePath: modPath})
	b.addEdge(Edge{ID: "edge:declares:package:" + pkgPath + "->" + id, From: "package:" + pkgPath, To: id, Kind: "declares", TrustLevel: TrustSyntaxObserved, Complete: true, SourceFile: b.rel(path), LineRange: line})

	structType, ok := spec.Type.(*ast.StructType)
	if !ok || structType.Fields == nil {
		return
	}
	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			fieldLine := lineRange(b.fset, name.Pos(), name.End())
			fieldID := "field:" + pkgPath + "." + spec.Name.Name + "." + name.Name
			b.addNode(Node{ID: fieldID, Kind: "field", Name: name.Name, TrustLevel: TrustSyntaxObserved, Freshness: "fresh", SourceFile: b.rel(path), SourceHash: fileHash, LineRange: fieldLine, PackagePath: pkgPath, ModulePath: modPath})
			b.addEdge(Edge{ID: "edge:contains:" + id + "->" + fieldID, From: id, To: fieldID, Kind: "contains", TrustLevel: TrustSyntaxObserved, Complete: true, SourceFile: b.rel(path), LineRange: fieldLine})
		}
	}
}

func (b *graphBuilder) indexFunc(path, fileHash, pkgPath, modPath string, fn *ast.FuncDecl) {
	id := "function:" + pkgPath + "." + fn.Name.Name
	kind := "function"
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		id = "method:" + pkgPath + "." + receiverName(fn.Recv.List[0].Type) + "." + fn.Name.Name
		kind = "method"
	}
	b.addNode(Node{ID: id, Kind: kind, Name: fn.Name.Name, TrustLevel: TrustSyntaxObserved, Freshness: "fresh", SourceFile: b.rel(path), SourceHash: fileHash, LineRange: lineRange(b.fset, fn.Pos(), fn.End()), PackagePath: pkgPath, ModulePath: modPath})
	b.addEdge(Edge{ID: "edge:declares:package:" + pkgPath + "->" + id, From: "package:" + pkgPath, To: id, Kind: "declares", TrustLevel: TrustSyntaxObserved, Complete: true, SourceFile: b.rel(path), LineRange: lineRange(b.fset, fn.Pos(), fn.End())})
}

func (b *graphBuilder) indexRuntimeMarker(path, fileHash, pkgPath, name string, start, end token.Pos) {
	line := lineRange(b.fset, start, end)
	id := "runtime_marker:" + b.rel(path) + ":" + line + ":" + name
	b.addNode(Node{ID: id, Kind: "runtime_marker", Name: name, TrustLevel: TrustSyntaxObserved, Freshness: "fresh", SourceFile: b.rel(path), SourceHash: fileHash, LineRange: line, PackagePath: pkgPath})
	b.addEdge(Edge{ID: "edge:contains_runtime_marker:file:" + b.rel(path) + "->" + id, From: "file:" + b.rel(path), To: id, Kind: "contains_runtime_marker", TrustLevel: TrustSyntaxObserved, Complete: true, SourceFile: b.rel(path), LineRange: line})
}

func (b *graphBuilder) indexCall(path, pkgPath string, call *ast.CallExpr) {
	name := callName(call.Fun)
	if name == "" || isBuiltin(name) {
		return
	}
	line := lineRange(b.fset, call.Pos(), call.End())
	id := "unresolved_call:" + b.rel(path) + ":" + line + ":" + name
	b.addNode(Node{ID: id, Kind: "unresolved_call", Name: name, TrustLevel: TrustSyntaxObserved, Synthetic: true, Freshness: "fresh", Reason: "call_target_not_type_resolved", SourceFile: b.rel(path), LineRange: line, PackagePath: pkgPath})
	b.addEdge(Edge{ID: "edge:calls:package:" + pkgPath + "->" + id, From: "package:" + pkgPath, To: id, Kind: "calls", TrustLevel: TrustSyntaxObserved, Synthetic: true, Complete: false, Reason: "target_symbol_not_type_resolved", SourceFile: b.rel(path), LineRange: line})
}

func (b *graphBuilder) importTargetID(importPath string) string {
	if isStdlibImport(importPath) {
		return "package:" + importPath
	}
	for _, mod := range b.modules {
		if importPath == mod.Path || strings.HasPrefix(importPath, mod.Path+"/") {
			id := "package:" + importPath
			if !b.hasNode(id) {
				b.addNode(Node{ID: "placeholder:package:" + importPath, Kind: "placeholder", Name: importPath, TrustLevel: TrustSyntaxObserved, Synthetic: true, Freshness: "fresh", Reason: "package_imported_but_no_go_files_discovered", PackagePath: importPath, ModulePath: mod.Path})
				return "placeholder:package:" + importPath
			}
			return id
		}
	}
	b.addNode(Node{ID: "placeholder:package:" + importPath, Kind: "placeholder", Name: importPath, TrustLevel: TrustSyntaxObserved, Synthetic: true, Freshness: "fresh", Reason: "imported_package_outside_known_workspace", PackagePath: importPath})
	return "placeholder:package:" + importPath
}
