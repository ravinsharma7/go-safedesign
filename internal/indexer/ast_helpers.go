package indexer

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"

	srcutil "go-safedesign/internal/source"
)

func packagePath(mod moduleInfo, dir string) string {
	relDir, err := filepath.Rel(mod.Dir, dir)
	if err != nil || relDir == "." {
		return mod.Path
	}
	return mod.Path + "/" + filepath.ToSlash(relDir)
}

func (b *graphBuilder) rel(path string) string {
	return srcutil.WorkspaceRel(b.sourceBase, path)
}

func fileLineRange(fset *token.FileSet, file *ast.File) string {
	return lineRange(fset, file.Pos(), file.End())
}

func lineRange(fset *token.FileSet, start, end token.Pos) string {
	s := fset.Position(start)
	e := fset.Position(end)
	return fmt.Sprintf("%d:%d-%d:%d", s.Line, s.Column, e.Line, e.Column)
}

func receiverName(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.StarExpr:
		return "*" + receiverName(x.X)
	default:
		return "receiver"
	}
}

func callName(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		left := callName(x.X)
		if left == "" {
			return x.Sel.Name
		}
		return left + "." + x.Sel.Name
	default:
		return ""
	}
}

func isBuiltin(name string) bool {
	switch name {
	case "append", "cap", "close", "complex", "copy", "delete", "imag", "len", "make", "new", "panic", "print", "println", "real", "recover":
		return true
	default:
		return false
	}
}

func isStdlibImport(importPath string) bool {
	return !strings.Contains(importPath, ".")
}

func reasonIf(ok bool, reason string) string {
	if ok {
		return reason
	}
	return ""
}
