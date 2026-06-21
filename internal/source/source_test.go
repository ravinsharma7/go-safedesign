package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLineRange(t *testing.T) {
	start, end, err := ParseLineRange("9:1-17:2")
	if err != nil {
		t.Fatal(err)
	}
	if start != 9 || end != 17 {
		t.Fatalf("range = %d-%d, want 9-17", start, end)
	}
}

func TestReadBlockRejectsPathEscape(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadBlock(dir, "../outside.go", "1:1-1:1")
	if err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("error = %v, want escape error", err)
	}
}

func TestReadBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workspace", "shop", "order", "service.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package order\n\nfunc PlaceOrder() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	block, err := ReadBlock(dir, "workspace/shop/order/service.go", "3:1-3:21")
	if err != nil {
		t.Fatal(err)
	}
	if block.StartLine != 3 || block.EndLine != 3 || !strings.Contains(block.Code, "func PlaceOrder") {
		t.Fatalf("block = %#v", block)
	}
}

func TestWorkspaceRelUsesExplicitSourceBase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pkg", "service.go")
	if got := WorkspaceRel(dir, path); got != "pkg/service.go" {
		t.Fatalf("relative path = %q, want pkg/service.go", got)
	}
}
