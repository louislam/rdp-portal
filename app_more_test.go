package main

import (
	"strings"
	"testing"
)

func TestAppVersion(t *testing.T) {
	svc, _ := newTestService(t)
	if got := svc.AppVersion(); got != Version {
		t.Fatalf("got %q, want %q", got, Version)
	}
}

func TestListViewCarriesAllFields(t *testing.T) {
	svc, _ := newTestService(t)

	if _, err := svc.Save(0, "server1", "server1", "3390", "alice", "pw", "LAB"); err != nil {
		t.Fatalf("save: %v", err)
	}

	views, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	v := views[0]
	if v.Name != "server1" || v.Hostname != "server1" || v.Port != "3390" ||
		v.Username != "alice" || v.Domain != "LAB" || v.Password != "pw" || v.Filename == "" {
		t.Fatalf("incomplete view: %+v", v)
	}
}

func TestOutOfRangeIndexes(t *testing.T) {
	svc, _ := newTestService(t)

	if _, err := svc.Save(9, "x", "x", "", "", "", ""); err == nil {
		t.Error("save on a missing index must fail")
	}
	if err := svc.Remove(9); err == nil {
		t.Error("remove on a missing index must fail")
	}
	if err := svc.Connect(9); err == nil {
		t.Error("connect on a missing index must fail")
	}
	if err := svc.Options(9); err == nil {
		t.Error("options on a missing index must fail")
	}
	if err := svc.Discard(9); err != nil {
		t.Errorf("discard on a missing index should be a no-op, got %v", err)
	}
}

func TestDiscardKeepsSavedProfiles(t *testing.T) {
	svc, _ := newTestService(t)

	if _, err := svc.Save(0, "server1", "server1", "", "", "", ""); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Discard only drops never-saved rows.
	if err := svc.Discard(0); err != nil {
		t.Fatalf("discard: %v", err)
	}
	views, _ := svc.List()
	if len(views) != 1 || views[0].Name != "server1" {
		t.Fatalf("saved profile was discarded: %+v", views)
	}
}

func TestPortIsNormalisedOnSave(t *testing.T) {
	cases := []struct {
		port, want string
	}{
		{"", "host1"},
		{"3389", "host1"},
		{"3390", "host1:3390"},
	}
	for _, c := range cases {
		svc, _ := newTestService(t)
		if _, err := svc.Save(0, "host1", "host1", c.port, "", "", ""); err != nil {
			t.Fatalf("save %q: %v", c.port, err)
		}
		views, _ := svc.List()
		if views[0].Port != strings.TrimPrefix(c.want, "host1:") && !(c.want == "host1" && views[0].Port == "") {
			t.Errorf("port %q stored as %q", c.port, views[0].Port)
		}
	}
}
