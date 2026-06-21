package source

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Block struct {
	File      string `json:"file"`
	LineRange string `json:"lineRange"`
	StartLine int    `json:"startLine"`
	EndLine   int    `json:"endLine"`
	Code      string `json:"code"`
}

func HashBytes(src []byte) string {
	sum := sha256.Sum256(src)
	return hex.EncodeToString(sum[:])[:16]
}

func WorkspaceRel(sourceBase, path string) string {
	r, err := filepath.Rel(filepath.Clean(sourceBase), path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(r)
}

func ReadBlock(sourceBase, sourceFile, lineRange string) (Block, error) {
	if sourceFile == "" {
		return Block{}, fmt.Errorf("missing source file")
	}
	startLine, endLine, err := ParseLineRange(lineRange)
	if err != nil {
		return Block{}, err
	}
	cleanSource := filepath.Clean(filepath.FromSlash(sourceFile))
	fullPath := filepath.Clean(filepath.Join(sourceBase, cleanSource))
	cleanBase := filepath.Clean(sourceBase)
	if fullPath != cleanBase && !strings.HasPrefix(fullPath, cleanBase+string(os.PathSeparator)) {
		return Block{}, fmt.Errorf("source file escapes source base")
	}
	src, err := os.ReadFile(fullPath)
	if err != nil {
		return Block{}, err
	}
	lines := strings.Split(string(src), "\n")
	if startLine < 1 {
		startLine = 1
	}
	if endLine < startLine {
		endLine = startLine
	}
	if startLine > len(lines) {
		startLine = len(lines)
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	var b strings.Builder
	for i := startLine; i <= endLine; i++ {
		fmt.Fprintf(&b, "%4d  %s", i, lines[i-1])
		if i != endLine {
			b.WriteByte('\n')
		}
	}
	return Block{File: filepath.ToSlash(cleanSource), LineRange: lineRange, StartLine: startLine, EndLine: endLine, Code: b.String()}, nil
}

func ParseLineRange(lineRange string) (int, int, error) {
	if lineRange == "" {
		return 1, 1, nil
	}
	parts := strings.Split(lineRange, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid line range %q", lineRange)
	}
	startParts := strings.Split(parts[0], ":")
	endParts := strings.Split(parts[1], ":")
	if len(startParts) == 0 || len(endParts) == 0 {
		return 0, 0, fmt.Errorf("invalid line range %q", lineRange)
	}
	startLine, err := strconv.Atoi(startParts[0])
	if err != nil {
		return 0, 0, err
	}
	endLine, err := strconv.Atoi(endParts[0])
	if err != nil {
		return 0, 0, err
	}
	return startLine, endLine, nil
}
