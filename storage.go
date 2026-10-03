package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type scopeConfig struct {
	Additional []string `yaml:"additional"`
	Ignore     []string `yaml:"ignore"`
}
type config struct {
	Histories struct {
		Global scopeConfig `yaml:"global"`
		Local  scopeConfig `yaml:"local"`
	} `yaml:"histories"`
}
type app struct {
	home, root, data, history string
	global                    bool
	config                    scopeConfig
}
type snapshot struct {
	Path   string
	Data   []byte
	Mode   os.FileMode
	Exists bool
}

func newApp(global bool, project string) (*app, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	root, err := filepath.Abs(project)
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	a := &app{home: home, root: root, global: global, data: filepath.Join(home, ".ai.cross")}
	if global {
		a.root = home
		a.history = filepath.Join(a.data, "instructions", "global")
	} else {
		relative := strings.TrimLeft(strings.TrimPrefix(root, filepath.VolumeName(root)), string(filepath.Separator))
		volume := strings.TrimSuffix(filepath.VolumeName(root), ":")
		volume = strings.NewReplacer("\\", "_", "/", "_").Replace(volume)
		a.history = filepath.Join(a.data, "instructions", "project_location", volume, relative)
	}
	var cfg config
	for _, name := range []string{"config.yml", "config.yaml"} {
		b, e := os.ReadFile(filepath.Join(a.data, name))
		if errors.Is(e, os.ErrNotExist) {
			continue
		}
		if e != nil {
			return nil, e
		}
		decoder := yaml.NewDecoder(bytes.NewReader(b))
		decoder.KnownFields(true)
		if err := decoder.Decode(&cfg); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		break
	}
	a.config = cfg.Histories.Local
	if global {
		a.config = cfg.Histories.Global
	}
	return a, nil
}
func (a *app) resolve(p string) (string, error) {
	if p == "~" {
		p = a.home
	} else if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		p = filepath.Join(a.home, p[2:])
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(a.root, p)
	}
	p = filepath.Clean(p)
	rel, err := filepath.Rel(a.root, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path outside scope: %s", p)
	}
	dataRel, _ := filepath.Rel(a.data, p)
	if dataRel == "." || (dataRel != ".." && !strings.HasPrefix(dataRel, ".."+string(filepath.Separator))) {
		return "", fmt.Errorf("instruction path overlaps tool data: %s", p)
	}
	return p, nil
}
func safePath(p string) error {
	for q := p; ; q = filepath.Dir(q) {
		info, err := os.Lstat(q)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink not supported: %s", q)
		}
		parent := filepath.Dir(q)
		if parent == q {
			break
		}
	}
	return nil
}
func (a *app) targets() ([]string, error) {
	var patterns []string
	for _, agent := range agents {
		patterns = append(patterns, agent.locations(a.global, a.home)...)
	}
	patterns = append(patterns, a.config.Additional...)
	ignored := []string{}
	for _, p := range a.config.Ignore {
		p, e := a.resolve(p)
		if e != nil {
			return nil, e
		}
		ignored = append(ignored, p)
	}
	found := map[string]bool{}
	for _, pattern := range patterns {
		p, e := a.resolve(pattern)
		if e != nil {
			return nil, e
		}
		matches := []string{p}
		if strings.ContainsAny(p, "*?[") {
			matches, e = filepath.Glob(p)
			if e != nil {
				return nil, e
			}
			if len(matches) == 0 && !strings.ContainsAny(filepath.Dir(p), "*?[") {
				suffix := ".md"
				if strings.HasSuffix(p, ".mdc") {
					suffix = ".mdc"
				}
				if strings.HasSuffix(p, ".instructions.md") {
					suffix = ".instructions.md"
				}
				matches = []string{filepath.Join(filepath.Dir(p), "ai-cross"+suffix)}
			}
		}
		for _, match := range matches {
			skip := false
			for _, pattern := range ignored {
				yes, e := filepath.Match(pattern, match)
				if e != nil {
					return nil, e
				}
				if yes {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
			if e := safePath(match); e != nil {
				return nil, e
			}
			found[match] = true
		}
	}
	result := []string{}
	for p := range found {
		result = append(result, p)
	}
	sort.Strings(result)
	return result, nil
}
func capture(p string) (snapshot, error) {
	s := snapshot{Path: p, Mode: 0600}
	if err := safePath(p); err != nil {
		return s, err
	}
	info, err := os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if !info.Mode().IsRegular() {
		return s, fmt.Errorf("not a regular file: %s", p)
	}
	s.Exists = true
	s.Mode = info.Mode().Perm()
	s.Data, err = os.ReadFile(p)
	return s, err
}
func writeFile(p string, b []byte, mode os.FileMode) error {
	if err := safePath(p); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(p), ".ai-cross-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(mode); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), p)
}
func transaction(next []snapshot) error {
	old := make([]snapshot, len(next))
	for i, s := range next {
		var err error
		old[i], err = capture(s.Path)
		if err != nil {
			return err
		}
	}
	for i, s := range next {
		err := put(s)
		if err != nil {
			for j := i; j >= 0; j-- {
				if e := put(old[j]); e != nil {
					err = errors.Join(err, fmt.Errorf("rollback %s: %w", old[j].Path, e))
				}
			}
			return err
		}
	}
	return nil
}
func put(s snapshot) error {
	if s.Exists {
		return writeFile(s.Path, s.Data, s.Mode)
	}
	err := os.Remove(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
func (a *app) apply(next []snapshot, content []byte, backup bool) (string, error) {
	if err := safePath(a.history); err != nil {
		return "", err
	}
	if err := safePath(a.data); err != nil {
		return "", err
	}
	if err := os.MkdirAll(a.data, 0700); err != nil {
		return "", err
	}
	lock := filepath.Join(a.data, "write.lock")
	f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", fmt.Errorf("another write may be active (%s): %w", lock, err)
	}
	f.Close()
	defer os.Remove(lock)
	if err := os.MkdirAll(a.history, 0700); err != nil {
		return "", err
	}
	name := time.Now().Format("20060102150405")
	for {
		_, e1 := os.Stat(filepath.Join(a.history, name))
		_, e2 := os.Stat(filepath.Join(a.history, name+".md"))
		if errors.Is(e1, os.ErrNotExist) && errors.Is(e2, os.ErrNotExist) {
			break
		}
		if e1 != nil && !errors.Is(e1, os.ErrNotExist) {
			return "", e1
		}
		if e2 != nil && !errors.Is(e2, os.ErrNotExist) {
			return "", e2
		}
		t, _ := time.Parse("20060102150405", name)
		name = t.Add(time.Second).Format("20060102150405")
	}
	stage, err := os.MkdirTemp(a.history, ".pending-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	manifest := []snapshot{}
	after := make([]snapshot, 0, len(next))
	for _, s := range next {
		old, e := capture(s.Path)
		if e != nil {
			return "", e
		}
		rel, e := filepath.Rel(a.root, s.Path)
		if e != nil {
			return "", e
		}
		if rel == ".ai-cross-manifest.json" {
			return "", errors.New("reserved instruction filename")
		}
		old.Path = rel
		saved := s
		saved.Path = rel
		after = append(after, saved)
		if backup && old.Exists {
			if e = writeFile(filepath.Join(stage, rel), old.Data, old.Mode); e != nil {
				return "", e
			}
		}
		old.Data = nil
		manifest = append(manifest, old)
	}
	meta, err := json.Marshal(manifest)
	if err != nil {
		return "", err
	}
	if backup {
		if err = writeFile(filepath.Join(stage, ".ai-cross-manifest.json"), meta, 0600); err != nil {
			return "", err
		}
	}
	afterData, err := json.Marshal(after)
	if err != nil {
		return "", err
	}
	next = append(next, snapshot{Path: filepath.Join(a.history, name+".json"), Data: afterData, Mode: 0600, Exists: true})
	record := filepath.Join(a.history, name+".md")
	next = append(next, snapshot{Path: record, Data: content, Mode: 0600, Exists: true})
	if backup {
		if err = os.Rename(stage, filepath.Join(a.history, name)); err != nil {
			return "", err
		}
	}
	if err = transaction(next); err != nil {

		return "", err
	}
	return name, nil
}
