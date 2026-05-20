package auth

import (
	"strings"
	"testing"
)

func TestHashToken(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"simple token", "hello"},
		{"refresh token shape", "prt_abc123XYZ"},
		{"long token", strings.Repeat("a", 256)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := hashToken(tt.input)

			// SHA-256 hex output is always 64 characters
			if len(h) != 64 {
				t.Errorf("hashToken(%q) length = %d, want 64", tt.input, len(h))
			}
			// Must be lowercase hex only
			for _, c := range h {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					t.Errorf("hashToken(%q) contains non-hex char %q", tt.input, c)
				}
			}
		})
	}
}

func TestHashTokenDeterministic(t *testing.T) {
	token := "same-input-different-calls"
	h1 := hashToken(token)
	h2 := hashToken(token)
	if h1 != h2 {
		t.Errorf("hashToken is not deterministic: %q != %q", h1, h2)
	}
}

func TestHashTokenDistinct(t *testing.T) {
	pairs := [][2]string{
		{"token-a", "token-b"},
		{"", "x"},
		{"abc", "ABC"},
		{"prt_abc", "prt_ABC"},
	}
	for _, p := range pairs {
		h1 := hashToken(p[0])
		h2 := hashToken(p[1])
		if h1 == h2 {
			t.Errorf("hashToken(%q) == hashToken(%q), expected distinct hashes", p[0], p[1])
		}
	}
}

func TestRateLimitKeys(t *testing.T) {
	tests := []struct {
		ip    string
		email string
	}{
		{"127.0.0.1", "user@example.com"},
		{"192.168.1.1", "admin@school.edu"},
		{"10.0.0.1", "student@unifranz.edu.bo"},
	}

	for _, tt := range tests {
		t.Run(tt.ip+"/"+tt.email, func(t *testing.T) {
			attemptKey, cooldownKey := rateLimitKeys(tt.ip, tt.email)

			if !strings.HasPrefix(attemptKey, "rate_limit:") {
				t.Errorf("attemptKey %q does not start with 'rate_limit:'", attemptKey)
			}
			if !strings.HasPrefix(cooldownKey, "rate_cool:") {
				t.Errorf("cooldownKey %q does not start with 'rate_cool:'", cooldownKey)
			}
			// Suffix is 16 hex chars
			suffix := strings.TrimPrefix(attemptKey, "rate_limit:")
			if len(suffix) != 16 {
				t.Errorf("attemptKey suffix length = %d, want 16", len(suffix))
			}
		})
	}
}

func TestRateLimitKeysDeterministic(t *testing.T) {
	ip, email := "1.2.3.4", "test@test.com"
	a1, c1 := rateLimitKeys(ip, email)
	a2, c2 := rateLimitKeys(ip, email)
	if a1 != a2 || c1 != c2 {
		t.Error("rateLimitKeys is not deterministic")
	}
}

func TestRateLimitKeysIsolation(t *testing.T) {
	// Different ip+email combos must produce different keys
	pairs := [][2][2]string{
		{{"1.1.1.1", "a@b.com"}, {"1.1.1.2", "a@b.com"}},
		{{"1.1.1.1", "a@b.com"}, {"1.1.1.1", "c@b.com"}},
	}
	for _, p := range pairs {
		a1, _ := rateLimitKeys(p[0][0], p[0][1])
		a2, _ := rateLimitKeys(p[1][0], p[1][1])
		if a1 == a2 {
			t.Errorf("rateLimitKeys(%v) == rateLimitKeys(%v), expected distinct", p[0], p[1])
		}
	}
}
