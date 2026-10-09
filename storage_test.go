package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func fixture(t *testing.T) *app {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(root, "home")
	project := filepath.Join(home, "project")
	if err = os.MkdirAll(project, 0700); err != nil {
		t.Fatal(err)
	}
	return &app{home: home, root: project, data: filepath.Join(home, ".ai.cross"), history: filepath.Join(home, ".ai.cross", "instructions", "project_location", "project")}
}
func TestBackupRestoreAndAppliedHistory(t *testing.T) {
	a := fixture(t)
	old := filepath.Join(a.root, "CLAUDE.md")
	created := filepath.Join(a.root, ".team", "VIBE.md")
	if err := os.WriteFile(old, []byte("original"), 0640); err != nil {
		t.Fatal(err)
	}
	next := []snapshot{{Path: old, Data: []byte("new"), Mode: 0640, Exists: true}, {Path: created, Data: []byte("new"), Mode: 0600, Exists: true}}
	name, err := a.apply(next, []byte("new"), true)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(a.history, name, "CLAUDE.md"))
	if err != nil || string(b) != "original" {
		t.Fatalf("backup: %q, %v", b, err)
	}
	if err = a.restore(name); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(old)
	if err != nil || string(b) != "original" {
		t.Fatalf("restore: %q, %v", b, err)
	}
	if _, err = os.Stat(created); !os.IsNotExist(err) {
		t.Fatalf("new file remains: %v", err)
	}
	if err = a.restore(name + ".md"); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(created)
	if err != nil || string(b) != "new" {
		t.Fatalf("applied restore: %q, %v", b, err)
	}
}
func TestNoBackupStillRecordsApply(t *testing.T) {
	a := fixture(t)
	name, err := a.apply([]snapshot{{Path: filepath.Join(a.root, "AGENTS.md"), Data: []byte("rules"), Mode: 0600, Exists: true}}, []byte("rules"), false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(filepath.Join(a.history, name)); !os.IsNotExist(err) {
		t.Fatalf("unexpected backup: %v", err)
	}
	for _, suffix := range []string{".md", ".json"} {
		if _, err = os.Stat(filepath.Join(a.history, name+suffix)); err != nil {
			t.Fatal(err)
		}
	}
}
func TestTransactionRollsBackOnWriteFailure(t *testing.T) {
	a := fixture(t)
	first := filepath.Join(a.root, "first.md")
	blocker := filepath.Join(a.root, "blocker")
	os.WriteFile(first, []byte("before"), 0600)
	next := []snapshot{{Path: first, Data: []byte("after"), Mode: 0600, Exists: true}, {Path: blocker, Data: []byte("file"), Mode: 0600, Exists: true}, {Path: filepath.Join(blocker, "child.md"), Data: []byte("rules"), Mode: 0600, Exists: true}}
	if err := transaction(next); err == nil {
		t.Fatal("expected failure")
	}
	b, err := os.ReadFile(first)
	if err != nil || string(b) != "before" {
		t.Fatalf("rollback: %q, %v", b, err)
	}
	if _, err = os.Stat(blocker); !os.IsNotExist(err) {
		t.Fatalf("created file not removed: %v", err)
	}
}
func TestPathsAndIgnores(t *testing.T) {
	a := fixture(t)
	for _, p := range []string{"../escape.md", filepath.Join(a.data, "config.yml")} {
		if _, err := a.resolve(p); err == nil {
			t.Fatalf("accepted %s", p)
		}
	}
	a.config = scopeConfig{Additional: []string{".team/VIBE.md"}, Ignore: []string{"CLAUDE.md", ".cursor/rules/*.mdc"}}
	targets, err := a.targets(agents)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range targets {
		rel, _ := filepath.Rel(a.root, p)
		if rel == "CLAUDE.md" || rel == filepath.Join(".cursor", "rules", "ai-cross.mdc") {
			t.Fatalf("ignored path: %s", rel)
		}
		if rel == filepath.Join(".team", "VIBE.md") {
			found = true
		}
	}
	if !found {
		t.Fatal("additional path missing")
	}
}
func TestConcurrentWriteRejected(t *testing.T) {
	a := fixture(t)
	os.MkdirAll(a.data, 0700)
	os.WriteFile(filepath.Join(a.data, "write.lock"), nil, 0600)
	if _, err := a.apply(nil, nil, true); err == nil {
		t.Fatal("accepted locked write")
	}
}

func TestSelectedAgentTargets(t *testing.T) {
	a := fixture(t)
	a.config = scopeConfig{Additional: []string{".team/VIBE.md"}, Ignore: []string{"AGENTS.override.md"}}
	selected, err := selectAgents(false, "codex")
	if err != nil {
		t.Fatal(err)
	}
	targets, err := a.targets(selected)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(a.root, ".team", "VIBE.md"), filepath.Join(a.root, "AGENTS.md")}
	if !reflect.DeepEqual(targets, want) {
		t.Fatalf("targets %v, want %v", targets, want)
	}
	targets, err = a.targets(nil)
	if err != nil || !reflect.DeepEqual(targets, want[:1]) {
		t.Fatalf("additional targets without agents: %v, %v", targets, err)
	}
	a.config = scopeConfig{}
	a.global = true
	a.root = a.home
	t.Setenv("CODEX_HOME", "")
	targets, err = a.targets(selected)
	want = []string{filepath.Join(a.home, ".codex", "AGENTS.md"), filepath.Join(a.home, ".codex", "AGENTS.override.md")}
	if err != nil || !reflect.DeepEqual(targets, want) {
		t.Fatalf("global targets %v, want %v: %v", targets, want, err)
	}
}

func TestBlackHoleAndRestore(t *testing.T) {
	for _, global := range []bool{false, true} {
		name := "local"
		if global {
			name = "global"
		}
		t.Run(name, func(t *testing.T) {
			a := fixture(t)
			other := filepath.Join(a.home, ".claude", "CLAUDE.md")
			files := []string{"AGENTS.md", "AGENTS.override.md", ".roo/rules-code/custom.md", ".team/rules.md"}
			if global {
				other = filepath.Join(a.root, "AGENTS.md")
				a.global = true
				a.root = a.home
				a.history = filepath.Join(a.data, "instructions", "global")
				files = []string{".codex/AGENTS.md", ".codex/AGENTS.override.md", ".roo/rules-code/custom.md", ".team/rules.md"}
			}
			t.Setenv("CODEX_HOME", "")
			t.Setenv("PATH", t.TempDir())
			a.config.Additional = []string{".team/*.md"}
			for _, p := range append(files, "notes.md") {
				if err := writeFile(filepath.Join(a.root, p), []byte("original"), 0640); err != nil {
					t.Fatal(err)
				}
			}
			if err := writeFile(other, []byte("other scope"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := a.blackHole(); err != nil {
				t.Fatal(err)
			}
			for _, p := range files {
				if _, err := os.Stat(filepath.Join(a.root, p)); !os.IsNotExist(err) {
					t.Fatalf("instruction remains: %s: %v", p, err)
				}
			}
			for _, p := range []string{other, filepath.Join(a.root, "notes.md")} {
				if _, err := os.Stat(p); err != nil {
					t.Fatalf("unrelated file removed: %s: %v", p, err)
				}
			}
			entries, err := os.ReadDir(a.history)
			if err != nil {
				t.Fatal(err)
			}
			var backup string
			for _, e := range entries {
				if e.IsDir() {
					backup = e.Name()
				}
			}
			if err := a.restore(backup); err != nil {
				t.Fatal(err)
			}
			for _, p := range files {
				b, err := os.ReadFile(filepath.Join(a.root, p))
				if err != nil || string(b) != "original" {
					t.Fatalf("restore %s: %q, %v", p, b, err)
				}
			}
		})
	}
}
