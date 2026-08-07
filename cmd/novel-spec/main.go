// Package main implements the novel-spec validation CLI.
//
// Usage:
//
//	novel-spec check [path]   validate a novel repository against novel-spec v0.1
//	novel-spec version        print the spec version this CLI enforces
//
// A repository is conformant if it satisfies SPEC.md §Conformance. Exit code is
// 0 when the repo validates cleanly, 1 on any violation.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const specVersion = "0.1"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "check":
		cmdCheck(os.Args[2:])
	case "version":
		fmt.Println("novel-spec", specVersion)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: novel-spec <check|version> [path]")
}

func cmdCheck(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	fs.Usage = func() { fmt.Fprintln(os.Stderr, "usage: novel-spec check [path]"); fs.PrintDefaults() }
	_ = fs.Parse(args)

	root := "."
	if fs.NArg() > 0 {
		root = fs.Arg(0)
	}
	exeDir, _ := os.Executable()
	exeDir = filepath.Dir(exeDir)
	v := &validator{root: root, schemaDir: resolveSchemaDir(exeDir, root)}
	v.run()
	v.report()
	if v.failed {
		os.Exit(1)
	}
}

// resolveSchemaDir finds the schema/ directory: next to the binary, or under the
// repo root being checked (so `go run ./cmd/novel-spec` works from the repo).
func resolveSchemaDir(exeDir, root string) string {
	for _, c := range []string{
		filepath.Join(exeDir, "schema"),
		filepath.Join(root, "schema"),
		filepath.Join(root, "..", "schema"),
		filepath.Join("schema"),
	} {
		if exists(filepath.Join(c, "character.json")) {
			return c
		}
	}
	return filepath.Join(root, "schema")
}

type violation struct{ path, msg string }

type validator struct {
	root       string
	schemaDir  string
	violations []violation
	failed     bool
}

func (v *validator) add(path, msg string) {
	v.violations = append(v.violations, violation{path, msg})
	v.failed = true
}

func (v *validator) run() {
	// 1. NOVEL.toml exists, parses, and validates.
	tomlPath := filepath.Join(v.root, "NOVEL.toml")
	if !exists(tomlPath) {
		v.add("NOVEL.toml", "missing (required by novel-spec v0.1)")
		return
	}
	book, err := decodeNovelTOML(tomlPath)
	if err != nil {
		v.add("NOVEL.toml", "parse error: "+err.Error())
		return
	}
	if book.SpecVersion != specVersion {
		v.add("NOVEL.toml", fmt.Sprintf("spec_version = %q; this CLI enforces %q", book.SpecVersion, specVersion))
	}
	if err := v.validateFile("novel.toml.json", tomlPath, true); err != nil {
		v.add("NOVEL.toml", err.Error())
	}

	// 2. .novel/state/ with at least world.json and characters/_index.json.
	stateDir := filepath.Join(v.root, ".novel", "state")
	if !exists(stateDir) {
		v.add(".novel/state/", "missing (principle 1: state is the single source of truth)")
	} else {
		if !exists(filepath.Join(stateDir, "world.json")) {
			v.add(".novel/state/world.json", "missing")
		}
		if !exists(filepath.Join(stateDir, "characters", "_index.json")) {
			v.add(".novel/state/characters/_index.json", "missing")
		}
	}

	// 3. Character files validate against character.json.
	charDir := filepath.Join(stateDir, "characters")
	if exists(charDir) {
		entries, _ := os.ReadDir(charDir)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || e.Name() == "_index.json" {
				continue
			}
			p := filepath.Join(charDir, e.Name())
			if err := v.validateFile("character.json", p, false); err != nil {
				v.add(p, err.Error())
				continue
			}
			if err := validateCharacterHistory(p); err != nil {
				v.add(p, err.Error())
			}
		}
	}

	// 4. foreshadow.json + timeline.json if present.
	for _, name := range []string{"foreshadow.json", "timeline.json"} {
		p := filepath.Join(stateDir, name)
		if exists(p) {
			if err := v.validateFile(name, p, false); err != nil {
				v.add(p, err.Error())
			}
		}
	}

	// 5. Chapter frontmatter validates against chapter-frontmatter.json.
	chapDir := filepath.Join(v.root, "chapters")
	if exists(chapDir) {
		filepath.WalkDir(chapDir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
				return nil
			}
			fm, err := parseFrontmatter(p)
			if err != nil {
				v.add(p, "frontmatter parse error: "+err.Error())
				return nil
			}
			if err := v.validateValue("chapter-frontmatter.json", fm); err != nil {
				v.add(p, err.Error())
			}
			return nil
		})
	}
}

func validateCharacterHistory(path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var character struct {
		History []struct {
			AsOfChapter int `json:"as_of_chapter"`
		} `json:"history"`
	}
	if err := json.Unmarshal(body, &character); err != nil {
		return fmt.Errorf("JSON parse: %w", err)
	}
	previous := 0
	for i, snapshot := range character.History {
		if snapshot.AsOfChapter <= previous {
			return fmt.Errorf("history[%d].as_of_chapter must be strictly increasing (got %d after %d)", i, snapshot.AsOfChapter, previous)
		}
		previous = snapshot.AsOfChapter
	}
	return nil
}

func (v *validator) report() {
	if len(v.violations) == 0 {
		fmt.Printf("novel-spec %s: %s OK\n", specVersion, v.root)
		return
	}
	fmt.Printf("novel-spec %s: %d violation(s) in %s\n", specVersion, len(v.violations), v.root)
	for _, vi := range v.violations {
		fmt.Printf("  ✗ %s: %s\n", vi.path, vi.msg)
	}
}

// validateFile validates a file's content against a schema. If isTOML, the file
// is decoded as TOML then re-encoded to JSON before validation.
func (v *validator) validateFile(schemaName, path string, isTOML bool) error {
	var doc any
	if isTOML {
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var m map[string]any
		if err := toml.Unmarshal(b, &m); err != nil {
			return err
		}
		doc = m
	} else {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		doc, err = jsonschema.UnmarshalJSON(f)
		if err != nil {
			return fmt.Errorf("JSON parse: %w", err)
		}
	}
	return v.validateValue(schemaName, doc)
}

func (v *validator) validateValue(schemaName string, doc any) error {
	schemaPath := filepath.Join(v.schemaDir, schemaName)
	if !exists(schemaPath) {
		return fmt.Errorf("schema file not found: %s", schemaPath)
	}
	c := jsonschema.NewCompiler()
	sch, err := c.Compile(schemaPath)
	if err != nil {
		return fmt.Errorf("schema %s compile: %w", schemaName, err)
	}
	if err := sch.Validate(doc); err != nil {
		return errors.New(formatSchemaErr(err))
	}
	return nil
}

func formatSchemaErr(err error) string {
	var ve *jsonschema.ValidationError
	if errors.As(err, &ve) {
		var b strings.Builder
		b.WriteString("schema validation failed")
		for _, leaf := range leaves(ve) {
			loc := "/" + strings.Join(leaf.InstanceLocation, "/")
			b.WriteString("; at " + loc + ": " + fmt.Sprint(leaf.ErrorKind))
		}
		return b.String()
	}
	return err.Error()
}

func leaves(ve *jsonschema.ValidationError) []*jsonschema.ValidationError {
	if len(ve.Causes) == 0 {
		return []*jsonschema.ValidationError{ve}
	}
	var out []*jsonschema.ValidationError
	for _, c := range ve.Causes {
		out = append(out, leaves(c)...)
	}
	return out
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
