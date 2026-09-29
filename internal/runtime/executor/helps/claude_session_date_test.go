package helps

import (
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestClaudeSessionDateMidnightAndPersistence(t *testing.T) {
	dir := t.TempDir()
	zone := time.FixedZone("AEST", 10*60*60)
	before := time.Date(2026, 9, 29, 23, 59, 59, 0, zone)
	after := before.Add(2 * time.Second)
	first := []byte(`{"system":"stable instructions","messages":[{"role":"user","content":"unique session root"}]}`)
	continuation := []byte(`{"system":"stable instructions","messages":[{"role":"user","content":[{"type":"text","text":"unique session root","cache_control":{"type":"ephemeral"}}]},{"role":"assistant","content":[{"type":"text","text":"answer"}]},{"role":"user","content":"next turn"}]}`)
	for i, input := range []struct {
		payload []byte
		now     time.Time
	}{
		{first, before},
		{continuation, after},
		{continuation, after.AddDate(0, 0, 30)},
	} {
		date, err := ClaudeSessionDate(dir, nil, input.payload, input.now)
		if err != nil || date.Format(time.DateOnly) != "2026-09-29" {
			t.Fatalf("call %d: date=%v error=%v", i, date, err)
		}
	}
	files, err := filepath.Glob(filepath.Join(dir, ".claude-session-dates", "*"))
	if err != nil || len(files) != 1 {
		t.Fatalf("want exactly one persisted date, got %v, %v", files, err)
	}
	data, err := os.ReadFile(files[0])
	if err != nil || string(data) != "2026-09-29" {
		t.Fatalf("persisted date = %q, %v", data, err)
	}
	info, err := os.Stat(files[0])
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("date file must have private permissions: %v, %v", info, err)
	}
	// No process-global cache exists: every call above reads this persisted record.
	// Removing it models an explicit operator reset rather than TTL eviction.
	if err := os.Remove(files[0]); err != nil {
		t.Fatal(err)
	}
	date, err := ClaudeSessionDate(dir, nil, continuation, after)
	if err != nil || date.Format(time.DateOnly) != "2026-09-30" {
		t.Fatalf("reset date=%v error=%v", date, err)
	}
}

func TestClaudeSessionDateExplicitSessionsAreIsolated(t *testing.T) {
	dir := t.TempDir()
	before := time.Date(2026, 9, 29, 23, 59, 59, 0, time.UTC)
	payload := []byte(`{"messages":[{"role":"user","content":"hello"}]}`)
	for _, item := range []struct {
		id, want string
		now      time.Time
	}{
		{"one", "2026-09-29", before},
		{"two", "2026-09-30", before.Add(time.Minute)},
		{"one", "2026-09-29", before.Add(48 * time.Hour)},
	} {
		headers := http.Header{"X-Session-Id": []string{item.id}}
		date, err := ClaudeSessionDate(dir, headers, payload, item.now)
		if err != nil || date.Format(time.DateOnly) != item.want {
			t.Fatalf("session %s: date=%v error=%v", item.id, date, err)
		}
	}
}

func TestClaudeSessionDateConcurrentFirstRequests(t *testing.T) {
	dir := t.TempDir()
	before := time.Date(2026, 9, 29, 23, 59, 59, 0, time.UTC)
	payload := []byte(`{"messages":[{"role":"user","content":"concurrent"}]}`)
	const callers = 16
	results := make(chan string, callers)
	var workers sync.WaitGroup
	for i := range callers {
		workers.Go(func() {
			now := before.Add(time.Duration(i%2) * time.Minute)
			date, err := ClaudeSessionDate(dir, nil, payload, now)
			if err != nil {
				t.Error(err)
				return
			}
			results <- date.Format(time.DateOnly)
		})
	}
	workers.Wait()
	close(results)
	want := ""
	for date := range results {
		if want == "" {
			want = date
		}
		if date != want {
			t.Fatalf("concurrent callers saw different dates: %s and %s", want, date)
		}
	}
}

func TestClaudeSessionDateCorruptRecordFailsClosed(t *testing.T) {
	dir := t.TempDir()
	payload := []byte(`{"messages":[{"role":"user","content":"corruption"}]}`)
	now := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	if _, err := ClaudeSessionDate(dir, nil, payload, now); err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob(filepath.Join(dir, ".claude-session-dates", "*.date"))
	if err := os.WriteFile(files[0], []byte("broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ClaudeSessionDate(dir, nil, payload, now.AddDate(0, 0, 1)); err == nil {
		t.Fatal("corrupt record must not silently rewrite the prefix")
	}
}

func TestClaudeSessionDateNoConversationOrStorage(t *testing.T) {
	now := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	for _, item := range []struct{ dir, payload string }{
		{"", `{"messages":[{"role":"user","content":"hello"}]}`},
		{t.TempDir(), `{"messages":[]}`},
	} {
		date, err := ClaudeSessionDate(item.dir, nil, []byte(item.payload), now)
		if err != nil || !date.Equal(now) {
			t.Fatalf("date=%v error=%v", date, err)
		}
	}
}
