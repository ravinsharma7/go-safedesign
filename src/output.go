package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const defaultOutputDir = "tmp/report"

func emitJSONOutput(name string, value any, stdout bool, outputDir string) error {
	if stdout {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(value)
	}
	if outputDir == "" {
		outputDir = defaultOutputDir
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(outputDir, name)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", path)
	return nil
}
