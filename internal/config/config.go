package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gkmz/sharp/internal/theme"
	"gopkg.in/yaml.v3"
)

const maxConfigBytes = 1024 * 1024

var userConfigDir = os.UserConfigDir

// Config retains both supported values and the YAML tree needed for safe edits.
type Config struct {
	path        string
	document    yaml.Node
	selected    string
	definitions map[string]theme.Definition
	rename      func(string, string) error
}

// DefaultPath returns Sharp's configuration file in the OS user config tree.
func DefaultPath() (string, error) {
	dir, err := userConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(dir, "sharp", "config.yaml"), nil
}

// Load reads path without treating a missing file as an error.
func Load(path string) (*Config, error) {
	cfg := &Config{
		path:        path,
		definitions: map[string]theme.Definition{},
		rename:      os.Rename,
	}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		cfg.document = emptyDocument()
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open theme config %s: %w", path, err)
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxConfigBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read theme config %s: %w", path, err)
	}
	if len(data) > maxConfigBytes {
		return nil, fmt.Errorf("read theme config %s: file exceeds %d bytes", path, maxConfigBytes)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		cfg.document = emptyDocument()
		return cfg, nil
	}
	if err := yaml.Unmarshal(data, &cfg.document); err != nil {
		return nil, fmt.Errorf("parse theme config %s: %w", path, err)
	}
	var decoded struct {
		Theme  string                      `yaml:"theme"`
		Themes map[string]theme.Definition `yaml:"themes"`
	}
	if err := cfg.document.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decode theme config %s: %w", path, err)
	}
	cfg.selected = decoded.Theme
	if decoded.Themes != nil {
		cfg.definitions = decoded.Themes
	}
	return cfg, nil
}

// SelectedTheme returns the persisted stable theme ID.
func (c *Config) SelectedTheme() string {
	if c == nil {
		return ""
	}
	return c.selected
}

// ThemeDefinitions returns a deep copy suitable for registry validation.
func (c *Config) ThemeDefinitions() map[string]theme.Definition {
	definitions := make(map[string]theme.Definition)
	if c == nil {
		return definitions
	}
	for id, definition := range c.definitions {
		definition.Light = cloneColors(definition.Light)
		definition.Dark = cloneColors(definition.Dark)
		definitions[id] = definition
	}
	return definitions
}

// Path returns the file used for persistence and error reporting.
func (c *Config) Path() string {
	if c == nil {
		return ""
	}
	return c.path
}

// SaveTheme atomically changes only the root theme scalar in the YAML tree.
func (c *Config) SaveTheme(id string) error {
	if c == nil {
		return errors.New("theme config is unavailable")
	}
	root := c.mappingRoot()
	setMappingScalar(root, "theme", id)

	var data bytes.Buffer
	encoder := yaml.NewEncoder(&data)
	encoder.SetIndent(2)
	if err := encoder.Encode(&c.document); err != nil {
		return fmt.Errorf("encode theme config %s: %w", c.path, err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("encode theme config %s: %w", c.path, err)
	}

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create theme config directory %s: %w", dir, err)
	}
	temporary, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary theme config in %s: %w", dir, err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("secure temporary theme config %s: %w", temporaryPath, err)
	}
	if _, err := temporary.Write(data.Bytes()); err != nil {
		temporary.Close()
		return fmt.Errorf("write temporary theme config %s: %w", temporaryPath, err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary theme config %s: %w", temporaryPath, err)
	}
	if err := c.rename(temporaryPath, c.path); err != nil {
		return fmt.Errorf("replace theme config %s: %w", c.path, err)
	}
	c.selected = id
	return nil
}

func (c *Config) mappingRoot() *yaml.Node {
	if c.document.Kind != yaml.DocumentNode {
		c.document = emptyDocument()
	}
	if len(c.document.Content) == 0 || c.document.Content[0].Kind != yaml.MappingNode {
		c.document.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}
	return c.document.Content[0]
}

func emptyDocument() yaml.Node {
	return yaml.Node{
		Kind:    yaml.DocumentNode,
		Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}},
	}
}

func setMappingScalar(mapping *yaml.Node, key, value string) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			existing := mapping.Content[i+1]
			existing.Kind = yaml.ScalarNode
			existing.Tag = "!!str"
			existing.Value = value
			return
		}
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)
}

func cloneColors(colors map[string]string) map[string]string {
	if colors == nil {
		return nil
	}
	clone := make(map[string]string, len(colors))
	for role, color := range colors {
		clone[role] = color
	}
	return clone
}
