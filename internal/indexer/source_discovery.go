package indexer

import (
	"os"
	"path/filepath"
	"strings"

	"go-safedesign/internal/core"
	srcutil "go-safedesign/internal/source"
)

func (b *graphBuilder) discoverSources() error {
	b.addSourceRecord(b.sourceRecord("user_entry", b.entryPath, "user_provided_path"))
	b.addSourceRecord(b.sourceRecord("project_root", b.root, "resolved_project_root"))
	b.addSourceRecord(b.sourceRecord("workspace_root", b.workspaceRoot, "resolved_workspace_root"))
	if b.configPath != "" {
		if _, err := os.Stat(b.configPath); err == nil {
			b.addSourceRecord(b.sourceRecord("config", b.configPath, "configuration_file"))
		} else if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	goWork := filepath.Join(b.workspaceRoot, "go.work")
	if _, err := os.Stat(goWork); err == nil {
		b.addSourceRecord(b.sourceRecord("go_work", goWork, "workspace_file"))
	} else if err != nil && !os.IsNotExist(err) {
		return err
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
		switch d.Name() {
		case "go.mod":
			b.addSourceRecord(b.sourceRecord("go_mod", path, "module_file"))
		default:
			if strings.HasSuffix(d.Name(), ".go") {
				b.addSourceRecord(b.sourceRecord("go_file", path, "go_source_file"))
			}
		}
		return nil
	})
}

func (b *graphBuilder) sourceRecord(kind, path, reason string) core.SourceRecord {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	rel := b.rel(abs)
	hash := ""
	if info, err := os.Stat(abs); err == nil && !info.IsDir() {
		if src, err := os.ReadFile(abs); err == nil {
			hash = srcutil.HashBytes(src)
		}
	}
	return core.SourceRecord{
		ID:         "source_record:" + kind + ":" + rel,
		Kind:       kind,
		Path:       rel,
		AbsPath:    filepath.Clean(abs),
		SourceHash: hash,
		Freshness:  "fresh",
		Reason:     reason,
		TrustLevel: core.TrustSyntaxObserved,
	}
}

func (b *graphBuilder) addSourceRecord(record core.SourceRecord) {
	record.FactMetadata = b.metadataForCurrentRun()
	b.sourceRecords = append(b.sourceRecords, record)
	b.freshness = append(b.freshness, Freshness{
		FactID:       record.ID,
		SourceFile:   record.Path,
		NewHash:      record.SourceHash,
		Status:       "fresh",
		Reason:       "source_discovered",
		Extractor:    core.ExtractorVersion,
		FactMetadata: record.FactMetadata,
	})
}
