package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	"golang.org/x/term"
)

var version = "dev"

const help = `ai-cross — synchronize AI agent instruction files

Commands:
  status [--global | --local] [--project DIR] [-v | --verbose]
  version
  apply (--global | --local) [--all | --agents NAMES] [--project DIR] [--file FILE | --url URL [--save-url] | --saved-url | --clipboard | --input] [--no-backup]
  black-hole (--global | --local) [--project DIR]
  histories [--global | --local] [--project DIR] [--applies | --backups]
  restore list [--global | --local] [--project DIR] [--applies | --backups]
  restore (--global | --local) --name TIMESTAMP[.md] [--project DIR]
  config
  help

Status and histories default to local scope. Local scope is --project or cwd.
Status defaults to a summary; -v or --verbose includes all instruction file locations.
Apply defaults to agents detected on PATH. --all selects every agent; --agents accepts comma-separated names or CLI commands.
Black-hole removes all registered instruction files and config additional paths, including ignored paths, with a backup.
Input defaults to a multiline editor; Ctrl+S applies, Esc cancels. Piped stdin is supported.
Use --url URL --save-url to remember a URL for the selected scope; --saved-url downloads it again.
Config: ~/.ai.cross/config.yml (or config.yaml).
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) == 0 {
		fmt.Print(help)
		return nil
	}
	command := args[0]
	args = args[1:]
	if command == "help" || command == "--help" || command == "-h" {
		fmt.Print(help)
		return nil
	}
	if command == "version" || command == "--version" {
		fmt.Println(version)
		return nil
	}
	if command == "restore" && len(args) > 0 && args[0] == "list" {
		command = "histories"
		args = args[1:]
	}
	switch command {
	case "status", "apply", "black-hole", "histories", "restore", "config":
	default:
		return fmt.Errorf("unknown command %q; use help", command)
	}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	global := fs.Bool("global", false, "use home-dir instructions")
	local := fs.Bool("local", false, "use project instructions")
	project := fs.String("project", ".", "project directory")
	var file, url, name, agentNames string
	var clipboard, input, noBackup, applies, backups bool
	var saveURL, savedURL bool
	var verbose bool
	var all bool
	if command == "status" {
		fs.BoolVar(&verbose, "v", false, "show full status with instruction file locations")
		fs.BoolVar(&verbose, "verbose", false, "show full status with instruction file locations")
	}
	if command == "apply" {
		fs.BoolVar(&all, "all", false, "apply to all registered agents regardless of detection")
		fs.StringVar(&agentNames, "agents", "", "comma-separated agent names or CLI commands, regardless of detection")
		fs.StringVar(&file, "file", "", "instruction source file")
		fs.StringVar(&url, "url", "", "instruction source HTTP(S) URL (raw text or Markdown)")
		fs.BoolVar(&saveURL, "save-url", false, "save --url in config after a successful apply for this scope")
		fs.BoolVar(&savedURL, "saved-url", false, "download and apply the saved URL for this scope")
		fs.BoolVar(&clipboard, "clipboard", false, "read clipboard")
		fs.BoolVar(&input, "input", false, "read interactive input or stdin")
		fs.BoolVar(&noBackup, "no-backup", false, "skip pre-apply backup")
	}
	if command == "restore" {
		fs.StringVar(&name, "name", "", "history record name")
	}
	if command == "histories" {
		fs.BoolVar(&applies, "applies", false, "only applied instructions")
		fs.BoolVar(&backups, "backups", false, "only pre-apply backups")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	if *global && *local {
		return errors.New("choose one scope: --global or --local")
	}
	if (command == "apply" || command == "restore" || command == "black-hole") && !*global && !*local {
		return errors.New("explicit --global or --local is required")
	}
	a, err := newApp(*global, *project)
	if err != nil {
		return err
	}
	switch command {
	case "black-hole":
		return a.blackHole()
	case "config":
		fmt.Printf("Config: %s (or config.yaml)\n", filepath.Join(a.data, "config.yml"))
		return nil
	case "status":
		if !verbose {
			var detected []string
			for _, agent := range agents {
				if agent.installed() != agentNotDetected {
					detected = append(detected, agent.Name)
				}
			}
			scope := "local"
			if a.global {
				scope = "global"
			}
			fmt.Printf("%s | %d/%d agents detected on PATH | -v for details\n", scope, len(detected), len(agents))
			if len(detected) > 0 {
				fmt.Println(strings.Join(detected, ", "))
			}
			return nil
		}
		fmt.Printf("ai-cross %s\nScope: %s\n", version, a.root)
		for _, agent := range agents {
			fmt.Printf("\n%s: %s\n", agent.Name, agent.installed())
			for _, scope := range []bool{false, true} {
				label, base := "local", a.root
				if a.global {
					base, err = filepath.Abs(*project)
					if err != nil {
						return err
					}
				}
				if scope {
					label, base = "global", a.home
				}
				locations := agent.locations(scope, a.home)
				if len(locations) == 0 {
					fmt.Printf("  %s: no fixed file; use config additional paths\n", label)
				}
				for _, p := range locations {
					if !filepath.IsAbs(p) {
						p = filepath.Join(base, p)
					}
					fmt.Printf("  %s: %s\n", label, p)
				}
			}
		}
		return nil
	case "histories":
		if applies && backups {
			return errors.New("choose --applies or --backups")
		}
		return a.list(applies, backups)
	case "apply":
		count := 0
		for _, v := range []bool{file != "", url != "", savedURL, clipboard, input} {
			if v {
				count++
			}
		}
		if count > 1 {
			return errors.New("choose one source: --file, --url, --saved-url, --clipboard, or --input")
		}
		if saveURL && url == "" {
			return errors.New("--save-url requires --url")
		}
		if savedURL {
			url = a.config.URL
			if url == "" {
				scope := "local"
				if a.global {
					scope = "global"
				}
				return fmt.Errorf("no saved URL for %s scope; use --url URL --save-url first", scope)
			}
		}
		selected, err := selectAgents(all, agentNames)
		if err != nil {
			return err
		}
		targets, err := a.targets(selected)
		if err != nil {
			return err
		}
		if len(targets) == 0 {
			return errors.New("no instruction targets; use status -v to check detection, --all, or --agents")
		}
		var content []byte
		switch {
		case file != "":
			content, err = os.ReadFile(file)
		case url != "":
			content, err = readURL(url)
		case clipboard:
			content, err = readClipboard()
		default:
			content, err = readInput()
		}
		if err != nil {
			return err
		}
		if len(strings.TrimSpace(string(content))) == 0 {
			return errors.New("instructions cannot be empty")
		}
		next := make([]snapshot, 0, len(targets))
		for _, p := range targets {
			s, e := capture(p)
			if e != nil {
				return e
			}
			s.Data = content
			s.Exists = true
			next = append(next, s)
		}
		urlToSave := ""
		if saveURL {
			urlToSave = url
		}
		name, err := a.applyWithURL(next, content, !noBackup, urlToSave)
		if err != nil {
			return err
		}
		fmt.Printf("Applied to %d files. History: %s\n", len(targets), filepath.Join(a.history, name+".md"))
		return nil
	case "restore":
		return a.restore(name)
	}
	return nil
}
func historyName(name string) bool {
	_, err := time.Parse("20060102150405", strings.TrimSuffix(name, ".md"))
	return err == nil && filepath.Base(name) == name
}
func (a *app) blackHole() error {
	selection := *a
	selection.config.Ignore = nil
	targets, err := selection.targets(agents)
	if err != nil {
		return err
	}
	var next []snapshot
	for _, p := range targets {
		s, err := capture(p)
		if err != nil {
			return err
		}
		if s.Exists {
			next = append(next, snapshot{Path: p})
		}
	}
	if len(next) == 0 {
		fmt.Println("No instruction files to remove.")
		return nil
	}
	name, err := a.apply(next, nil, true)
	if err != nil {
		return err
	}
	fmt.Printf("Removed %d instruction files. Backup: %s\n", len(next), filepath.Join(a.history, name))
	return nil
}
func (a *app) list(applies, backups bool) error {
	entries, err := os.ReadDir(a.history)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Println("No history.")
		return nil
	}
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, e := range entries {
		if !historyName(e.Name()) || applies && e.IsDir() || backups && !e.IsDir() {
			continue
		}
		t, _ := time.ParseInLocation("20060102150405", strings.TrimSuffix(e.Name(), ".md"), time.Local)
		kind := "apply"
		if e.IsDir() {
			kind = "backup"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Name(), kind, t.Format("2006-01-02 15:04:05 MST"), filepath.Join(a.history, e.Name()))
	}
	return w.Flush()
}
func (a *app) restore(name string) error {
	if !historyName(name) {
		return errors.New("--name must be a timestamp or timestamp.md")
	}
	p := filepath.Join(a.history, name)
	if err := safePath(p); err != nil {
		return err
	}
	var next []snapshot
	var content []byte
	if strings.HasSuffix(name, ".md") {
		var err error
		content, err = os.ReadFile(p)
		if err != nil {
			return err
		}
		metadata := strings.TrimSuffix(p, ".md") + ".json"
		if err := safePath(metadata); err != nil {
			return err
		}
		b, err := os.ReadFile(metadata)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(b, &next); err != nil {
			return err
		}
		seen := map[string]bool{}
		for i := range next {
			relative := next[i].Path
			if filepath.IsAbs(relative) || relative == "." {
				return errors.New("invalid history path")
			}
			next[i].Path, err = a.resolve(relative)
			if err != nil {
				return err
			}
			if seen[next[i].Path] {
				return errors.New("duplicate history path")
			}
			seen[next[i].Path] = true
		}
	} else {
		metadata := filepath.Join(p, ".ai-cross-manifest.json")
		if err := safePath(metadata); err != nil {
			return err
		}
		b, err := os.ReadFile(metadata)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(b, &next); err != nil {
			return err
		}
		seen := map[string]bool{}
		for i := range next {
			s := &next[i]
			relative := s.Path
			if filepath.IsAbs(relative) || relative == "." {
				return errors.New("invalid backup path")
			}
			s.Path, err = a.resolve(relative)
			if err != nil {
				return err
			}
			if seen[s.Path] {
				return errors.New("duplicate backup path")
			}
			seen[s.Path] = true
			if s.Exists {
				source := filepath.Join(p, relative)
				if err = safePath(source); err != nil {
					return err
				}
				s.Data, err = os.ReadFile(source)
				if err != nil {
					return err
				}
			}
			content = append(content, []byte(fmt.Sprintf("\n## %s\n\n", relative))...)
			content = append(content, s.Data...)
		}
	}
	if len(next) == 0 {
		return errors.New("history has no files to restore")
	}
	record, err := a.apply(next, content, true)
	if err != nil {
		return err
	}
	fmt.Printf("Restored %s. History: %s\n", name, filepath.Join(a.history, record+".md"))
	return nil
}
func readURL(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid instruction URL: %w", err)
	}
	if (req.URL.Scheme != "http" && req.URL.Scheme != "https") || req.URL.Host == "" {
		return nil, errors.New("instruction URL must use http:// or https:// with a host")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch instructions: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("fetch instructions: HTTP %s", resp.Status)
	}
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read instructions: %w", err)
	}
	return content, nil
}
func readClipboard() ([]byte, error) {
	var commands [][]string
	switch runtime.GOOS {
	case "darwin":
		commands = [][]string{{"pbpaste"}}
	case "windows":
		commands = [][]string{{"powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new(); Get-Clipboard -Raw"}}
	default:
		commands = [][]string{{"wl-paste", "--no-newline"}, {"xclip", "-selection", "clipboard", "-o"}, {"xsel", "--clipboard", "--output"}}
	}
	var failures error
	for _, args := range commands {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		b, err := exec.CommandContext(ctx, args[0], args[1:]...).Output()
		cancel()
		if err == nil {
			return b, nil
		}
		failures = errors.Join(failures, err)
	}
	return nil, fmt.Errorf("clipboard unavailable: %w", failures)
}
func readInput() ([]byte, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return io.ReadAll(os.Stdin)
	}
	return editInput()
}
