package core

import "testing"

func TestSplitAddress(t *testing.T) {
	cases := []struct {
		in, host, port string
	}{
		{"server1", "server1", ""},
		{"server1:3390", "server1", "3390"},
		{" server1:3390 ", "server1", "3390"},
		{"server1:3389", "server1", "3389"},
		{"example.com:3391", "example.com", "3391"},
		{"[::1]:3390", "::1", "3390"},
		{"[::1]", "::1", ""},
		{"fe80::1", "fe80::1", ""},
		{"", "", ""},
	}
	for _, c := range cases {
		host, port := SplitAddress(c.in)
		if host != c.host || port != c.port {
			t.Errorf("SplitAddress(%q) = %q, %q; want %q, %q", c.in, host, port, c.host, c.port)
		}
	}
}

func TestJoinAddress(t *testing.T) {
	cases := []struct {
		host, port, want string
	}{
		{"server1", "", "server1"},
		{"server1", "3389", "server1"},
		{"server1", "3390", "server1:3390"},
		{"::1", "3390", "[::1]:3390"},
		{"example.com", " 3391 ", "example.com:3391"},
	}
	for _, c := range cases {
		if got := JoinAddress(c.host, c.port); got != c.want {
			t.Errorf("JoinAddress(%q, %q) = %q; want %q", c.host, c.port, got, c.want)
		}
	}
}

func TestAddressRoundTrip(t *testing.T) {
	for _, address := range []string{"server1", "server1:3390", "[::1]:3392"} {
		if got := JoinAddress2(SplitAddress(address)); got != address {
			t.Errorf("round trip %q -> %q", address, got)
		}
	}
}

// JoinAddress2 is a tiny helper so the round-trip test reads well.
func JoinAddress2(host, port string) string { return JoinAddress(host, port) }

