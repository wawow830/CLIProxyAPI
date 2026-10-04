package config

import "testing"

func TestParseConfigBytesClaudePromptCacheTTL(t *testing.T) {
	for _, input := range []string{
		"claude:\n  prompt-cache-ttl: 5m\n",
		"config-version: 8\nupstream:\n  claude:\n    prompt-cache-ttl: 5m\n",
	} {
		cfg, err := ParseConfigBytes([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Claude.PromptCacheTTL != "5m" {
			t.Fatalf("PromptCacheTTL = %q, want 5m", cfg.Claude.PromptCacheTTL)
		}
	}
}
