package core

import (
	"strconv"
	"strings"
)

// DefaultRDPPort is what mstsc.exe assumes when no port is given.
const DefaultRDPPort = "3389"

// SplitAddress splits a stored "full address" into host and port.
// Supported: "host", "host:3390", "[::1]:3390", "fe80::1".
func SplitAddress(address string) (host, port string) {
	address = strings.TrimSpace(address)

	if strings.HasPrefix(address, "[") {
		if end := strings.Index(address, "]"); end > 0 {
			host = address[1:end]
			rest := strings.TrimSpace(address[end+1:])
			if strings.HasPrefix(rest, ":") {
				if p := strings.TrimSpace(rest[1:]); isPort(p) {
					return host, p
				}
			}
			return host, ""
		}
	}

	if i := strings.LastIndex(address, ":"); i > 0 && strings.Count(address, ":") == 1 {
		if p := strings.TrimSpace(address[i+1:]); isPort(p) {
			return strings.TrimSpace(address[:i]), p
		}
	}

	return address, ""
}

// JoinAddress rebuilds the "full address" value; the default port is omitted so
// the rdp file looks the same as before the split.
func JoinAddress(host, port string) string {
	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)

	switch {
	case port == "" || port == DefaultRDPPort:
		return host
	case strings.Contains(host, ":") && !strings.HasPrefix(host, "["):
		return "[" + host + "]:" + port
	default:
		return host + ":" + port
	}
}

func isPort(s string) bool {
	if s == "" || len(s) > 5 {
		return false
	}
	n, err := strconv.Atoi(s)
	return err == nil && n > 0 && n <= 65535
}

// IsValidPort is the exported form used by the UI layer for error messages.
func IsValidPort(s string) bool { return isPort(s) }
