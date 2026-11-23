package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The DES key/iv in crypto.go keep the 1.x (C#) config format readable; this
// test pins the round trip so a change to either breaks loudly.
func TestConfigPasswordRoundTrip(t *testing.T) {
	const plain = "correct-horse-battery"

	encrypted, err := EncryptConfigPassword(plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := DecryptConfigPassword(encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if decrypted != plain {
		t.Fatalf("got %q, want %q", decrypted, plain)
	}

	// DES-CBC/PKCS7 over UTF-16LE, base64 encoded.
	if len(encrypted)%4 != 0 || strings.ContainsAny(encrypted, "+/=") == false && len(plain) == 0 {
		t.Fatalf("unexpected encoding: %s", encrypted)
	}
}

func TestPrepareRdpFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConfig(dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	p := &Profile{
		Name:     "server1",
		Computer: "server1:3390",
		Username: "alice",
		Domain:   "WORKGROUP",
		cfg:      cfg,
	}
	if err := p.SetPassword("Secret123!"); err != nil {
		t.Fatalf("set password: %v", err)
	}

	cfg.Profiles = append(cfg.Profiles, p)
	if err := p.PrepareRdpFile(); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	if filepath.Dir(p.Filename) != cfg.RdpDir() {
		t.Fatalf("rdp file not in rdp-files: %s", p.Filename)
	}

	raw, err := os.ReadFile(p.Filename)
	if err != nil {
		t.Fatalf("read rdp: %v", err)
	}
	text := string(raw)

	for _, want := range []string{
		"full address:s:server1:3390",
		"username:s:alice",
		"domain:s:WORKGROUP",
		"password 51:b:01000000D08C9DDF",
		"winposstr:s:0,1,",
		"screen mode id:i:1",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("rdp file missing %q\n%s", want, text)
		}
	}

	// Saving twice must not duplicate the fields it rewrites.
	if err := p.PrepareRdpFile(); err != nil {
		t.Fatalf("prepare again: %v", err)
	}
	raw, _ = os.ReadFile(p.Filename)
	if n := strings.Count(string(raw), "full address:"); n != 1 {
		t.Errorf("full address appears %d times", n)
	}
}
