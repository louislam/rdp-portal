package core

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Profile is one saved connection. The JSON shape matches the C# app.
type Profile struct {
	Name              string `json:"Name"`
	Computer          string `json:"Computer"`
	Username          string `json:"Username"`
	Domain            string `json:"Domain"`
	EncryptedPassword string `json:"EncryptedPassword"`
	Filename          string `json:"Filename"`

	cfg *Config
}

// DisplayName is what the list shows, matching the C# Name getter.
func (p *Profile) DisplayName() string {
	if p.Name == "" {
		return "<New Profile>"
	}
	return p.Name
}

// IsPlaceholder reports a row that was added in the UI but never saved.
func (p *Profile) IsPlaceholder() bool {
	return p.Name == "" && p.Computer == "" && p.Filename == "" && p.EncryptedPassword == ""
}

// Password decrypts the stored password; an empty blob means no password.
func (p *Profile) Password() (string, error) {
	if p.EncryptedPassword == "" {
		return "", nil
	}
	return DecryptConfigPassword(p.EncryptedPassword)
}

// SetPassword encrypts and stores the password the same way the C# app did.
func (p *Profile) SetPassword(plain string) error {
	if plain == "" {
		p.EncryptedPassword = ""
		return nil
	}
	enc, err := EncryptConfigPassword(plain)
	if err != nil {
		return err
	}
	p.EncryptedPassword = enc
	return nil
}

// PrepareRdpFile rewrites the .rdp file: it keeps whatever the user configured
// through mstsc.exe, but overwrites the connection fields, the encrypted
// password and the window position.
func (p *Profile) PrepareRdpFile() error {
	if p.cfg == nil {
		return errors.New("profile is not attached to a config")
	}

	justCreated := false

	if p.Filename == "" {
		for i := 0; ; i++ {
			suffix := ""
			if i > 0 {
				suffix = "_" + strconv.Itoa(i)
			}
			name := filepath.Join(p.cfg.RdpDir(), sanitizeFilename(p.DisplayName())+suffix+".rdp")
			if _, err := os.Stat(name); os.IsNotExist(err) {
				if err := os.WriteFile(name, nil, 0o644); err != nil {
					return err
				}
				p.Filename = name
				justCreated = true
				break
			}
		}
	}

	if _, err := os.Stat(p.Filename); os.IsNotExist(err) {
		if err := os.WriteFile(p.Filename, nil, 0o644); err != nil {
			return err
		}
		justCreated = true
	}

	existing, err := os.ReadFile(p.Filename)
	if err != nil {
		return err
	}

	removePrefixes := []string{
		"full address:",
		"username:",
		"password",
		"domain:",
		"winposstr",
	}

	width, height := 1280, 720
	result := make([]string, 0, 24)

	for _, line := range strings.Split(strings.ReplaceAll(string(existing), "\r\n", "\n"), "\n") {
		if line == "" {
			continue
		}

		keep := true
		for _, prefix := range removePrefixes {
			if strings.HasPrefix(line, prefix) {
				keep = false
				break
			}
		}

		if strings.HasPrefix(line, "desktopwidth:i:") {
			if v, err := strconv.Atoi(strings.TrimPrefix(line, "desktopwidth:i:")); err == nil {
				width = v
			}
		}
		if strings.HasPrefix(line, "desktopheight:i:") {
			if v, err := strconv.Atoi(strings.TrimPrefix(line, "desktopheight:i:")); err == nil {
				height = v
			}
		}

		if keep {
			result = append(result, line)
		}
	}

	if p.Computer != "" {
		result = append(result, "full address:s:"+p.Computer)
	}

	if p.Username != "" {
		result = append(result, "username:s:"+p.Username)
	}

	pw, err := p.Password()
	if err != nil {
		return err
	}
	if pw != "" {
		blob, err := EncryptMstscPassword(pw)
		if err != nil {
			return fmt.Errorf("cannot encrypt password for mstsc: %w", err)
		}
		result = append(result, "password 51:b:"+blob)
	}

	if p.Domain != "" {
		result = append(result, "domain:s:"+p.Domain)
	}

	// Centre the session window on the primary screen, like the C# app.
	screenW, screenH := PrimaryScreenSize()
	const xBuffer, yBuffer = 10, 25
	left := screenW/2 - width/2 - xBuffer
	top := screenH/2 - height/2 - yBuffer
	right := screenW/2 + width/2 + xBuffer
	bottom := screenH/2 + height/2 + yBuffer
	result = append(result, fmt.Sprintf("winposstr:s:0,1,%d,%d,%d,%d", left, top, right, bottom))

	if justCreated {
		result = append(result,
			"desktopwidth:i:1280",
			"desktopheight:i:720",
			"use multimon:i:0",
			"screen mode id:i:1",
			"authentication level:i:0",
			"prompt for credentials:i:0",
			"promptcredentialonce:i:0",
		)
	}

	return os.WriteFile(p.Filename, []byte(strings.Join(result, "\r\n")+"\r\n"), 0o644)
}

// Connect writes the rdp file and launches mstsc.exe, blocking until it exits.
func (p *Profile) Connect() error {
	if strings.TrimSpace(p.Computer) == "" {
		return fmt.Errorf("invalid connection")
	}
	if err := p.PrepareRdpFile(); err != nil {
		return err
	}
	cmd := exec.Command("mstsc.exe", p.Filename)
	return cmd.Run()
}

// ShowMoreOptions opens mstsc.exe in edit mode for the generated rdp file.
func (p *Profile) ShowMoreOptions() error {
	if p.Filename == "" {
		if err := p.PrepareRdpFile(); err != nil {
			return err
		}
	}
	cmd := exec.Command("mstsc.exe", "/edit", p.Filename)
	return cmd.Run()
}

// DeleteRDPFile removes the generated rdp file, ignoring errors like the C# app.
func (p *Profile) DeleteRDPFile() {
	if p.Filename == "" {
		return
	}
	_ = os.Remove(p.Filename)
}

func sanitizeFilename(name string) string {
	invalid := `<>:"/\|?*`
	out := strings.Map(func(r rune) rune {
		if strings.ContainsRune(invalid, r) || r < 32 {
			return '_'
		}
		return r
	}, name)
	return strings.TrimSpace(out)
}

// randomName produces an 8 character A-Z0-9 suffix for the rdp filename.
func randomName(n int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		for i := range buf {
			buf[i] = 0
		}
	}
	for i := range buf {
		buf[i] = chars[int(buf[i])%len(chars)]
	}
	return string(buf)
}
