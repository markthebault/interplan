# Interplan Remaining Features Specification

This file defines the remaining Interplan features to implement. It is an implementation contract for coding agents. Follow this file as authoritative for the remaining work.

## 1. Implementation Order

Implement features in this exact order:

1. Agent reply display in the browser chrome.
2. Rich DOM snapshot generation.
3. Structured input collection.
4. Mermaid node annotation.
5. Layout warning detection and open-time layout gate.
6. Server lifecycle management: `stop`, ownership checks, idle shutdown.
7. Playbooks and design guidance commands.
8. Export command.
9. Share command.
10. Agent hook installer.

Do not implement export, share, or hooks before items 1 through 7 are complete and tested.

## 2. Existing Baseline

The current implementation already provides:

- Go CLI entrypoint.
- Session store and deterministic session keys.
- `open`, bare HTML path normalization, `poll`, `end`, and foreground `server` commands.
- TOON output by default and JSON output with `--json`.
- Browser chrome with iframe, basic element annotation, general comments, Send, Send & End, End Session, and automatic live reload.
- Local HTTP APIs for sessions, polling, prompts, end, agent reply, layout warning ingestion, artifact serving, and SSE reload events.

New work must preserve this behavior.

## 3. Agent Reply Display

### 3.1 CLI Behavior

`interplan poll <html-file> --agent-reply <message>` must append an agent chat message to the session before entering the poll wait.

The CLI must send the reply through the existing `POST /api/agent-reply` endpoint before calling the poll endpoint.

If `--agent-reply` is used with an empty string, return an error:

```text
--agent-reply requires a non-empty message
```

### 3.2 Session Model

Chat entries must be stored in `session.Session.Chat` using this shape:

```json
{
  "role": "agent",
  "message": "Updated the heading and card color.",
  "created_at": "2026-07-07T12:00:00Z"
}
```

Browser-originated messages must use `role: "user"` when displayed or persisted as chat messages. Prompt queue items remain prompts and must not be converted into chat entries unless they are general side-panel messages sent without a target.

### 3.3 Browser UI

The right-side panel must contain a chat section above the prompt textarea.

Chat rendering rules:

- Agent messages are right-aligned.
- User messages are left-aligned.
- Each message shows role label and text.
- New messages appear without a full browser page reload.
- Messages survive browser reload because they are loaded from session state.

### 3.4 HTTP API

Add:

```http
GET /api/{key}/chat
```

Response:

```json
{
  "chat": [
    {"role":"agent","message":"Updated.","created_at":"2026-07-07T12:00:00Z"}
  ]
}
```

The browser must poll this endpoint every 1000ms while the session page is open.

### 3.5 Tests

Add Go tests proving:

- `--agent-reply` posts before polling.
- Chat entries are persisted.
- `GET /api/{key}/chat` returns persisted entries.
- Session chrome includes the chat container and polling code.

## 4. Rich DOM Snapshot Generation

### 4.1 Browser Behavior

When the user clicks Send or Send & End, the browser must include a compact DOM snapshot in `domSnapshot`.

The snapshot must include:

- Page title.
- Visible headings `h1` through `h4`.
- Landmark elements: `header`, `nav`, `main`, `aside`, `footer`, `section`, `article`.
- Elements with `data-review-id`, `data-testid`, `data-id`, `aria-label`, `role`, or `id`.
- Text content collapsed to single spaces.
- CSS selector for each listed element.

The snapshot must exclude:

- `<script>` and `<style>` content.
- Hidden elements where computed `display` is `none`, `visibility` is `hidden`, or opacity is `0`.
- Text from elements outside the viewport by more than 2000px vertically.

### 4.2 Snapshot Format

Use plain text lines. Maximum total length is 12000 characters. Truncate after a complete line and append:

```text
... truncated
```

Line format:

```text
<title> Interplan rollout proposal
h1 header[data-review-id="hero"] Interplan rollout proposal
section section[data-review-id="overview"] Initial technical approach Browser review Long-poll API
button button[data-interplan-action="confirm-priority"] Confirm priority
```

### 4.3 Tests

Add browser-level JavaScript unit coverage or Go string fixture coverage for the snapshot function. Tests must verify hidden elements are excluded and selector-bearing elements are included.

## 5. Structured Input Collection

### 5.1 `window.interplan.queuePrompt`

The injected SDK must expose:

```js
window.interplan.queuePrompt(prompt)
```

The function must normalize, validate, deduplicate, and enqueue structured prompts in the browser queue. It must not immediately send prompts to the server unless the caller passes `send: true`.

Accepted prompt fields:

```json
{
  "tag": "input",
  "queueKey": "pricing-plan-choice",
  "prompt": "User chose the Pro plan.",
  "label": "Pricing plan",
  "value": {"plan":"pro","seats":8},
  "send": false
}
```

Validation rules:

- `tag` is required.
- `prompt` is required.
- `tag` must be one of `message`, `element`, `input`, `choice`, `action`, `question`, `mermaid-node`, `layout-warning`.
- `queueKey`, when present, must be a non-empty string.
- `value`, when present, may be any JSON value.

### 5.2 Native Controls

The SDK must capture structured input from native controls when the user clicks Send or Send & End.

Controls to scan:

- `input`
- `textarea`
- `select`
- radio groups
- checkbox groups

Prompt generation rules:

- Text-like `input`, `textarea`, and `select` generate `tag: "input"`.
- Radio and checkbox groups generate `tag: "choice"`.
- Label is derived from, in order: associated `<label>`, `aria-label`, `placeholder`, `name`, `id`.
- Queue key is derived from, in order: `data-interplan-queue-key`, `data-interplan-question`, `name`, `id`.
- Empty controls are skipped.
- Password inputs are skipped.
- File inputs are skipped.

### 5.3 Action Attributes

Clicking an element with `data-interplan-action` must queue an action prompt.

Required attributes:

```html
<button data-interplan-action="confirm-priority" data-interplan-prompt="User confirmed priority.">Confirm</button>
```

Queued prompt shape:

```json
{
  "tag": "action",
  "queueKey": "action:confirm-priority",
  "action": "confirm-priority",
  "label": "Confirm",
  "prompt": "User confirmed priority."
}
```

If `data-interplan-prompt` is missing, use the element text as the prompt.

### 5.4 Reversible Questions

Elements with `data-interplan-question` must replace earlier undelivered answers with the same queue key.

Queue key derivation:

1. `data-interplan-queue-key`
2. `data-interplan-question`
3. `name`
4. `id`

Only the latest value for a queue key must be sent.

### 5.5 Tests

Add tests for:

- Text input capture.
- Select capture.
- Radio group capture.
- Checkbox group capture.
- `data-interplan-action` click capture.
- Queue key deduplication.
- Password and file inputs skipped.

## 6. Mermaid Node Annotation

### 6.1 Supported Markup

Support Mermaid diagrams rendered inside elements matching one of:

```css
.mermaid
[data-interplan-diagram]
svg[aria-roledescription="flowchart-v2"]
```

Authors must identify diagrams with:

```html
<div class="mermaid" data-interplan-diagram="checkout-flow">
```

If no `data-interplan-diagram` exists, derive a diagram id as `mermaid-<index>` based on document order.

### 6.2 Annotation Behavior

In annotate mode, clicking a Mermaid SVG node must create `tag: "mermaid-node"` instead of generic `tag: "element"`.

Prompt target shape:

```json
{
  "tag": "mermaid-node",
  "prompt": "This transition should mention retry behavior.",
  "target": {
    "kind": "mermaid-node",
    "diagram_id": "checkout-flow",
    "node_id": "PaymentFailed",
    "label": "Payment failed",
    "selector": "[data-interplan-diagram=\"checkout-flow\"]"
  }
}
```

### 6.3 Node Identification Rules

When a clicked element is inside Mermaid-rendered SVG:

1. Find the nearest ancestor SVG group with an `id`, `data-id`, or class containing `node`.
2. Extract `node_id` from `data-id` first, then `id`, then normalized visible label text.
3. Extract `label` from `span`, `text`, or `foreignObject` content inside the node group.
4. Use the diagram container selector as `target.selector`.

### 6.4 Pan And Zoom

Explore mode must allow normal Mermaid pan and zoom behavior when the diagram provides it.

Annotate mode must prevent pan and zoom gestures on Mermaid diagrams and use clicks for annotation.

### 6.5 Tests

Add a browser fixture containing a Mermaid-rendered SVG and prove that a node click queues `tag: "mermaid-node"` with `diagram_id`, `node_id`, `label`, and selector.

## 7. Layout Warning Detection And Open-Time Gate

### 7.1 Required Checks

Run layout checks inside the artifact iframe after:

1. The iframe `load` event.
2. `document.fonts.ready` resolves, when available.
3. One `requestAnimationFrame` after fonts are ready.

Checks must detect:

- Horizontal document overflow.
- Element overflow outside viewport or nearest scroll container.
- Clipped text where scroll dimensions exceed client dimensions on non-scrollable elements.
- Overlapping visible text boxes.
- Text spilling outside parent container.
- Unreadably small content caused by iframe zoom or scale errors.

### 7.2 Exclusions

Exclude elements where:

- `display: none`.
- `visibility: hidden`.
- `opacity: 0`.
- The element or nearest relevant container has intentional `overflow-x` or `overflow-y` set to `auto` or `scroll`.
- The element has `data-interplan-ignore-layout`.

### 7.3 Warning Shape

Every warning must use this shape:

```json
{
  "key": "clipped-text:.summary-card h2:Text appears clipped vertically.",
  "kind": "clipped-text",
  "severity": "error",
  "selector": ".summary-card h2",
  "message": "Text appears clipped vertically.",
  "overflowPx": 92,
  "viewportWidth": 1440,
  "viewport": {"width": 1440, "height": 900},
  "box": {"x": 240, "y": 180, "width": 320, "height": 32},
  "persistent": false
}
```

Allowed `kind` values:

- `document-overflow`
- `element-overflow`
- `clipped-text`
- `overlapping-text`
- `spilling-text`
- `small-content`

Allowed `severity` values:

- `error`
- `warning`
- `note`

### 7.4 Deduplication

The browser must compute `key` as:

```text
kind + ":" + selector + ":" + message
```

The server must store delivered warning keys in `delivered_layout_warning_keys` after poll returns warnings to the agent.

If a warning key is posted again after being delivered once, the server must set `persistent: true` before returning it in poll output.

### 7.5 Open-Time Gate

On initial session page load, show a gate overlay over the artifact iframe.

Gate behavior:

1. Overlay text: `Checking layout before review...`
2. Run layout audit.
3. If no `severity: "error"` warnings exist, remove overlay.
4. If one or more error warnings exist:
   - Keep overlay visible.
   - Post warnings to `POST /api/{key}/layout-warnings`.
   - Change overlay text to `Layout issues were found and sent to the agent.`
   - Show a `Show anyway` button.
5. If audit does not complete within 5000ms:
   - Remove overlay.
   - Show a non-blocking banner: `Layout audit timed out.`

Add CLI flag:

```sh
interplan open <file> --no-gate
```

When `--no-gate` is set, session URL must include `?gate=false`, and browser chrome must skip the gate.

### 7.6 Persistent Banner

After the gate reveals, if layout warnings exist, show a banner above the iframe:

```text
Layout warnings were sent to the agent.
```

The banner remains until the iframe is reloaded and a new audit returns zero warnings.

### 7.7 Tests

Add tests proving:

- Horizontal overflow warning is posted.
- Clipped text warning is posted.
- Intentional scroll containers are ignored.
- Warning deduplication marks repeated delivered warnings as persistent.
- Gate masks artifact when error warnings exist.
- `?gate=false` disables the gate.

## 8. Server Lifecycle Management

### 8.1 Server Metadata File

When the server starts, write a metadata file next to the state file:

```text
<state-dir>/server.json
```

Shape:

```json
{
  "pid": 12345,
  "port": 37917,
  "token": "64 random hex characters",
  "started_at": "2026-07-07T12:00:00Z",
  "binary": "/absolute/path/to/interplan"
}
```

The token must be generated with `crypto/rand`.

### 8.2 Health Response

`GET /health` must return:

```json
{
  "ok": true,
  "name": "interplan",
  "protocol_version": 3,
  "pid": 12345,
  "port": 37917
}
```

### 8.3 Stop Command

Implement:

```sh
interplan stop
```

Behavior:

1. Read `server.json`.
2. Confirm the process is alive.
3. Confirm `/health` returns `name: "interplan"` on the recorded port.
4. POST to `/shutdown` with header `X-Interplan-Token: <token>`.
5. Wait up to 3000ms for the process to exit.
6. Remove `server.json` after successful shutdown.

Output:

```toon
server:
  status: stopped
  port: 37917
next_step: Interplan server stopped.
```

If no server metadata exists, output:

```toon
server:
  status: not-running
next_step: No Interplan server was running.
```

### 8.4 Shutdown Endpoint

Add:

```http
POST /shutdown
```

Rules:

- Require `X-Interplan-Token` to match `server.json`.
- Return `403` on token mismatch.
- Return `200` before shutting down.
- Shutdown must only stop the current Interplan server process.

### 8.5 Idle Shutdown

Server must shut down automatically after 30 minutes of idle time.

Idle means all of these are true:

- No active SSE browser connections.
- No active long-poll requests.
- No open sessions updated in the last 30 minutes.

Environment variable:

```text
INTERPLAN_IDLE_TIMEOUT
```

Rules:

- Unset: `30m`.
- `0` or `off`: disable idle shutdown.
- Values must parse with Go `time.ParseDuration`.
- Invalid values fail server startup.

### 8.6 Tests

Add tests for:

- `server.json` creation.
- `stop` not-running output.
- `/shutdown` rejects bad token.
- `/shutdown` accepts correct token.
- Idle timeout parser.

## 9. Playbooks And Design Guidance

### 9.1 Commands

Implement:

```sh
interplan playbook <id>
interplan design
```

Both commands must support TOON by default and JSON with `--json`.

### 9.2 Playbook IDs

Implement these exact playbook IDs:

- `diagram`
- `table`
- `comparison`
- `plan`
- `code`
- `input`
- `slides`

Unknown ID error:

```text
unknown playbook "<id>"
```

### 9.3 Playbook Output Shape

TOON shape:

```toon
playbook:
  id: diagram
  title: Diagram Artifact
  purpose: Architecture, flows, state machines, Mermaid and node annotation guidance.
checks[3]{id,instruction}:
  structure,"Use clear nodes and directional relationships."
  labels,"Use stable node IDs and human-readable labels."
  review,"Add data-interplan-diagram to Mermaid containers."
next_step: Write the HTML artifact, then run interplan <file>.
```

JSON shape:

```json
{
  "playbook": {
    "id": "diagram",
    "title": "Diagram Artifact",
    "purpose": "Architecture, flows, state machines, Mermaid and node annotation guidance.",
    "checks": [
      {"id":"structure","instruction":"Use clear nodes and directional relationships."}
    ]
  },
  "next_step": "Write the HTML artifact, then run interplan <file>."
}
```

### 9.4 Required Playbook Content

`diagram` checks:

- Use clear nodes and directional relationships.
- Use stable node IDs and human-readable labels.
- Add `data-interplan-diagram` to Mermaid containers.
- Keep diagram text legible at 100% zoom.

`table` checks:

- Use sticky headers for long tables.
- Keep columns scannable and avoid horizontal overflow.
- Use concise cell text and expandable details for long content.
- Include summary counts when relevant.

`comparison` checks:

- Show current state, proposed state, tradeoffs, and recommendation.
- Use consistent row structure across options.
- Highlight differences without relying only on color.

`plan` checks:

- Include phases, dependencies, risks, and acceptance criteria.
- Make next actions explicit.
- Separate completed work from remaining work.

`code` checks:

- Reference file paths and functions clearly.
- Show before/after behavior.
- Keep code blocks horizontally scrollable only when intentional.

`input` checks:

- Use labels for every control.
- Add stable `name` or `data-interplan-queue-key` attributes.
- Use `data-interplan-question` for reversible choices.
- Use `data-interplan-action` for explicit decision buttons.

`slides` checks:

- Use one primary idea per slide.
- Keep text large and unclipped.
- Include slide numbers.

### 9.5 Design Command

`interplan design` must print design source priority and fallback design guidance.

TOON shape:

```toon
design:
  priority[3]: user-request, project-system, interplan-fallback
  fallback:
    layout: Use dense, readable artifact layouts with restrained styling.
    typography: Use system fonts, readable sizes, and no viewport-scaled text.
    color: Use accessible contrast and avoid single-hue monotony.
next_step: Apply this guidance before opening review.
```

### 9.6 Tests

Add tests for:

- Every playbook ID works.
- Unknown playbook fails.
- `design` prints expected priority order.
- TOON output decodes successfully.
- JSON output validates expected keys.

## 10. Export Command

### 10.1 Command

Implement:

```sh
interplan export <html-file> [--out <output-file>]
```

If `--out` is omitted, output path must be:

```text
<input basename>.export.html
```

Example:

```text
/tmp/plan.html -> /tmp/plan.export.html
```

### 10.2 Parsing

Use `golang.org/x/net/html` for HTML parsing and serialization. Do not use regular expressions for HTML parsing.

### 10.3 Asset Resolution

Resolve local relative assets from the artifact directory.

Inline these asset references:

- `<link rel="stylesheet" href="local.css">`
- `<script src="local.js"></script>`
- `<img src="local.png">`
- `<source src="local.webm">`
- CSS `url(...)` references inside local stylesheets.
- `@font-face` URLs inside local stylesheets.

Remote `http:` and `https:` references must remain unchanged.

Root-relative references such as `/assets/app.css` must be treated as unresolved local assets and left unchanged with a warning.

`file://` references must be removed and reported as rejected.

### 10.4 Data URI Rules

Use data URIs for binary assets.

MIME types:

- `.png`: `image/png`
- `.jpg`, `.jpeg`: `image/jpeg`
- `.gif`: `image/gif`
- `.svg`: `image/svg+xml`
- `.webp`: `image/webp`
- `.woff`: `font/woff`
- `.woff2`: `font/woff2`
- `.ttf`: `font/ttf`
- `.mp4`: `video/mp4`
- `.webm`: `video/webm`

For unknown binary extensions, use `application/octet-stream`.

### 10.5 Size Limits

Limits:

- Maximum single asset size: 10 MB.
- Maximum exported HTML size: 25 MB.

If an asset exceeds 10 MB:

- Leave the reference unchanged.
- Add warning `asset-too-large`.
- Continue export.

If final output would exceed 25 MB:

- Do not write the output file.
- Return non-zero exit.
- Print error in selected output format.

### 10.6 SDK Removal

The exported file must not contain Interplan review SDK injection, browser chrome scripts, or `/api/` references added by the review server.

Since the source artifact is not modified by review serving, export must read the source artifact from disk, not `/artifact/{key}/index.html`.

### 10.7 CSP Handling

If a CSP meta tag exists, update it to allow required inlined assets:

- Add `'unsafe-inline'` to `style-src` for inlined styles.
- Add `data:` to `img-src` when images are inlined.
- Add `data:` to `font-src` when fonts are inlined.
- Add `data:` to `media-src` when media is inlined.

Do not add `'unsafe-inline'` to `script-src`. Inline scripts from local files only when no CSP meta tag exists. If a CSP meta tag exists and a local script would need inlining, leave the script external and add warning `script-blocked-by-csp`.

### 10.8 Output Shape

TOON:

```toon
export:
  source: /path/artifact.html
  output: /path/artifact.export.html
  bytes: 123456
  unresolved_local_assets: 0
  rejected_file_urls: 0
warnings[1]{kind,path,message}:
  asset-too-large,/path/big.png,"Asset exceeds 10 MB and was left unchanged."
next_step: Wrote standalone HTML.
```

### 10.9 Tests

Add tests for:

- CSS file inlining.
- JS file inlining without CSP.
- Image data URI inlining.
- CSS `url(...)` inlining.
- Remote URLs preserved.
- Root-relative URLs warned and preserved.
- `file://` URLs removed and warned.
- Oversized asset warning.
- 25 MB output cap failure.

## 11. Share Command

### 11.1 Command

Implement:

```sh
interplan share <html-file> --provider local-dir --out-dir <directory>
```

This repository implements exactly one share provider: `local-dir`.

The command must:

1. Run export using the same implementation as `interplan export`.
2. Copy the exported HTML into `<directory>`.
3. Write a manifest file `<directory>/interplan-share.json`.
4. Print the absolute path to the shared file.

### 11.2 Required Flags

`--provider local-dir` is required.

`--out-dir <directory>` is required.

No other provider is accepted. Unknown provider error:

```text
unsupported share provider "<provider>"
```

### 11.3 Manifest Shape

```json
{
  "source": "/absolute/path/artifact.html",
  "output": "/absolute/path/shared/artifact.export.html",
  "provider": "local-dir",
  "created_at": "2026-07-07T12:00:00Z",
  "public": false
}
```

### 11.4 Output Shape

```toon
share:
  provider: local-dir
  source: /path/artifact.html
  output: /path/shared/artifact.export.html
  public: false
next_step: Shared exported HTML in local directory.
```

### 11.5 Tests

Add tests for:

- Missing provider fails.
- Missing out-dir fails.
- Unsupported provider fails.
- Exported HTML copied to out-dir.
- Manifest written with `public: false`.

## 12. Agent Hook Installer

### 12.1 Command

Implement:

```sh
interplan setup hooks
```

This command installs local Interplan agent guidance into:

```text
.agents/skills/interplan/SKILL.md
```

relative to the current working directory.

### 12.2 Behavior

The command must:

1. Create `.agents/skills/interplan/` if missing.
2. Write the current bundled Interplan skill markdown.
3. Preserve an existing file by copying it to `SKILL.md.bak` before overwrite.
4. Print output in TOON by default and JSON with `--json`.

### 12.3 Output Shape

```toon
hooks:
  skill_path: /repo/.agents/skills/interplan/SKILL.md
  backup_path: /repo/.agents/skills/interplan/SKILL.md.bak
  status: installed
next_step: Restart or refresh the agent so it can load the Interplan skill.
```

If no previous file existed, set `backup_path` to an empty string.

### 12.4 Tests

Add tests for:

- Directory creation.
- Skill file writing.
- Backup creation when file exists.
- JSON output.

## 13. Security Hardening

### 13.1 Iframe Sandbox

The artifact iframe must use this exact sandbox attribute:

```html
sandbox="allow-scripts allow-forms allow-same-origin allow-popups allow-downloads"
```

### 13.2 CSP Header For Chrome

The browser chrome route `/session/{key}` must set:

```http
Content-Security-Policy: default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self'; frame-src 'self';
```

### 13.3 Artifact Path Safety

Keep existing path traversal protection for `/artifact/{key}/assets/*`.

Add tests for encoded traversal attempts:

- `%2e%2e/secret.txt`
- `..%2fsecret.txt`
- `%2e%2e%2fsecret.txt`

All must return `403` or `404`; none may return file contents.

## 14. Test Requirements

Before marking this specification complete, all commands must pass:

```sh
gofmt -w .
go test ./...
go build -o ./bin/interplan ./cmd/interplan
```

Add or update `just test` and `just build` only if needed to preserve those commands.

## 15. Manual Acceptance Scenario

After all features are implemented, this scenario must pass:

```sh
cat > /tmp/interplan-acceptance.html <<'HTML'
<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Acceptance</title>
  <style>
    .card { width: 180px; height: 24px; overflow: hidden; }
  </style>
</head>
<body>
  <h1>Draft</h1>
  <section data-review-id="summary">
    <div class="card">This text is intentionally too long and should be clipped by the layout audit.</div>
  </section>
  <label for="priority">Priority</label>
  <select id="priority" name="priority" data-interplan-queue-key="priority-choice">
    <option value="mvp">MVP</option>
    <option value="polish">Polish</option>
  </select>
  <button data-interplan-action="confirm" data-interplan-prompt="User confirmed the plan.">Confirm</button>
  <div class="mermaid" data-interplan-diagram="acceptance-flow">
    graph TD
      Draft[Draft] --> Review[Review]
  </div>
</body>
</html>
HTML

interplan /tmp/interplan-acceptance.html
interplan poll /tmp/interplan-acceptance.html
```

Expected behavior:

1. Browser opens.
2. Open-time layout gate detects clipped text and posts a layout warning.
3. Poll returns the layout warning.
4. Agent fixes the clipped text.
5. Browser auto-reloads.
6. User selects a value, clicks Confirm, annotates Mermaid node `Review`, and clicks Send & End.
7. Poll returns structured input, action prompt, Mermaid node prompt, `session_ended: true`, and `ended_by: user`.
8. Agent stops polling.

## 16. Completion Definition

The remaining feature set is complete when:

- Every command in this file works.
- Every HTTP route in this file works.
- Every browser behavior in this file works.
- All specified tests pass.
- The manual acceptance scenario passes.
- `README.md` and `.agents/skills/interplan/SKILL.md` accurately describe the implemented behavior.
