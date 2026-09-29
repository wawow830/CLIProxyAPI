package executor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v8/internal/runtime/executor/helps"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
	"github.com/tidwall/gjson"
)

func TestClaudeSessionDateKeepsWirePrefixAcrossMidnight(t *testing.T) {
	dir := t.TempDir()
	before := time.Date(2026, 9, 29, 23, 59, 59, 0, time.FixedZone("AEST", 10*60*60))
	payload := []byte(`{"model":"claude-opus-5-5","messages":[{"role":"user","content":"midnight prefix test"}]}`)
	var previous string
	for _, now := range []time.Time{before, before.Add(time.Minute), before.Add(24 * time.Hour)} {
		pinned, err := helps.ClaudeSessionDate(dir, nil, payload, now)
		if err != nil {
			t.Fatal(err)
		}
		out := injectClaudeCodeCurrentDate(payload, claudeCodeLocalDate(pinned))
		prefix := gjson.GetBytes(out, "messages.0").Raw
		if previous != "" && prefix != previous {
			t.Fatalf("historical prefix changed:\n%s\n%s", previous, prefix)
		}
		previous = prefix
		if got := gjson.GetBytes(out, "messages.0.content.0.text").String(); got != claudeCodeCurrentDateReminder(claudeCodeLocalDate(before)) {
			t.Fatalf("unexpected date reminder: %q", got)
		}
	}
}

func TestClaudeSessionDateUsedByCloakingAndNativeBypassed(t *testing.T) {
	cfg := &config.Config{AuthDir: t.TempDir()}
	payload := []byte(`{"model":"claude-opus-5-5","messages":[{"role":"user","content":"persisted session"}]}`)
	initial := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)
	if _, err := helps.ClaudeSessionDate(cfg.AuthDir, nil, payload, initial); err != nil {
		t.Fatal(err)
	}
	auth := &cliproxyauth.Auth{ID: "session-date-test", Provider: "claude"}
	out, cloaked, err := applyCloaking(context.Background(), cfg, auth, payload, "sk-ant-oat-test", false, false)
	if err != nil || !cloaked {
		t.Fatalf("cloaked=%v error=%v", cloaked, err)
	}
	if got := gjson.GetBytes(out, "messages.0.content.0.text").String(); got != claudeCodeCurrentDateReminder(claudeCodeLocalDate(initial)) {
		t.Fatalf("cloaking ignored persisted date: %q", got)
	}
	out, cloaked, err = applyCloaking(context.Background(), cfg, auth, payload, "sk-ant-oat-test", true, false)
	if err != nil || cloaked || string(out) != string(payload) {
		t.Fatalf("native request was modified: %v %v", cloaked, err)
	}
	files, _ := filepath.Glob(filepath.Join(cfg.AuthDir, ".claude-session-dates", "*.date"))
	if err := os.WriteFile(files[0], []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := applyCloaking(context.Background(), cfg, auth, payload, "sk-ant-oat-test", false, false); err == nil {
		t.Fatal("cloaking must not silently change the date on storage failure")
	}
}
