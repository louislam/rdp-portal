package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// A row added in the UI but never filled in must not reach config.json.
func TestSaveSkipsPlaceholders(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	cfg.Profiles = append(cfg.Profiles,
		&Profile{Name: "server1", Computer: "server1"},
		&Profile{},
		&Profile{},
	)

	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, ConfigFilename))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var onDisk struct {
		Profiles []struct {
			Name string
		}
		KeepOpening bool
	}
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(onDisk.Profiles) != 1 || onDisk.Profiles[0].Name != "server1" {
		t.Fatalf("placeholders were persisted: %s", raw)
	}
	if !onDisk.KeepOpening {
		t.Errorf("KeepOpening default lost: %s", raw)
	}
}

// Empty rows written by an older build are dropped on load.
func TestLoadDropsPlaceholders(t *testing.T) {
	dir := t.TempDir()
	seed := `{"Profiles":[{"Name":"","Computer":"","Username":"","Domain":"","EncryptedPassword":"","Filename":""},{"Name":"server1","Computer":"server1"}],"KeepOpening":false}`
	if err := os.WriteFile(filepath.Join(dir, ConfigFilename), []byte(seed), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(cfg.Profiles) != 1 || cfg.Profiles[0].Name != "server1" {
		t.Fatalf("got %d profiles", len(cfg.Profiles))
	}
	if cfg.KeepOpening {
		t.Errorf("KeepOpening not read")
	}
}
