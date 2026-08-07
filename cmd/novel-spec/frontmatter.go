package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

type novelTOML struct {
	SpecVersion string                 `toml:"spec_version"`
	Book        map[string]any         `toml:"book"`
	Modules     map[string]bool        `toml:"modules"`
}

func decodeNovelTOML(path string) (*novelTOML, error) {
	var b novelTOML
	if _, err := toml.DecodeFile(path, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

// parseFrontmatter reads a Markdown chapter file, extracts the YAML frontmatter
// block (between leading --- fences), and returns it as a generic Go value for
// schema validation. The body after the closing fence is ignored.
//
// We parse a minimal YAML subset (key: value, lists via [a, b] or indented -
// items) ourselves to stay dependency-light, matching Reasonix's lean-dependency
// philosophy. For v0.1 the frontmatter is flat key/value with simple list
// literals, so a full YAML parser is unnecessary.
func parseFrontmatter(path string) (any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	body := strings.TrimSpace(string(b))
	if !strings.HasPrefix(body, "---") {
		return nil, fmt.Errorf("no leading --- frontmatter fence")
	}
	// Drop the opening fence.
	rest := strings.TrimPrefix(body, "---")
	rest = strings.TrimPrefix(rest, "\r\n")
	rest = strings.TrimPrefix(rest, "\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, fmt.Errorf("missing closing --- frontmatter fence")
	}
	fmText := rest[:end]
	return parseSimpleYAML(fmText), nil
}

// parseSimpleYAML parses the minimal frontmatter subset we use: `key: value`,
// `key: [a, b, c]`, and `key:` followed by indented `- item` lines. Values are
// coerced to string / int / bool / []any. Unknown shapes fall back to string.
func parseSimpleYAML(text string) any {
	result := map[string]any{}
	lines := strings.Split(text, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		colon := strings.Index(trimmed, ":")
		if colon < 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:colon])
		val := strings.TrimSpace(trimmed[colon+1:])
		if val == "" {
			// Possibly a block list: following indented "- ..." lines.
			items, advanced := parseBlockList(lines, i+1)
			if advanced > 0 {
				result[key] = items
				i += advanced
			} else {
				result[key] = []any{}
			}
			continue
		}
		result[key] = parseScalar(val)
	}
	return result
}

// parseBlockList collects consecutive "- item" lines starting at lineIdx.
// Returns the list and how many lines it consumed.
func parseBlockList(lines []string, lineIdx int) ([]any, int) {
	var items []any
	consumed := 0
	for j := lineIdx; j < len(lines); j++ {
		t := strings.TrimSpace(lines[j])
		if !strings.HasPrefix(t, "-") {
			break
		}
		item := strings.TrimSpace(strings.TrimPrefix(t, "-"))
		items = append(items, parseScalar(item))
		consumed++
	}
	return items, consumed
}

// parseScalar coerces a literal to bool / int / string, or parses an inline
// [a, b, c] list. Quoted strings have their quotes stripped.
func parseScalar(s string) any {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		inner := strings.TrimSuffix(strings.TrimPrefix(s, "["), "]")
		if inner == "" {
			return []any{}
		}
		parts := strings.Split(inner, ",")
		out := make([]any, len(parts))
		for i, p := range parts {
			out[i] = parseScalar(p)
		}
		return out
	}
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	switch s {
	case "true":
		return true
	case "false":
		return false
	}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err == nil && s == fmt.Sprintf("%d", n) {
		return n
	}
	return s
}
