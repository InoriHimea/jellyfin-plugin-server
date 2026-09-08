package handler

import "testing"

func TestIsValidJellyfinVersion(t *testing.T) {
	valid := []string{"10.11.8", "10.11.8.0", "10.11", "v10.10.7"}
	invalid := []string{"", "10", "10.", "10.11.x", "abc", "10.11.8-", "1.2.3.4.5"}
	for _, v := range valid {
		if !isValidJellyfinVersion(v) {
			t.Errorf("isValidJellyfinVersion(%q) = false, want true", v)
		}
	}
	for _, v := range invalid {
		if isValidJellyfinVersion(v) {
			t.Errorf("isValidJellyfinVersion(%q) = true, want false", v)
		}
	}
}
