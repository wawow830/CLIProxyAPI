package fastmode

import "testing"

func TestTrim(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"gpt-6-sol-fast", "gpt-6-sol", true},
		{"GPT-6-SOL-FAST", "GPT-6-SOL", true},
		{"gpt-6-sol-fast(high)", "gpt-6-sol(high)", true},
		{"gpt-6-sol", "", false},
		{"-fast", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := Trim(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("Trim(%q) = %q,%v want %q,%v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestEligible(t *testing.T) {
	for id, want := range map[string]bool{
		"gpt-6.1-sol":       true,
		"gpt-6.1-sol-fast":  false,
		"gpt-image-2":       false,
		"codex-auto-review": false,
		"claude-opus-4-7":   false,
	} {
		if got := Eligible(id); got != want {
			t.Errorf("Eligible(%q) = %v want %v", id, got, want)
		}
	}
}
