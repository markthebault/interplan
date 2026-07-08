---
name: interplan
description: Use Interplan for browser-reviewable local HTML artifacts when an agent needs user feedback on plans, UI mockups, architecture docs, reports, or other rich HTML deliverables. Trigger when asked to open an artifact for review, collect browser annotations, poll for user feedback, watch an Interplan session, or apply targeted comments returned by `interplan poll`.
---

# Interplan Review

Use Interplan to put a local HTML artifact in front of the user, let them annotate elements in a browser, then receive structured feedback through the CLI.

## Core Loop

1. Build or update a complete `.html` artifact.
2. Run Interplan:

```sh
interplan /absolute/path/to/artifact.html
```

If `interplan` is not on `PATH`, look for a local binary first:

```sh
./bin/interplan /absolute/path/to/artifact.html
```

3. Tell the user the review URL from stdout if the browser does not open automatically.
4. **Begin polling immediately**. Do not wait for user confirmation. The `poll` command will block until feedback arrives:

```sh
interplan poll /absolute/path/to/artifact.html
```

   The poll will return when:
   - The user submits feedback ("Send" or "Send & End")
   - The session is ended
   - A timeout is reached (if `--timeout-ms` was specified)

5. Apply feedback to the source artifact or surrounding project files.
6. If the session is still open (`session_ended: false`), poll again. Stop when poll returns `session_ended: true` or `status: ended`.
7. Continue the loop until the user ends the session.

### Example Full Flow

```sh
# Open the session (browser opens automatically)
interplan /tmp/plan.html

# Start polling immediately - this blocks until feedback
while true; do
  # Poll blocks here waiting for user input
  interplan poll /tmp/plan.html
  
  # When poll returns, check if session ended
  # (in practice, parse the output to check session_ended)
  # If ended, break; otherwise apply feedback and continue
done
```

Do not ask the user to "let you know" when they're done. The polling mechanism handles synchronization automatically.

## Watching For Feedback

Interplan feedback arrives through polling, not by reading the HTML file. Do not grep the artifact to find annotations.

For active review, keep a poll running:

```sh
interplan poll /absolute/path/to/artifact.html
```

For a bounded watch loop:

```sh
while true; do
  interplan poll /absolute/path/to/artifact.html --timeout-ms 30000
done
```

Break the loop when the returned session says `session_ended: true`, `status: ended`, or `ended_by: user`.

If using the repo manual demo, get the port from the generated port file:

```sh
INTERPLAN_PORT="$(cat /tmp/interplan-complex-demo.port)" ./bin/interplan poll /tmp/interplan-complex-demo.html --timeout-ms 1000
```

## Applying Annotations

Default output is TOON. Parse it as structured Interplan output, not prose.

Important prompt fields:

- `tag: message`: general feedback from the side-panel textarea.
- `tag: element`: targeted UI annotation.
- `tag: text`: targeted selected-text annotation.
- `prompt`: the user's requested change.
- `selector`: CSS selector for the clicked element or the element containing the selected text.
- `text`: nearby visible text for element annotations, or the selected quote for text annotations.
- `target`: structured metadata such as kind, tag, selector, text, and text-selection context.

For `tag: element`, use both `selector` and `text`. Selectors can become stale after edits, so use the text as fallback context.

For `tag: text`, treat `text` as the selected quote and `target.context` as nearby containing text. Use the selector to locate the containing element, then use the quote/context pair to find the exact wording if the selector drifts.

Example:

```toon
prompts[1]{tag,prompt,text,selector}:
  element,"Make this red","UI Browser review...","div[data-testid=\"browser-review-card\"]"
  text,"Shorten this sentence","selected quote","p:nth-of-type(2)"
```

Apply this as a targeted change to the element or selected quote represented by the selector/text, then explain what changed.

## Updating The Artifact During Review

✅ **Live reload is now automatic!**

When the agent edits the HTML file:
1. The server detects the change (polling every 500ms)
2. Sends a reload event to the browser via SSE
3. Browser automatically reloads the iframe
4. Scroll position is preserved

The agent should:
1. Apply changes to the HTML file
2. Announce: "I've updated the file. The browser will reload automatically."
3. Continue polling for more feedback

Example agent message:
```
I've updated the title and changed the card background to red.
The browser should reload automatically to show the changes.
I'm polling for your next round of feedback...
```

The user does not need to click Reload manually. The changes appear automatically within ~500ms.

## Manual Test Helpers

For **manual testing by a human** (not normal agent workflow), the test scripts are split into two commands:

```sh
just test-manual-complex-demo  # Opens browser, then STOPS
```

This opens the browser and **intentionally does NOT poll** so a human can manually test the browser UI.

After manually submitting feedback in the browser:

```sh
just test-manual-poll-complex-demo  # Polls once with timeout
```

If `just` is unavailable:

```sh
./scripts/manual-complex-demo.sh
# MANUAL TESTING: Script stops here for human interaction
INTERPLAN_PORT="$(cat /tmp/interplan-complex-demo.port)" ./bin/interplan poll /tmp/interplan-complex-demo.html --timeout-ms 1000
```

### Agent Workflow vs Manual Testing

**For agents in production** (normal use):
- Open the session
- **Immediately start polling** (blocking, no timeout)
- Automatically process feedback when it arrives
- Continue polling until session ends
- Do NOT ask user to confirm they're ready

**For manual testing** (developers testing the tool itself):
- Use the split test commands above
- Open the browser, manually interact with UI
- Run poll command separately to verify feedback was captured

### Correct Agent Behavior Example

```sh
# Agent opens session
interplan /tmp/plan.html
# Output: session opened at http://...

# Agent immediately starts polling (BLOCKS here)
interplan poll /tmp/plan.html
# Agent announces: "I'm now polling for your feedback. Take your time reviewing."
# ... waits silently until user submits feedback ...
# Poll returns with feedback

# Agent processes and applies feedback
# If session still open, poll again
```

**Incorrect:** Asking "let me know when you're done" and waiting for user confirmation before polling.

## Safety And State

Annotations are stored in Interplan's local state file until delivered by `poll`. A successful poll consumes pending prompts so they are not delivered repeatedly.

On macOS, local state is:

```text
~/Library/Application Support/interplan/state.json
```

Read the state file only for debugging. Use `interplan poll` for normal agent workflows.

## Port Conflicts

The default port is `37917` (configurable via `INTERPLAN_PORT`). If you get:

```
server did not become healthy on port 37917
```

Another interplan server is already running on that port. Options:

1. **Use a different port**: `INTERPLAN_PORT=37918 ./bin/interplan /path/to/file.html`
2. **Stop old servers**: `pkill interplan`
3. **Use the test script**: `./scripts/manual-complex-demo.sh` (auto-finds free port)

Check running servers:

```bash
lsof -nP -iTCP:37917 -sTCP:LISTEN
```
