# interplan

AI agent interactive planning tool: open an HTML artifact for review, collect browser feedback, and return structured agent-readable output.

This implementation follows `lavish_axi_clone_spec.html` as authoritative.

## Current build slice

Implemented first:

- Section 16.1 Phase 1: Go module, command normalization, canonical path/session key helpers, JSON state store with mutex and atomic writes, TOON output with JSON fallback, and unit coverage.
- Section 21 minimal protocol: open a session, accept browser feedback at `POST /api/{key}/prompts`, and return final feedback through poll.
- Managed `open`: starts/reuses the local server and opens the review URL unless `--no-open` or `INTERPLAN_NO_OPEN=1` is set.
- Minimal browser review shell at `/session/{key}` with iframe artifact rendering, comment send, and Send & End.
- Server-backed `poll` with `--timeout-ms` support.
- Basic MVP APIs: `/api/sessions`, `/api/poll`, `/api/end`, `/api/{key}/prompts`, `/api/{key}/end`, `/api/{key}/layout-warnings`, and `/api/agent-reply`.
- `.html` and `.htm` bare paths are accepted.

Intentionally not implemented yet:

- Export.
- Share/publish.
- Agent hook installation.
- Full annotation UI, Mermaid node picking, and layout gate detection.

Those are later parity features and should remain behind the proven open/post/poll loop.

## Live Reload

✅ **Automatic live reload is now implemented!**

When you edit the HTML artifact file:
- The server detects the change (checks every 500ms)
- Sends a reload event via Server-Sent Events (SSE)
- Browser automatically reloads the iframe
- Scroll position is preserved

No manual "Reload" button click needed. The agent can edit the file and the user will see changes automatically.

## Intentional choices

- Distribution target remains a native Go binary with embedded assets in later phases.
- State uses OS-appropriate Interplan paths:
  - macOS: `~/Library/Application Support/interplan/state.json`
  - Linux: `${XDG_STATE_HOME:-~/.local/state}/interplan/state.json`
  - Windows: `%LocalAppData%\interplan\state.json`
- Default command output uses TOON. JSON is available with `--json`.
- TOON is backed by the maintained `github.com/toon-format/toon-go` encoder/decoder. Tests validate default output by decoding it in strict mode.

## Minimal flow

```sh
interplan /tmp/doc.html
curl -X POST http://127.0.0.1:37917/api/<session-key>/prompts \
  -H 'content-type: application/json' \
  -d '{"prompts":[{"tag":"message","prompt":"Change the title."}],"domSnapshot":"h1 Draft","endSession":true}'
interplan poll /tmp/doc.html
```

The Section 21 proof is captured in `internal/server/section21_test.go`.

## Releases

Release automation uses GitHub Actions, Conventional Commits, and Release Please. See [`doc/RELEASES.md`](doc/RELEASES.md) for the full process.
