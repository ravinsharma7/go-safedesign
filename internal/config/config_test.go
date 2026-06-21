package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSharedConfigSections(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(`{"packageImportPolicy":{"rules":[]},"complexity":{"max":10}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	doc, found, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("config was not found")
	}
	if got := string(doc.Section("packageImportPolicy")); got != `{"rules":[]}` {
		t.Fatalf("packageImportPolicy section = %s", got)
	}
	if got := string(doc.Section("missing")); got != "" {
		t.Fatalf("missing section = %s", got)
	}
}

func TestResolvePathUsesExplicitOrFixtureDefault(t *testing.T) {
	if got := ResolvePath("/tmp/shop", "/tmp/custom.json"); got != "/tmp/custom.json" {
		t.Fatalf("explicit path = %s", got)
	}
	if got := ResolvePath("/tmp/shop", ""); got != filepath.Join("/tmp/shop", FileName) {
		t.Fatalf("default path = %s", got)
	}
}
