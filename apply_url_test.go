package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSavedURLScopesAndRefresh(t *testing.T) {
	a := fixture(t)
	t.Setenv("HOME", a.home)
	t.Setenv("USERPROFILE", a.home)
	t.Setenv("CODEX_HOME", "")
	content := "first"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(content + r.URL.Path))
	}))
	defer server.Close()
	for _, scope := range []string{"--local", "--global"} {
		args := []string{"apply", scope, "--project", a.root, "--agents", "codex", "--url", server.URL + "/" + scope, "--save-url"}
		if err := run(args); err != nil {
			t.Fatal(err)
		}
	}
	b, err := os.ReadFile(filepath.Join(a.data, "config.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Histories.Local.URL != server.URL+"/--local" || cfg.Histories.Global.URL != server.URL+"/--global" {
		t.Fatalf("scope URLs not preserved: %s", b)
	}
	// A one-off URL apply must leave the saved source intact.
	if err := run([]string{"apply", "--local", "--project", a.root, "--agents", "codex", "--url", server.URL + "/one-off"}); err != nil {
		t.Fatal(err)
	}
	content = "updated"
	for _, scope := range []string{"--local", "--global"} {
		if err := run([]string{"apply", scope, "--project", a.root, "--agents", "codex", "--saved-url"}); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(a.root, "AGENTS.md")
		if scope == "--global" {
			p = filepath.Join(a.home, ".codex", "AGENTS.md")
		}
		b, err := os.ReadFile(p)
		if err != nil || string(b) != "updated/"+scope {
			t.Fatalf("refresh %s: %q, %v", scope, b, err)
		}
	}
}

func TestSaveURLPreservesConfig(t *testing.T) {
	for _, filename := range []string{"config.yml", "config.yaml"} {
		t.Run(filename, func(t *testing.T) {
			a := fixture(t)
			original := "# My settings\nhistories:\n  global:\n    url: https://example.com/global\n    ignore: [CLAUDE.md]\n  local:\n    additional: [.team/rules.md] # Keep this\n"
			p := filepath.Join(a.data, filename)
			if err := writeFile(p, []byte(original), 0640); err != nil {
				t.Fatal(err)
			}
			if _, err := a.applyWithURL(nil, []byte("rules"), false, "https://example.com/local"); err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			var cfg config
			if err := yaml.Unmarshal(b, &cfg); err != nil {
				t.Fatal(err)
			}
			if cfg.Histories.Global.URL != "https://example.com/global" || cfg.Histories.Local.URL != "https://example.com/local" || len(cfg.Histories.Global.Ignore) != 1 || len(cfg.Histories.Local.Additional) != 1 {
				t.Fatalf("settings lost: %s", b)
			}
			if !strings.Contains(string(b), "# My settings") || !strings.Contains(string(b), "# Keep this") {
				t.Fatalf("comments lost: %s", b)
			}
			info, err := os.Stat(p)
			if err != nil || info.Mode().Perm() != 0640 {
				t.Fatalf("config permissions: %v, %v", info, err)
			}
			if filename == "config.yaml" {
				if _, err := os.Stat(filepath.Join(a.data, "config.yml")); !os.IsNotExist(err) {
					t.Fatalf("created higher-priority config: %v", err)
				}
			}
		})
	}
}

func TestSavedURLValidationAndDownloadFailure(t *testing.T) {
	a := fixture(t)
	t.Setenv("HOME", a.home)
	t.Setenv("USERPROFILE", a.home)
	base := []string{"apply", "--local", "--project", a.root, "--agents", "codex"}
	for _, flags := range [][]string{
		{"--save-url"},
		{"--saved-url"},
		{"--saved-url", "--url", "https://example.com"},
		{"--saved-url", "--file", "rules.md"},
		{"--saved-url", "--clipboard"},
		{"--saved-url", "--input"},
		{"--saved-url", "--save-url"},
	} {
		if err := run(append(append([]string{}, base...), flags...)); err == nil {
			t.Fatalf("accepted invalid options: %v", flags)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	p := filepath.Join(a.data, "config.yml")
	original := []byte("histories:\n  local:\n    url: https://example.com/previous\n")
	if err := writeFile(p, original, 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/fail", "/empty"} {
		if err := run(append(append([]string{}, base...), "--url", server.URL+path, "--save-url")); err == nil {
			t.Fatalf("accepted failed or empty download: %s", path)
		}
		b, err := os.ReadFile(p)
		if err != nil || string(b) != string(original) {
			t.Fatalf("failed download changed config: %q, %v", b, err)
		}
	}
	if _, err := os.Stat(filepath.Join(a.root, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("failed input wrote target: %v", err)
	}
}

func TestSaveURLRollsBackOnApplyFailure(t *testing.T) {
	a := fixture(t)
	p := filepath.Join(a.data, "config.yml")
	original := []byte("histories:\n  local:\n    url: https://example.com/previous\n")
	if err := writeFile(p, original, 0600); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(a.root, "first.md")
	if err := writeFile(first, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	blocker := filepath.Join(a.root, "blocker")
	next := []snapshot{
		{Path: first, Data: []byte("new"), Mode: 0600, Exists: true},
		{Path: blocker, Data: []byte("file"), Mode: 0600, Exists: true},
		{Path: filepath.Join(blocker, "child.md"), Data: []byte("new"), Mode: 0600, Exists: true},
	}
	if _, err := a.applyWithURL(next, []byte("new"), true, "https://example.com/new"); err == nil {
		t.Fatal("expected write failure")
	}
	for path, want := range map[string]string{p: string(original), first: "original"} {
		b, err := os.ReadFile(path)
		if err != nil || string(b) != want {
			t.Fatalf("rollback %s: %q, %v", path, b, err)
		}
	}
}
