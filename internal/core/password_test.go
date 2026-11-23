package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The blob written into the .rdp file must be a DPAPI blob that mstsc.exe (same
// user) can decrypt back to the stored password. Anything else means a password
// prompt on every connection.
func TestRdpPasswordBlobDecrypts(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	const password = "P@ssw0rd with spaces and ümlaut"
	p := &Profile{Name: "server1", Computer: "server1", cfg: cfg}
	if err := p.SetPassword(password); err != nil {
		t.Fatalf("set password: %v", err)
	}
	if err := p.PrepareRdpFile(); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	raw, err := os.ReadFile(p.Filename)
	if err != nil {
		t.Fatalf("read rdp: %v", err)
	}

	var blob string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "password 51:b:") {
			blob = strings.TrimPrefix(line, "password 51:b:")
			break
		}
	}
	if blob == "" {
		t.Fatalf("no password line in %s", raw)
	}
	if !strings.HasPrefix(blob, "01000000D08C9DDF") {
		t.Fatalf("not a DPAPI blob: %.32s", blob)
	}

	got, err := decryptMstscPassword(blob)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != password {
		t.Fatalf("blob decrypts to %q, want %q", got, password)
	}
}

// A saved profile must never be readable without DPAPI: the rdp file holds the
// blob, config.json holds the DES ciphertext, neither holds the plaintext.
func TestNoPlaintextPasswordOnDisk(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	const password = "hunter2-secret"
	p := &Profile{Name: "server1", Computer: "server1", cfg: cfg}
	if err := p.SetPassword(password); err != nil {
		t.Fatalf("set password: %v", err)
	}
	cfg.Profiles = append(cfg.Profiles, p)
	if err := p.PrepareRdpFile(); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	for _, file := range []string{filepath.Join(dir, ConfigFilename), p.Filename} {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		if strings.Contains(string(raw), password) {
			t.Errorf("plaintext password found in %s", file)
		}
	}
}
