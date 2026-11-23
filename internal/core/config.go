package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config is the on-disk config.json. Field order and names match the C# app so
// an existing portable folder keeps working unchanged.
type Config struct {
	Profiles    []*Profile `json:"Profiles"`
	KeepOpening bool       `json:"KeepOpening"`

	folder     string
	configFile string
	rdpDir     string
}

// LoadConfig reads <folder>/config.json, creating it and rdp-files/ if missing.
func LoadConfig(folder string) (*Config, error) {
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return nil, err
	}

	cfg := &Config{
		folder:     folder,
		configFile: filepath.Join(folder, ConfigFilename),
		rdpDir:     filepath.Join(folder, RdpDirName),
		KeepOpening: true,
	}

	if _, err := os.Stat(cfg.configFile); os.IsNotExist(err) {
		if err := os.WriteFile(cfg.configFile, []byte("{}"), 0o644); err != nil {
			return nil, err
		}
	}

	if err := os.MkdirAll(cfg.rdpDir, 0o755); err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(cfg.configFile)
	if err != nil {
		return nil, err
	}

	loaded := *cfg
	loaded.KeepOpening = true
	if err := json.Unmarshal(raw, &loaded); err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", cfg.configFile, err)
	}
	loaded.folder = cfg.folder
	loaded.configFile = cfg.configFile
	loaded.rdpDir = cfg.rdpDir

	if loaded.Profiles == nil {
		loaded.Profiles = []*Profile{}
	}

	// Drop empty rows that an older build may have written to disk.
	kept := loaded.Profiles[:0]
	for _, p := range loaded.Profiles {
		if p.IsPlaceholder() {
			continue
		}
		kept = append(kept, p)
	}
	loaded.Profiles = kept

	for _, p := range loaded.Profiles {
		p.cfg = &loaded
	}

	if err := loaded.Save(); err != nil {
		return nil, err
	}

	return &loaded, nil
}

const (
	ConfigFilename = "config.json"
	RdpDirName     = "rdp-files"
)

// RdpDir is where generated .rdp files live.
func (c *Config) RdpDir() string { return c.rdpDir }

// Save writes config.json with two-space indentation, like the C# app did.
// Placeholder rows (added in the UI but never filled in) are not persisted.
func (c *Config) Save() error {
	persisted := make([]*Profile, 0, len(c.Profiles))
	for _, p := range c.Profiles {
		if p.IsPlaceholder() {
			continue
		}
		persisted = append(persisted, p)
	}

	onDisk := struct {
		Profiles    []*Profile `json:"Profiles"`
		KeepOpening bool       `json:"KeepOpening"`
	}{
		Profiles:    persisted,
		KeepOpening: c.KeepOpening,
	}

	out, err := json.MarshalIndent(onDisk, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.configFile, out, 0o644)
}

// NewProfile returns an unsaved profile bound to this config.
func (c *Config) NewProfile() *Profile {
	return &Profile{cfg: c}
}

// Placeholder returns the existing unsaved placeholder row, if any.
func (c *Config) Placeholder() (*Profile, int) {
	for i, p := range c.Profiles {
		if p.IsPlaceholder() {
			return p, i
		}
	}
	return nil, -1
}

// IndexOf returns the profile at index, or an error when out of range.
func (c *Config) IndexOf(index int) (*Profile, error) {
	if index < 0 || index >= len(c.Profiles) {
		return nil, fmt.Errorf("profile not found: %d", index)
	}
	return c.Profiles[index], nil
}
