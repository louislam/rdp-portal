package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	cases := map[string]string{
		"server 1":      "server 1",
		`a/b\c:d*e?f"g<h>i|j`: "a_b_c_d_e_f_g_h_i_j",
		"  padded  ":    "padded",
		"tab\there":     "tab_here",
	}
	for in, want := range cases {
		if got := sanitizeFilename(in); got != want {
			t.Errorf("sanitizeFilename(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRandomName(t *testing.T) {
	name := randomName(8)
	if len(name) != 8 {
		t.Fatalf("want 8 chars, got %q", name)
	}
	for _, r := range name {
		if !strings.ContainsRune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", r) {
			t.Fatalf("unexpected character %q in %q", r, name)
		}
	}
}

func TestPrepareRdpFileKeepsUserSettings(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// A file mstsc.exe already wrote, with a custom resolution and other keys.
	existing := strings.Join([]string{
		"full address:s:old",
		"username:s:old",
		"desktopwidth:i:1920",
		"desktopheight:i:1080",
		"audiomode:i:2",
		"winposstr:s:0,1,0,0,10,10",
		"screen mode id:i:2",
	}, "\r\n")
	path := filepath.Join(cfg.RdpDir(), "kept.rdp")
	if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	p := &Profile{Name: "kept", Computer: "server1", cfg: cfg, Filename: path}
	if err := p.PrepareRdpFile(); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	raw, _ := os.ReadFile(path)
	text := string(raw)

	for _, want := range []string{
		"audiomode:i:2",       // untouched setting survives
		"screen mode id:i:2",  // not overwritten on an existing file
		"full address:s:server1",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q\n%s", want, text)
		}
	}
	for _, unwanted := range []string{"full address:s:old", "username:s:old"} {
		if strings.Contains(text, unwanted) {
			t.Errorf("stale key kept: %q", unwanted)
		}
	}

	// winposstr must be recalculated, not the old one.
	if strings.Contains(text, "winposstr:s:0,1,0,0,10,10") {
		t.Error("winposstr was not recalculated")
	}
}

func TestDeleteRDPFile(t *testing.T) {
	dir := t.TempDir()
	cfg, _ := LoadConfig(dir)
	path := filepath.Join(cfg.RdpDir(), "gone.rdp")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	p := &Profile{Filename: path, cfg: cfg}
	p.DeleteRDPFile()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file still there")
	}

	// No filename and a missing file must not panic.
	(&Profile{cfg: cfg}).DeleteRDPFile()
	(&Profile{Filename: filepath.Join(dir, "nope.rdp"), cfg: cfg}).DeleteRDPFile()
}

func TestPrepareRdpFileWithoutConfig(t *testing.T) {
	if err := (&Profile{Name: "x"}).PrepareRdpFile(); err == nil {
		t.Fatal("expected an error for a profile with no config")
	}
}

func TestDecryptErrors(t *testing.T) {
	if _, err := DecryptConfigPassword("not base64!"); err == nil {
		t.Error("expected error for invalid base64")
	}
	// Valid base64, wrong length.
	if _, err := DecryptConfigPassword("AAAA"); err == nil {
		t.Error("expected error for short ciphertext")
	}
	// Valid length, broken padding.
	if _, err := DecryptConfigPassword("AAAAAAAAAAAAAAAAAAAAAA=="); err == nil {
		t.Error("expected error for broken padding")
	}
}

func TestEncryptEmptyMstscPassword(t *testing.T) {
	if _, err := EncryptMstscPassword(""); err == nil {
		t.Fatal("expected an error for an empty password")
	}
}

func TestPasswordSetterClearsValue(t *testing.T) {
	p := &Profile{}
	if err := p.SetPassword("secret"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if p.EncryptedPassword == "" {
		t.Fatal("password not encrypted")
	}
	if pw, err := p.Password(); err != nil || pw != "secret" {
		t.Fatalf("roundtrip: %q %v", pw, err)
	}
	if err := p.SetPassword(""); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if p.EncryptedPassword != "" {
		t.Fatal("password not cleared")
	}
	if pw, _ := p.Password(); pw != "" {
		t.Fatalf("empty password returned %q", pw)
	}
}

func TestDisplayNamePlaceholder(t *testing.T) {
	if got := (&Profile{}).DisplayName(); got != "<New Profile>" {
		t.Fatalf("got %q", got)
	}
	if got := (&Profile{Name: "server1"}).DisplayName(); got != "server1" {
		t.Fatalf("got %q", got)
	}
}

func TestNameTaken(t *testing.T) {
	a := &Profile{Name: "Server1"}
	b := &Profile{Name: "other"}
	cfg := &Config{Profiles: []*Profile{a, b}}

	if !cfg.NameTaken("server1", b) {
		t.Error("case-insensitive match expected")
	}
	if cfg.NameTaken("server1", a) {
		t.Error("profile must not collide with itself")
	}
	if cfg.NameTaken("", nil) {
		t.Error("empty name is never taken")
	}
	if cfg.NameTaken("fresh", nil) {
		t.Error("unknown name reported as taken")
	}
}
