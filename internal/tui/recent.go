package tui

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gkmz/sharp/pkg/tool"
)

const recentLimit = 5

type recentTools struct {
	ids     []string
	path    string
	warning string
}

func (r *recentTools) record(id string) bool {
	if id == "" || (len(r.ids) > 0 && r.ids[0] == id) {
		return false
	}
	ids := []string{id}
	for _, previous := range r.ids {
		if previous != id && len(ids) < recentLimit {
			ids = append(ids, previous)
		}
	}
	r.ids = ids
	return true
}

func loadRecentTools(path string, registry *tool.Registry) (recentTools, error) {
	r := recentTools{path: path}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return r, nil
	}
	if err != nil {
		return r, err
	}
	defer f.Close()
	// The file only contains five IDs; cap reads of malformed/foreign files.
	data, err := io.ReadAll(io.LimitReader(f, 64*1024+1))
	if err != nil {
		return r, err
	}
	if len(data) > 64*1024 {
		return r, fmt.Errorf("recent tools file is too large")
	}
	var saved struct {
		Tools []string `json:"tools"`
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		return r, err
	}
	seen := make(map[string]bool)
	for _, id := range saved.Tools {
		if _, ok := registry.Get(id); ok && !seen[id] {
			r.ids = append(r.ids, id)
			seen[id] = true
			if len(r.ids) == recentLimit {
				break
			}
		}
	}
	return r, nil
}

func (r recentTools) save() error {
	if r.path == "" {
		return nil
	}
	data, err := json.Marshal(struct {
		Tools []string `json:"tools"`
	}{r.ids})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(r.path), ".recent-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(append(data, '\n')); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), r.path)
}

func (m *model) initRecent() {
	dir, err := os.UserConfigDir()
	if err != nil {
		m.recent.warning = "History is available for this session only"
		return
	}
	m.recent, err = loadRecentTools(filepath.Join(dir, "sharp", "recent-tools.json"), m.registry)
	if err != nil {
		m.recent.warning = "Saved history could not be loaded; starting a new list"
	}
}

// Consume successful operations once at the root boundary, including delayed
// previews on a workspace that is no longer selected. Never infer use from View.
func (m *model) collectRecentUsage() {
	for _, page := range m.pages {
		id := page.takeToolUse()
		if _, ok := m.registry.Get(id); !ok {
			continue
		}
		changed := m.recent.record(id)
		if changed || m.recent.warning != "" {
			if err := m.recent.save(); err != nil {
				m.recent.warning = "History could not be saved; available for this session"
			} else if m.recent.path != "" {
				m.recent.warning = ""
			}
		}
	}
}
