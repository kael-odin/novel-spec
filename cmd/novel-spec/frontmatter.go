package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

type novelTOML struct {
	SpecVersion string          `toml:"spec_version"`
	Book        map[string]any  `toml:"book"`
	Modules     map[string]bool `toml:"modules"`
}

func decodeNovelTOML(path string) (*novelTOML, error) {
	var b novelTOML
	if _, err := toml.DecodeFile(path, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

func parseFrontmatter(path string) (any, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := strings.TrimPrefix(string(body), string(rune(0xFEFF)))
	if !strings.HasPrefix(text, "---") {
		return nil, fmt.Errorf("no leading --- frontmatter fence")
	}
	rest := strings.TrimPrefix(text, "---")
	rest = strings.TrimPrefix(rest, "\r\n")
	rest = strings.TrimPrefix(rest, "\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, fmt.Errorf("missing closing --- frontmatter fence")
	}
	var value map[string]any
	if err := yaml.Unmarshal([]byte(rest[:end]), &value); err != nil {
		return nil, fmt.Errorf("YAML parse: %w", err)
	}
	return value, nil
}
