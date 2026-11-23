package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestService(t *testing.T) (*PortalService, string) {
	t.Helper()
	dir := t.TempDir()
	svc, err := NewPortalServiceIn(dir)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return svc, dir
}

func TestStartupHasSinglePlaceholder(t *testing.T) {
	svc, _ := newTestService(t)

	views, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(views) != 1 || !views[0].IsNew {
		t.Fatalf("want one unsaved profile, got %+v", views)
	}
}

// Clicking New repeatedly must reuse the pending row, not stack placeholders.
func TestAddReusesPlaceholder(t *testing.T) {
	svc, _ := newTestService(t)

	for i := 0; i < 5; i++ {
		if _, err := svc.Add(); err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
	}

	views, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("placeholder rows piled up: %d", len(views))
	}
}

// Saving a brand new (placeholder) profile used to panic on a nil config.
func TestSaveNewProfile(t *testing.T) {
	svc, dir := newTestService(t)

	if _, err := svc.Add(); err != nil {
		t.Fatalf("add: %v", err)
	}

	view, err := svc.Save(0, "server1", "server1", "3390", "alice", "secret", "LAB")
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if view.Name != "server1" || view.Hostname != "server1" || view.Port != "3390" {
		t.Fatalf("unexpected view: %+v", view)
	}
	if view.IsNew {
		t.Error("profile should no longer be a placeholder")
	}

	// .rdp file written next to the config
	entries, err := os.ReadDir(filepath.Join(dir, "rdp-files"))
	if err != nil {
		t.Fatalf("rdp-files: %v", err)
	}
	if len(entries) != 1 || !strings.HasSuffix(entries[0].Name(), ".rdp") {
		t.Fatalf("want one .rdp file, got %+v", entries)
	}

	// password is encrypted in config.json, not stored in the clear
	raw, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(raw), "secret") {
		t.Errorf("password stored in clear: %s", raw)
	}
	if !strings.Contains(string(raw), "EncryptedPassword") {
		t.Errorf("encrypted password missing: %s", raw)
	}
}

func TestSaveValidation(t *testing.T) {
	cases := []struct {
		name           string
		name_, host    string
		port, existing string
		wantErr        string
	}{
		{name: "no hostname", host: "", wantErr: "hostname is required"},
		{name: "bad port", host: "server1", port: "99999", wantErr: "invalid port"},
		{name: "port not numeric", host: "server1", port: "abc", wantErr: "invalid port"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, _ := newTestService(t)
			_, err := svc.Save(0, c.name_, c.host, c.port, "", "", "")
			if err == nil {
				t.Fatalf("expected an error")
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("got %q, want %q", err, c.wantErr)
			}
		})
	}
}

func TestSaveFallsBackToHostnameAndRejectsDuplicates(t *testing.T) {
	svc, _ := newTestService(t)

	// empty name -> hostname
	if _, err := svc.Save(0, "", "server1", "", "", "", ""); err != nil {
		t.Fatalf("save first: %v", err)
	}
	views, _ := svc.List()
	if views[0].Name != "server1" {
		t.Fatalf("name should fall back to hostname, got %q", views[0].Name)
	}

	// second profile with the same name is rejected
	if _, err := svc.Add(); err != nil {
		t.Fatalf("add: %v", err)
	}
	_, err := svc.Save(1, "SERVER1", "server2", "", "", "", "")
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("duplicate name accepted: %v", err)
	}
}

func TestDeleteLastProfileKeepsOneRow(t *testing.T) {
	svc, _ := newTestService(t)

	if _, err := svc.Save(0, "server1", "server1", "", "", "", ""); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := svc.Remove(0); err != nil {
		t.Fatalf("remove: %v", err)
	}

	views, _ := svc.List()
	if len(views) != 1 || !views[0].IsNew {
		t.Fatalf("want a fresh placeholder, got %+v", views)
	}
}

func TestConnectAndOptionsRejectUnsavedProfile(t *testing.T) {
	svc, _ := newTestService(t)

	if err := svc.Connect(0); err == nil || !strings.Contains(err.Error(), "save the profile first") {
		t.Fatalf("connect on placeholder: %v", err)
	}
	if err := svc.Options(0); err == nil || !strings.Contains(err.Error(), "save the profile first") {
		t.Fatalf("options on placeholder: %v", err)
	}
}

func TestKeepOpeningPersists(t *testing.T) {
	svc, dir := newTestService(t)

	if err := svc.SetKeepOpening(false); err != nil {
		t.Fatalf("set: %v", err)
	}

	reloaded, err := NewPortalServiceIn(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.KeepOpening() {
		t.Fatal("KeepOpening did not persist")
	}
}
