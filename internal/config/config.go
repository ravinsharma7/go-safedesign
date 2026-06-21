package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const FileName = "safedesign.config.json"

type Document struct {
	Path     string
	Raw      []byte
	Sections map[string]json.RawMessage
}

func DefaultPath(fixtureRoot string) string {
	if fixtureRoot == "" {
		return ""
	}
	return filepath.Join(fixtureRoot, FileName)
}

func ResolvePath(fixtureRoot, explicitPath string) string {
	if explicitPath != "" {
		return explicitPath
	}
	return DefaultPath(fixtureRoot)
}

func Load(path string) (Document, bool, error) {
	if path == "" {
		return Document{}, false, nil
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Document{}, false, nil
		}
		return Document{}, false, err
	}
	if info.IsDir() {
		return Document{}, false, fmt.Errorf("config path is a directory: %s", path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Document{}, false, err
	}
	sections := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &sections); err != nil {
		return Document{}, false, err
	}
	return Document{Path: path, Raw: raw, Sections: sections}, true, nil
}

func (d Document) Section(name string) []byte {
	raw := d.Sections[name]
	if len(raw) == 0 {
		return nil
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	return out
}
