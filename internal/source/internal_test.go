package source

import "testing"

func TestIsMutableRef(t *testing.T) {
	cases := map[string]bool{
		"main":    true, // branch
		"v1.2.0":  true, // tag
		"v1":      true, // floating major tag
		"1a2b3c4": true, // short SHA (not 40 chars)
		"0123456789abcdef0123456789abcdef01234567": false, // full 40-hex SHA
		"0123456789ABCDEF0123456789abcdef01234567": false, // uppercase hex
		"z123456789abcdef0123456789abcdef01234567": true,  // 40 chars, non-hex
	}
	for ref, want := range cases {
		if got := isMutableRef(ref); got != want {
			t.Errorf("isMutableRef(%q) = %v, want %v", ref, got, want)
		}
	}
}
