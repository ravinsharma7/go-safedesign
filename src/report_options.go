package main

import "strings"

type compactReportOptions struct {
	Limit        int
	ScopeModule  string
	ScopePackage string
	ScopeFile    string
}

func (options compactReportOptions) normalized() compactReportOptions {
	if options.Limit < 1 {
		options.Limit = 1
	}
	options.ScopeModule = strings.TrimSpace(options.ScopeModule)
	options.ScopePackage = strings.TrimSpace(options.ScopePackage)
	options.ScopeFile = strings.TrimSpace(options.ScopeFile)
	return options
}

func splitCommaList(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
