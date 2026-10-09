package main

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestSelectAgents(t *testing.T) {
	bin := t.TempDir()
	t.Setenv("PATH", bin)
	command := "codex"
	if runtime.GOOS == "windows" {
		command += ".exe"
		t.Setenv("PATHEXT", ".EXE")
	}
	if err := os.WriteFile(filepath.Join(bin, command), []byte("test executable"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name  string
		all   bool
		names string
		want  []string
	}{
		{"detected", false, "", []string{"OpenAI Codex"}},
		{"explicit undetected", false, "claude,Roo Code", []string{"Claude Code", "Roo Code"}},
		{"aliases and duplicates", false, " CODEX ,OpenAI Codex,cursor-agent", []string{"OpenAI Codex", "Cursor"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			selected, err := selectAgents(tt.all, tt.names)
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, a := range selected {
				names = append(names, a.Name)
			}
			if !reflect.DeepEqual(names, tt.want) {
				t.Fatalf("selected %v, want %v", names, tt.want)
			}
		})
	}
	selected, err := selectAgents(true, "")
	if err != nil || !reflect.DeepEqual(selected, agents) {
		t.Fatalf("all agents: %v, %v", selected, err)
	}
	for _, names := range []string{"unknown", "codex,unknown", "codex,", " "} {
		if _, err := selectAgents(false, names); err == nil {
			t.Fatalf("accepted invalid selection %q", names)
		}
	}
	if _, err := selectAgents(true, "codex"); err == nil {
		t.Fatal("accepted conflicting flags")
	}
	t.Setenv("PATH", t.TempDir())
	selected, err = selectAgents(false, "")
	if err != nil || len(selected) != 0 {
		t.Fatalf("no detected agents: %v, %v", selected, err)
	}
}
