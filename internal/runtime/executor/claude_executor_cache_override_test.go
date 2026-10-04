package executor

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/config"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v8/sdk/translator"
	"github.com/tidwall/gjson"
)

func TestClaudeExecutor_FiveMinutePromptCacheOverride(t *testing.T) {
	for _, stream := range []bool{false, true} {
		for _, callerTTL := range []bool{false, true} {
			name := "execute"
			if stream {
				name = "stream"
			}
			if callerTTL {
				name += "/caller-1h"
			} else {
				name += "/injected-markers"
			}
			t.Run(name, func(t *testing.T) {
				var seenBody []byte
				var seenHeaders http.Header
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					seenBody, _ = io.ReadAll(r.Body)
					seenHeaders = r.Header.Clone()
					if stream {
						w.Header().Set("Content-Type", "text/event-stream")
						_, _ = w.Write([]byte("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-opus-4-6\"}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
					} else {
						w.Header().Set("Content-Type", "application/json")
						_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","model":"claude-opus-4-6","role":"assistant","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1}}`))
					}
				}))
				defer server.Close()

				cfg := &config.Config{Claude: config.ClaudeConfig{PromptCacheTTL: "5m"}}
				executor := NewClaudeExecutor(cfg)
				auth := &cliproxyauth.Auth{
					ID: "cache-5m-override",
					Attributes: map[string]string{
						"api_key":    "sk-ant-oat-cache-5m-override",
						"base_url":   server.URL,
						"cloak_mode": "always",
					},
					Metadata: claudeOAuthTestMetadata(),
				}
				marker := ""
				if callerTTL {
					marker = `,"cache_control":{"type":"ephemeral","ttl":"1h"}`
				}
				payload := []byte(`{"model":"claude-opus-4-6","system":[{"type":"text","text":"test system"` + marker + `}],"messages":[{"role":"user","content":[{"type":"text","text":"x"` + marker + `}]}]}`)
				req := cliproxyexecutor.Request{Model: "claude-opus-4-6", Payload: payload}
				opts := cliproxyexecutor.Options{SourceFormat: sdktranslator.FormatClaude}
				if stream {
					result, err := executor.ExecuteStream(context.Background(), auth, req, opts)
					if err != nil {
						t.Fatal(err)
					}
					for chunk := range result.Chunks {
						if chunk.Err != nil {
							t.Fatal(chunk.Err)
						}
					}
				} else if _, err := executor.Execute(context.Background(), auth, req, opts); err != nil {
					t.Fatal(err)
				}

				markers := 0
				forEachClaudeCacheControlBlock(seenBody, func(path string, block gjson.Result) {
					cc := block.Get("cache_control")
					if !cc.IsObject() {
						return
					}
					markers++
					if cc.Get("type").String() != "ephemeral" || cc.Get("ttl").Exists() {
						t.Errorf("%s must retain ephemeral caching without ttl (5m default): %s", path, cc.Raw)
					}
				})
				if markers == 0 {
					t.Fatal("override must not disable prompt caching")
				}
				if strings.Contains(seenHeaders.Get("Anthropic-Beta"), claudeExtendedCacheTTLBeta) {
					t.Fatal("5m requests must not advertise extended-cache-ttl")
				}
			})
		}
	}
}
