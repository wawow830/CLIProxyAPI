package helps

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	cliproxysession "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/session"
	sdktranslator "github.com/router-for-me/CLIProxyAPI/v8/sdk/translator"
	log "github.com/sirupsen/logrus"
)

// ClaudeSessionDate pins the generated date reminder to the first request seen for
// a conversation. Rewriting the first user message at midnight invalidates both
// prompt caches and prefix-bound thinking signatures. No later date is injected.
//
// Dates are stored separately from credentials, without prompts or session IDs.
// There is deliberately no TTL: an idle session must retain its prefix after a
// restart. Anonymous callers with identical conversation roots share a date.
func ClaudeSessionDate(authDir string, headers http.Header, payload []byte, now time.Time) (time.Time, error) {
	if strings.TrimSpace(authDir) == "" {
		return now, nil
	}
	// Do not use the runtime LCP/affinity ID: it can change after a restart or as
	// an initially anonymous conversation acquires an assistant turn.
	identity := ""
	if info, ok := cliproxysession.ExtractSessionInfo(headers, payload, nil); ok && info.ClientType != "lcp" {
		identity = info.SessionID
	}
	if identity == "" {
		identity = cliproxysession.DeriveID(sdktranslator.FormatClaude, payload, "")
	}
	if identity == "" {
		return now, nil
	}
	dir := filepath.Join(authDir, ".claude-session-dates")
	path := filepath.Join(dir, fmt.Sprintf("%x.date", sha256.Sum256([]byte(identity))))
	if date, errRead := readClaudeSessionDate(path, now.Location()); !errors.Is(errRead, os.ErrNotExist) {
		return date, errRead
	}
	if errMkdir := os.MkdirAll(dir, 0o700); errMkdir != nil {
		return time.Time{}, fmt.Errorf("create Claude session date directory: %w", errMkdir)
	}
	file, errCreate := os.CreateTemp(dir, ".date-*")
	if errCreate != nil {
		return time.Time{}, fmt.Errorf("create Claude session date: %w", errCreate)
	}
	defer func() {
		if errRemove := os.Remove(file.Name()); errRemove != nil && !errors.Is(errRemove, os.ErrNotExist) {
			log.Warnf("remove temporary Claude session date: %v", errRemove)
		}
	}()
	_, errWrite := file.WriteString(now.Format(time.DateOnly))
	if errWrite == nil {
		errWrite = file.Sync()
	}
	errClose := file.Close()
	if errWrite != nil {
		return time.Time{}, fmt.Errorf("write Claude session date: %w", errWrite)
	}
	if errClose != nil {
		return time.Time{}, fmt.Errorf("close Claude session date: %w", errClose)
	}
	// Publish a complete file without overwriting a concurrent winner, including
	// another process using the same auth directory. Readers never see partial data.
	if errLink := os.Link(file.Name(), path); errLink != nil && !errors.Is(errLink, os.ErrExist) {
		return time.Time{}, fmt.Errorf("publish Claude session date: %w", errLink)
	}
	return readClaudeSessionDate(path, now.Location())
}

func readClaudeSessionDate(path string, location *time.Location) (time.Time, error) {
	data, errRead := os.ReadFile(path)
	if errRead != nil {
		return time.Time{}, fmt.Errorf("read Claude session date: %w", errRead)
	}
	date, errParse := time.ParseInLocation(time.DateOnly, string(data), location)
	if errParse != nil {
		// Fail rather than silently replace a corrupt record with today's date and
		// invalidate the entire historical prefix again.
		return time.Time{}, fmt.Errorf("invalid persisted Claude session date: %w", errParse)
	}
	return date, nil
}
