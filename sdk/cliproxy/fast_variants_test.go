package cliproxy

import (
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
)

func TestCatalogAliasesAddFastVariantsForCodex(t *testing.T) {
	models := []*ModelInfo{{ID: "gpt-6-sol"}, {ID: "gpt-image-2"}, {ID: "codex-auto-review"}}
	got := applyModelCatalogAliasesForAuth(&config.Config{}, "codex", "oauth", nil, models)
	ids := map[string]bool{}
	for _, m := range got {
		ids[m.ID] = true
	}
	for _, want := range []string{"gpt-6-sol", "gpt-6-sol-fast", "gpt-image-2", "codex-auto-review"} {
		if !ids[want] {
			t.Errorf("missing %q in %v", want, ids)
		}
	}
	for _, bad := range []string{"gpt-image-2-fast", "codex-auto-review-fast"} {
		if ids[bad] {
			t.Errorf("unexpected %q", bad)
		}
	}
	if other := applyModelCatalogAliasesForAuth(&config.Config{}, "claude", "oauth", nil, []*ModelInfo{{ID: "gpt-x"}}); len(other) != 1 {
		t.Errorf("non-codex channel got fast variants: %d", len(other))
	}
}
