package core

import "strings"

// NameTaken reports whether another profile already uses this name
// (case-insensitive, so Windows-like behaviour).
func (c *Config) NameTaken(name string, except *Profile) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, other := range c.Profiles {
		if other == except {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(other.Name), name) {
			return true
		}
	}
	return false
}
