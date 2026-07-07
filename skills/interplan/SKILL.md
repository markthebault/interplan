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
4. Poll for feedback:

```sh
interplan poll /absolute/path/to/artifact.html
```

Use `--timeout-ms` only when you intentionally want a bounded poll during manual tests or automation.

5. Apply feedback to the source artifact or surrounding project files.
6. If the session is still open, poll again. Stop when poll returns `session_ended: true` or `status: ended`.

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
- `prompt`: the user's requested change.
- `selector`: CSS selector for the clicked element.
- `text`: nearby visible text from the clicked element.
- `target`: structured metadata such as element kind, tag, selector, and text.

For `tag: element`, use both `selector` and `text`. Selectors can become stale after edits, so use the text as fallback context.

Example:

```toon
prompts[1]{tag,prompt,text,selector}:
  element,"Make this red","UI Browser review...","div[data-testid=\"browser-review-card\"]"
```

Apply this as a targeted change to the element represented by the selector/text, then explain what changed.

## Updating The Artifact During Review

After applying feedback to the HTML file, tell the user to click **Reload** in the Interplan browser UI if they need to inspect the updated artifact.

Do not claim automatic live reload is available unless the current repository implements it. At present, use the browser Reload control or reopen the same artifact with Interplan.

## Manual Test Helpers

In this repository, use:

```sh
just test-manual-complex-demo
just test-manual-poll-complex-demo
```

If `just` is unavailable:

```sh
./scripts/manual-complex-demo.sh
INTERPLAN_PORT="$(cat /tmp/interplan-complex-demo.port)" ./bin/interplan poll /tmp/interplan-complex-demo.html --timeout-ms 1000
```

## Safety And State

Annotations are stored in Interplan's local state file until delivered by `poll`. A successful poll consumes pending prompts so they are not delivered repeatedly.

On macOS, local state is:

```text
~/Library/Application Support/interplan/state.json
```

Read the state file only for debugging. Use `interplan poll` for normal agent workflows.
