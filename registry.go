package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type agent struct {
	Name                    string
	Commands, Local, Global []string
}

func paths(s string) []string { return strings.Fields(s) }

var agents = []agent{
	{"Claude Code", paths("claude"), paths("CLAUDE.md .claude/CLAUDE.md CLAUDE.local.md .claude/rules/*.md"), paths(".claude/CLAUDE.md .claude/rules/*.md")},
	{"OpenAI Codex", paths("codex"), paths("AGENTS.md AGENTS.override.md"), paths(".codex/AGENTS.md .codex/AGENTS.override.md")},
	{"GitHub Copilot", paths("copilot"), paths(".github/copilot-instructions.md .github/instructions/*.instructions.md AGENTS.md"), paths(".copilot/copilot-instructions.md")},
	{"Cursor", paths("cursor cursor-agent"), paths("AGENTS.md CLAUDE.md .cursor/rules/*.mdc"), nil},
	{"Gemini CLI", paths("gemini"), paths("GEMINI.md"), paths(".gemini/GEMINI.md")},
	{"Google Antigravity", paths("antigravity"), paths("AGENTS.md GEMINI.md .agents/AGENTS.md .agents/GEMINI.md .agents/rules/*.md .agent/rules/*.md"), paths(".gemini/AGENTS.md .gemini/GEMINI.md .gemini/config/AGENTS.md .gemini/config/GEMINI.md .gemini/config/rules/*.md")},
	{"OpenCode", paths("opencode"), paths("AGENTS.md CLAUDE.md"), paths(".config/opencode/AGENTS.md .claude/CLAUDE.md")},
	{"Windsurf / Devin Desktop", paths("windsurf"), paths("AGENTS.md .devin/rules/*.md .windsurf/rules/*.md .windsurfrules"), paths(".codeium/windsurf/memories/global_rules.md")},
	{"Cline", paths("cline"), paths(".clinerules/*.md .cline/rules/*.md"), paths("Documents/Cline/Rules/*.md .cline/rules/*.md Cline/Rules/*.md")},
	{"Roo Code", nil, paths(".roo/rules/*.md .roo/rules-*/*.md .roorules AGENTS.md AGENT.md"), paths(".roo/rules/*.md .roo/rules-*/*.md")},
	{"Kilo Code", paths("kilo"), paths("AGENTS.md CLAUDE.md CONTEXT.md"), paths(".config/kilo/AGENTS.md")},
	{"Kiro", paths("kiro kiro-cli"), paths(".kiro/steering/*.md"), paths(".kiro/steering/*.md")},
	{"JetBrains Junie", paths("junie"), paths(".junie/AGENTS.md AGENTS.md .junie/playbook.md .junie/rules/*.md .junie/guidelines.md .junie/guidelines/*.md"), paths(".junie/AGENTS.md")},
	{"Augment", paths("auggie"), paths("CLAUDE.md AGENTS.md .augment-guidelines .augment/rules/*.md"), paths(".augment/rules/*.md")},
	{"Continue", paths("cn"), paths(".continue/rules/*.md"), nil},
	{"Aider", paths("aider"), nil, nil},
	{"Replit Agent", nil, paths("replit.md"), nil},
	{"Devin cloud", nil, paths("AGENTS.md"), nil},
	{"Warp", paths("warp warp-cli"), paths("AGENTS.md WARP.md"), nil},
	{"goose", paths("goose"), paths(".goosehints"), paths(".config/goose/.goosehints")},
	{"Mistral Vibe", paths("vibe"), paths("AGENTS.md"), paths(".vibe/AGENTS.md")},
	{"Factory Droid", paths("droid"), paths("AGENTS.md .factory/AGENTS.md .agents/AGENTS.md .agent/AGENTS.md CLAUDE.md"), paths(".factory/AGENTS.md .agents/AGENTS.md .agent/AGENTS.md")},
}

func (a agent) locations(global bool, home string) []string {
	if !global {
		return a.Local
	}
	result := append([]string(nil), a.Global...)
	if a.Name == "OpenAI Codex" && os.Getenv("CODEX_HOME") != "" {
		for i, p := range result {
			result[i] = filepath.Join(os.Getenv("CODEX_HOME"), filepath.Base(p))
		}
	}
	return result
}

const agentNotDetected = "not detected on PATH (IDE/cloud installs may exist)"

func (a agent) installed() string {
	for _, cmd := range a.Commands {
		if p, err := exec.LookPath(cmd); err == nil {
			return p
		}
	}
	return agentNotDetected
}

func selectAgents(all bool, names string) ([]agent, error) {
	if all && names != "" {
		return nil, fmt.Errorf("choose --all or --agents")
	}
	if all {
		return agents, nil
	}
	var selected []agent
	if names == "" {
		for _, a := range agents {
			if a.installed() != agentNotDetected {
				selected = append(selected, a)
			}
		}
		return selected, nil
	}
	seen := map[string]bool{}
	for _, name := range strings.Split(names, ",") {
		name = strings.TrimSpace(name)
		found := false
		for _, a := range agents {
			aliases := append([]string{a.Name}, a.Commands...)
			for _, alias := range aliases {
				if !strings.EqualFold(name, alias) {
					continue
				}
				if !seen[a.Name] {
					selected = append(selected, a)
					seen[a.Name] = true
				}
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("unknown agent %q; use names from status -v or registered CLI commands", name)
		}
	}
	return selected, nil
}
