# Local Claude session-date patch (rebased on upstream 6d57ac90)

This patch keeps CPA's generated `currentDate` reminder stable instead of rewriting
its date in the first user message at every local midnight. Rewriting that prefix
invalidates Anthropic prompt caches and prefix-bound thinking blocks.

## Behavior

- Pin the local calendar date when this patched proxy first sees a conversation.
- Prefer an explicit client session identity; otherwise derive an identity from
  the caller's leading instructions and first user input, independent of later
  assistant/tool turns and in-memory affinity state.
- Anonymous conversations with identical roots share a date. Clients that need
  independent dates should send an explicit session ID.
- Persist only `YYYY-MM-DD`, in private, hashed `.date` files beneath
  `<auth-dir>/.claude-session-dates/`. No prompts, tokens, or raw session IDs are
  written. Atomic publication prevents concurrent requests/processes from
  selecting different dates.
- Do not expire dates or inject later date updates. Use a tool such as `date` when
  actual current time matters. Deleting a date record resets that conversation's
  prefix on its next request, so do not clear this directory as a routine cache.
- Fail the request on unreadable/corrupt date state rather than silently changing
  history. Native Claude Code passthrough requests remain untouched.
- Embedded callers without a configured auth directory retain upstream behavior.

## Migration

Existing conversations pin the date of their first request after deployment, not
necessarily their original creation date. Already-invalidated thinking blocks are
not repaired. Starting fresh or compacting once may be useful for affected sessions.

Keep the date directory when upgrading or rolling back; an unpatched binary simply
ignores it. Future upstream updates must retain this patch (or implement equivalent
behavior) to avoid resuming daily prefix rewrites.

## Validation

```sh
GOMAXPROCS=2 go test -p 1 ./internal/runtime/executor ./internal/runtime/executor/helps
GOMAXPROCS=2 go test -race -p 1 ./internal/runtime/executor/helps -run '^TestClaudeSessionDate'
go build -o cli-proxy-api ./cmd/server
```

Tests cover simulated midnight, persisted state, calendar-day rollover in AEST,
explicit-session isolation, anonymous multi-turn identity, cache-control markers,
concurrent first requests, corrupt state, native passthrough, and the actual
cloaking call site's use of a pre-existing date.

## Interaction with upstream pinning

Upstream pins `currentDate` in memory per continuity key (1h idle TTL, lost on
restart). This patch layers on top: when `auth-dir` is configured, the persisted
record wins, and upstream's in-memory pinned date seeds a new record. Without
`auth-dir` or a resolvable identity, upstream's behavior applies unchanged.
