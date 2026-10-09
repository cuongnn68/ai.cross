package main

import (
	"flag"
	"io"
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
		{"apply", "-lsnu", server.URL, "-p", a.root, "-acodex"},
		{"apply", "--local", "-p", a.root, "--agents", "codex", "-r"},
		{"apply", "-gAf", source, "-p", a.root},
		{"status", "-gv", "-p", a.root},
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
		{[]string{"apply", "-gl"}, "choose one scope"},
		{[]string{"apply", "-lAa", "codex"}, "choose --all or --agents"},
	} {
		args := append(append([]string{}, tt.args...), "-p", a.root)
		if err := run(args); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Fatalf("%v: %v, want %q", args, err, tt.want)
		}
	}
}

func TestCombinedFlagParsing(t *testing.T) {
	for _, tt := range []struct {
		name                 string
		args                 []string
		wantGlobal, wantSave bool
		wantURL              string
		wantArgs             string
		wantErr              string
	}{
		{name: "group with separate value", args: []string{"-gsu", "https://example.com/rules"}, wantGlobal: true, wantSave: true, wantURL: "https://example.com/rules"},
		{name: "attached value", args: []string{"-gsuhttps://example.com/rules"}, wantGlobal: true, wantSave: true, wantURL: "https://example.com/rules"},
		{name: "equals value", args: []string{"-gsu=https://example.com/rules?a=b"}, wantGlobal: true, wantSave: true, wantURL: "https://example.com/rules?a=b"},
		{name: "explicit boolean", args: []string{"-gs=false"}, wantGlobal: true},
		{name: "long flags", args: []string{"--global", "--save-url", "--url", "-gsu"}, wantGlobal: true, wantSave: true, wantURL: "-gsu"},
		{name: "single dash long", args: []string{"-global", "-url=-gsu"}, wantGlobal: true, wantURL: "-gsu"},
		{name: "value resembles group", args: []string{"-gu", "-gsu"}, wantGlobal: true, wantURL: "-gsu"},
		{name: "terminator", args: []string{"-g", "--", "-gsu"}, wantGlobal: true, wantArgs: "-gsu"},
		{name: "positional", args: []string{"value", "-gsu"}, wantArgs: "value -gsu"},
		{name: "unknown short", args: []string{"-gz"}, wantErr: "flag provided but not defined: -z"},
		{name: "unknown long", args: []string{"--gsu"}, wantErr: "flag provided but not defined: -gsu"},
		{name: "missing value", args: []string{"-gsu"}, wantErr: "flag needs an argument: -u"},
		{name: "combined help", args: []string{"-hg"}, wantErr: flag.ErrHelp.Error()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.SetOutput(io.Discard)
			var global, save bool
			var url string
			for _, name := range []string{"g", "global"} {
				fs.BoolVar(&global, name, false, "")
			}
			for _, name := range []string{"s", "save-url"} {
				fs.BoolVar(&save, name, false, "")
			}
			for _, name := range []string{"u", "url"} {
				fs.StringVar(&url, name, "", "")
			}
			err := fs.Parse(expandShortFlags(fs, tt.args))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error: %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if global != tt.wantGlobal || save != tt.wantSave || url != tt.wantURL || strings.Join(fs.Args(), " ") != tt.wantArgs {
				t.Fatalf("parsed global=%v save=%v url=%q args=%v", global, save, url, fs.Args())
			}
		})
	}
}
