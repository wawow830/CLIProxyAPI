package auth

import (
	"testing"

	internalconfig "github.com/router-for-me/CLIProxyAPI/v8/internal/config"
)

func TestResolveOAuthUpstreamModel_FastSuffix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		channel string
		input   string
		want    string
	}{
		{"codex", "gpt-6-sol-fast", "gpt-6-sol"},
		{"codex", "gpt-6-sol-fast(high)", "gpt-6-sol(high)"},
		{"codex", "gpt-6-sol", ""},
		{"claude", "claude-opus-fast", ""},
	}
	for _, tt := range tests {
		mgr := NewManager(nil, nil, nil)
		mgr.SetConfig(&internalconfig.Config{})
		got := mgr.resolveOAuthUpstreamModel(createAuthForChannel(tt.channel), tt.input)
		if got != tt.want {
			t.Errorf("resolveOAuthUpstreamModel(%s, %q) = %q, want %q", tt.channel, tt.input, got, tt.want)
		}
	}
}

func TestResolveOAuthUpstreamModel_ExplicitAliasBeatsFastSuffix(t *testing.T) {
	t.Parallel()

	mgr := NewManager(nil, nil, nil)
	mgr.SetConfig(&internalconfig.Config{})
	mgr.SetOAuthModelAlias(map[string][]internalconfig.OAuthModelAlias{
		"codex": {{Name: "gpt-special", Alias: "gpt-6-sol-fast"}},
	})
	if got := mgr.resolveOAuthUpstreamModel(createAuthForChannel("codex"), "gpt-6-sol-fast"); got != "gpt-special" {
		t.Errorf("got %q, want gpt-special", got)
	}
}
