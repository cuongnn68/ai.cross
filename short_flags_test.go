package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShortFlagWorkflow(t *testing.T) {
	a := fixture(t)
	t.Setenv("HOME", a.home)
	t.Setenv("USERPROFILE", a.home)
	t.Setenv("CODEX_HOME", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("URL rules"))
	}))
	defer server.Close()
	source := filepath.Join(a.home, "source.md")
	if err := os.WriteFile(source, []byte("file rules"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"apply", "-l", "-p", a.root, "-a", "codex", "-u", server.URL, "-s", "-n"},
		{"apply", "--local", "-p", a.root, "--agents", "codex", "-r"},
		{"apply", "-g", "-p", a.root, "-A", "-f", source},
		{"status", "-g", "-p", a.root, "-v"},
		{"histories", "-l", "-p", a.root, "-a"},
		{"restore", "list", "-l", "-p", a.root, "-b"},
		{"-V"},
		{"-h"},
		{"apply", "-h"},
	} {
		if err := run(args); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	for path, want := range map[string]string{
		filepath.Join(a.root, "AGENTS.md"):           "URL rules",
		filepath.Join(a.home, ".codex", "AGENTS.md"): "file rules",
	} {
		b, err := os.ReadFile(path)
		if err != nil || string(b) != want {
			t.Fatalf("%s: %q, %v", path, b, err)
		}
	}
	local, err := newApp(false, a.root)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(local.history)
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".md") {
			record = entry.Name()
			break
		}
	}
	if record == "" {
		t.Fatal("missing applied history")
	}
	if err := run([]string{"black-hole", "-l", "-p", a.root}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"restore", "-l", "-p", a.root, "-n", record}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(a.root, "AGENTS.md"))
	if err != nil || string(b) != "URL rules" {
		t.Fatalf("short restore: %q, %v", b, err)
	}
}

func TestShortFlagConflicts(t *testing.T) {
	a := fixture(t)
	t.Setenv("HOME", a.home)
	t.Setenv("USERPROFILE", a.home)
	for _, tt := range []struct {
		args []string
		want string
	}{
		{[]string{"apply", "-g", "--local"}, "choose one scope"},
		{[]string{"apply", "-l", "-A", "--agents", "codex"}, "choose --all or --agents"},
		{[]string{"apply", "-l", "-c", "--file", "rules.md"}, "choose one source"},
		{[]string{"apply", "-l", "-i", "--saved-url"}, "choose one source"},
		{[]string{"apply", "-l", "-s"}, "--save-url requires --url"},
		{[]string{"histories", "-a", "--backups"}, "choose --applies or --backups"},
	} {
		args := append(append([]string{}, tt.args...), "-p", a.root)
		if err := run(args); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("%v: %v, want %q", args, err, tt.want)
		}
	}
}
