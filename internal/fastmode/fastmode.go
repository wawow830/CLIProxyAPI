// Package fastmode implements the "<model>-fast" naming convention for Codex.
//
// A request for "<model>-fast" is served by "<model>" with the OpenAI priority
// service tier ("Fast mode"). Thinking suffixes are preserved, so
// "gpt-6-sol-fast(high)" resolves to "gpt-6-sol(high)".
package fastmode

import (
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v8/internal/thinking"
)

const (
	// Suffix is appended to a base model ID to request Fast mode.
	Suffix = "-fast"
	// ServiceTier is the service_tier value sent upstream for Fast mode.
	ServiceTier = "priority"
	// Channel is the only OAuth channel that supports Fast mode.
	Channel = "codex"
)

// Trim returns the base model for a "-fast" request, keeping any thinking suffix.
// ok is false when model is not a fast variant.
func Trim(model string) (base string, ok bool) {
	model = strings.TrimSpace(model)
	parsed := thinking.ParseSuffix(model)
	name := strings.TrimSpace(parsed.ModelName)
	if name == "" {
		name = model
	}
	if len(name) <= len(Suffix) || !strings.EqualFold(name[len(name)-len(Suffix):], Suffix) {
		return "", false
	}
	base = name[:len(name)-len(Suffix)]
	if parsed.HasSuffix && parsed.RawSuffix != "" {
		base += "(" + parsed.RawSuffix + ")"
	}
	return base, true
}

// IsFast reports whether model is a "-fast" variant.
func IsFast(model string) bool {
	_, ok := Trim(model)
	return ok
}

// Eligible reports whether a catalog model ID gets an automatic "-fast" variant.
// Only text GPT models qualify; image models and already-fast IDs do not.
func Eligible(id string) bool {
	id = strings.ToLower(strings.TrimSpace(id))
	if !strings.HasPrefix(id, "gpt-") || strings.Contains(id, "image") {
		return false
	}
	return !IsFast(id)
}
