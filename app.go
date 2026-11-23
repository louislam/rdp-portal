package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/louislam/rdp-portal/internal/core"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Version is overridden at build time with -ldflags "-X main.Version=x.y.z".
var Version = "2.0.0"

// ProfileView is what the frontend sees for one connection.
type ProfileView struct {
	Index    int    `json:"index"`
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Domain   string `json:"domain"`
	Password string `json:"password"`
	Filename string `json:"filename"`
	IsNew    bool   `json:"isNew"`
}

// PortalService is the HTTP-free bridge between Vue and the core package.
type PortalService struct {
	cfg   *core.Config
	newAt map[int]bool // indices that were added but never saved
}

func NewPortalService() (*PortalService, error) {
	return NewPortalServiceIn(".")
}

// NewPortalServiceIn loads config.json from folder; tests use a temp dir.
func NewPortalServiceIn(folder string) (*PortalService, error) {
	cfg, err := core.LoadConfig(folder)
	if err != nil {
		return nil, err
	}
	svc := &PortalService{cfg: cfg, newAt: map[int]bool{}}

	// Like the WinForms app: an empty config starts with one unsaved profile.
	if len(cfg.Profiles) == 0 {
		cfg.Profiles = append(cfg.Profiles, cfg.NewProfile())
		svc.newAt[0] = true
	}

	return svc, nil
}

func (s *PortalService) view(index int) (ProfileView, error) {
	p, err := s.cfg.IndexOf(index)
	if err != nil {
		return ProfileView{}, err
	}
	pw, err := p.Password()
	if err != nil {
		return ProfileView{}, err
	}
	hostname, port := core.SplitAddress(p.Computer)
	return ProfileView{
		Index:    index,
		Name:     p.DisplayName(),
		Hostname: hostname,
		Port:     port,
		Username: p.Username,
		Domain:   p.Domain,
		Password: pw,
		Filename: p.Filename,
		IsNew:    s.newAt[index],
	}, nil
}

// List returns every profile, in config order.
func (s *PortalService) List() ([]ProfileView, error) {
	out := make([]ProfileView, 0, len(s.cfg.Profiles))
	for i := range s.cfg.Profiles {
		v, err := s.view(i)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

// Add returns the unsaved row, creating one only when none is pending.
func (s *PortalService) Add() (ProfileView, error) {
	if _, index := s.cfg.Placeholder(); index >= 0 {
		s.newAt[index] = true
		return s.view(index)
	}

	s.cfg.Profiles = append(s.cfg.Profiles, s.cfg.NewProfile())
	index := len(s.cfg.Profiles) - 1
	s.newAt[index] = true
	return s.view(index)
}

// Save writes the edited fields, regenerates the rdp file and persists config.
func (s *PortalService) Save(index int, name, hostname, port, username, password, domain string) (ProfileView, error) {
	p, err := s.cfg.IndexOf(index)
	if err != nil {
		return ProfileView{}, err
	}

	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		return ProfileView{}, fmt.Errorf("hostname is required")
	}
	if port != "" && !core.IsValidPort(strings.TrimSpace(port)) {
		return ProfileView{}, fmt.Errorf("invalid port: %s", strings.TrimSpace(port))
	}

	name = strings.TrimSpace(name)
	if name == "" {
		name = hostname
	}
	if s.cfg.NameTaken(name, p) {
		return ProfileView{}, fmt.Errorf("a profile named %q already exists", name)
	}

	p.Name = name
	p.Computer = core.JoinAddress(hostname, port)
	p.Username = strings.TrimSpace(username)
	p.Domain = strings.TrimSpace(domain)
	if err := p.SetPassword(password); err != nil {
		return ProfileView{}, err
	}

	if err := p.PrepareRdpFile(); err != nil {
		return ProfileView{}, err
	}
	if err := s.cfg.Save(); err != nil {
		return ProfileView{}, err
	}

	delete(s.newAt, index)
	return s.view(index)
}

// Remove deletes the profile, its rdp file and persists config.
func (s *PortalService) Remove(index int) error {
	p, err := s.cfg.IndexOf(index)
	if err != nil {
		return err
	}
	p.DeleteRDPFile()

	s.cfg.Profiles = append(s.cfg.Profiles[:index], s.cfg.Profiles[index+1:]...)

	// Re-key the unsaved markers after the slice shift.
	rekeyed := map[int]bool{}
	for old := range s.newAt {
		if old < index {
			rekeyed[old] = true
		} else if old > index {
			rekeyed[old-1] = true
		}
	}
	s.newAt = rekeyed

	// The list is never empty: deleting the last row leaves a fresh one.
	if len(s.cfg.Profiles) == 0 {
		s.cfg.Profiles = append(s.cfg.Profiles, s.cfg.NewProfile())
		s.newAt[0] = true
	}

	return s.cfg.Save()
}

// Discard drops a never-saved profile (the Cancel button on a fresh row).
func (s *PortalService) Discard(index int) error {
	if !s.newAt[index] {
		return nil
	}
	return s.Remove(index)
}

// Connect saves the rdp file and runs mstsc.exe, blocking until it exits.
func (s *PortalService) Connect(index int) error {
	p, err := s.cfg.IndexOf(index)
	if err != nil {
		return err
	}
	if p.IsPlaceholder() {
		return fmt.Errorf("save the profile first")
	}
	if strings.TrimSpace(p.Computer) == "" {
		return fmt.Errorf("hostname is required")
	}
	if err := p.Connect(); err != nil {
		return fmt.Errorf("mstsc.exe failed: %w", err)
	}
	if !s.cfg.KeepOpening {
		application.Get().Quit()
	}
	return nil
}

// Options opens mstsc.exe in edit mode so the user can tweak resolution etc.
func (s *PortalService) Options(index int) error {
	p, err := s.cfg.IndexOf(index)
	if err != nil {
		return err
	}
	if p.IsPlaceholder() {
		return fmt.Errorf("save the profile first")
	}
	if err := p.ShowMoreOptions(); err != nil {
		return fmt.Errorf("mstsc.exe failed: %w", err)
	}
	return nil
}

// KeepOpening reports the "keep opening RDP Portal" checkbox state.
func (s *PortalService) KeepOpening() bool { return s.cfg.KeepOpening }

// SetKeepOpening persists the checkbox state.
func (s *PortalService) SetKeepOpening(v bool) error {
	s.cfg.KeepOpening = v
	return s.cfg.Save()
}

// AppVersion feeds the About dialog.
func (s *PortalService) AppVersion() string { return Version }

func (s *PortalService) startup() {
	log.Printf("RDP Portal %s, config in %s", Version, s.cfg.RdpDir())
}

