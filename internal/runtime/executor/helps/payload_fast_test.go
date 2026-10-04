package helps

import (
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
	"github.com/tidwall/gjson"
)

func TestApplyPayloadConfigFastSuffixSetsPriorityTier(t *testing.T) {
	payload := []byte(`{"model":"gpt-6-sol","input":[]}`)
	got := ApplyPayloadConfigWithRoot(&config.Config{}, "gpt-6-sol", "codex", "", payload, nil, "gpt-6-sol-fast", "")
	if tier := gjson.GetBytes(got, "service_tier").String(); tier != "priority" {
		t.Fatalf("service_tier = %q, want priority", tier)
	}
	got = ApplyPayloadConfigWithRoot(&config.Config{}, "gpt-6-sol", "codex", "", payload, nil, "gpt-6-sol", "")
	if gjson.GetBytes(got, "service_tier").Exists() {
		t.Fatalf("unexpected service_tier on non-fast request: %s", got)
	}
	got = ApplyPayloadConfigWithRoot(&config.Config{}, "gpt-6-sol", "claude", "", payload, nil, "gpt-6-sol-fast", "")
	if gjson.GetBytes(got, "service_tier").Exists() {
		t.Fatalf("unexpected service_tier on non-codex protocol: %s", got)
	}
}
