package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSampleNovelConforms(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "sample-novel")
	v := newTestValidator(t, root)
	v.run()
	if v.failed {
		t.Fatalf("sample novel should conform, got violations: %+v", v.violations)
	}
}

func TestEmptyRepositoryFails(t *testing.T) {
	v := newTestValidator(t, t.TempDir())
	v.run()
	assertViolationContains(t, v, "NOVEL.toml", "missing")
}

func TestChapterWithoutFrontmatterFails(t *testing.T) {
	root := copySampleNovel(t)
	path := filepath.Join(root, "chapters", "v01-c001.md")
	if err := os.WriteFile(path, []byte("# The Wreck\n\nNo metadata.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	v := newTestValidator(t, root)
	v.run()
	assertViolationContains(t, v, "v01-c001.md", "no leading ---")
}

func TestInvalidCharacterFails(t *testing.T) {
	root := copySampleNovel(t)
	path := filepath.Join(root, ".novel", "state", "characters", "keeper.json")
	if err := os.WriteFile(path, []byte(`{"id":"keeper","name":"Aren"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	v := newTestValidator(t, root)
	v.run()
	assertViolationContains(t, v, "keeper.json", "schema validation failed")
}

func TestCharacterHistoryMustIncrease(t *testing.T) {
	root := copySampleNovel(t)
	path := filepath.Join(root, ".novel", "state", "characters", "keeper.json")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body = []byte(strings.Replace(string(body), `"as_of_chapter": 2`, `"as_of_chapter": 1`, 1))
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}

	v := newTestValidator(t, root)
	v.run()
	assertViolationContains(t, v, "keeper.json", "strictly increasing")
}

func newTestValidator(t *testing.T, root string) *validator {
	t.Helper()
	schemaDir, err := filepath.Abs(filepath.Join("..", "..", "schema"))
	if err != nil {
		t.Fatal(err)
	}
	return &validator{root: root, schemaDir: schemaDir}
}

func assertViolationContains(t *testing.T, v *validator, pathPart, messagePart string) {
	t.Helper()
	for _, got := range v.violations {
		if strings.Contains(got.path, pathPart) && strings.Contains(got.msg, messagePart) {
			return
		}
	}
	t.Fatalf("expected violation path containing %q and message containing %q; got %+v", pathPart, messagePart, v.violations)
}

func copySampleNovel(t *testing.T) string {
	t.Helper()
	src := filepath.Join("..", "..", "examples", "sample-novel")
	dst := filepath.Join(t.TempDir(), "sample-novel")
	if err := copyTree(src, dst); err != nil {
		t.Fatal(err)
	}
	return dst
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, body, 0o644)
	})
}
