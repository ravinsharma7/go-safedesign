package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEmitJSONOutputWritesToConfiguredDirectory(t *testing.T) {
	dir := t.TempDir()

	if err := emitJSONOutput("report.json", map[string]string{"status": "ok"}, false, dir); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["status"] != "ok" {
		t.Fatalf("report = %#v", got)
	}
}
