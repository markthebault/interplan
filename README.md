# Interplan

Interplan is a local browser review loop for coding agents.

It lets an agent create an HTML artifact, open it in your browser, collect your visual annotations and comments, then return that feedback to the agent through a structured CLI protocol.

Use it for things that are easier to review visually than in chat:

- implementation plans
- architecture diagrams
- UI mockups
- reports
- comparisons
- tables
- prototypes
- design reviews

## Quick Start

Install the Interplan CLI:

```sh
curl -fsSL https://raw.githubusercontent.com/markthebault/interplan/main/scripts/install.sh | sh
```

Install the Interplan skill in the [Agent Skills](https://agentskills.io) format with [`npx skills`](https://github.com/vercel-labs/skills):

```sh
npx skills add markthebault/interplan --skill interplan
```

Then ask your coding agent to create or open an Interplan artifact for browser review.

## Use With Coding Agents

After installing the skill, invoke it from your coding agent with a prompt and an HTML file path.

Pi:

```text
/skill:interplan Create a browser-reviewable implementation plan /tmp/plan.html
```

Codex:

```text
$interplan Create a browser-reviewable implementation plan /tmp/plan.html
```

You can also ask naturally if your agent auto-loads skills:

```text
Use Interplan to create a browser-reviewable architecture plan at /tmp/architecture.html
```

The agent should open the artifact, immediately run `interplan poll`, apply browser feedback, and continue until you end the session.

## Install Details

The CLI installer downloads the latest GitHub Release for your platform, verifies `checksums.txt`, and installs `interplan` into `/usr/local/bin` or `$HOME/.local/bin`.

To choose another install directory:

```sh
INTERPLAN_INSTALL_DIR="$HOME/bin" sh -c "$(curl -fsSL https://raw.githubusercontent.com/markthebault/interplan/main/scripts/install.sh)"
```

Coding agents need the Interplan skill so they know when to create browser-reviewable artifacts, how to poll for feedback, and when to stop. The skill install command places the skill in an Agent Skills-compatible location for supported coding agents.

## How it works

1. The agent writes a standalone HTML file.
2. The agent runs:

   ```sh
   interplan /absolute/path/to/artifact.html
   ```

3. Interplan starts a local server and opens a browser review page.
4. You review the artifact in the browser.
5. You click elements, add comments, or send general feedback.
6. The agent immediately waits for feedback with:

   ```sh
   interplan poll /absolute/path/to/artifact.html
   ```

7. Poll returns structured feedback as TOON by default, or JSON with `--json`.
8. The agent edits the artifact or project files.
9. The browser reloads automatically when the HTML file changes.
10. The loop continues until you click **Send & End** or end the session.

The key behavior is that the agent does not ask you to “let me know when done”. It opens the session, starts polling, and waits silently until browser feedback arrives.

## Basic usage

Create an artifact:

```sh
cat > /tmp/plan.html <<'HTML'
<!doctype html>
<html>
  <head><meta charset="utf-8"><title>Plan</title></head>
  <body>
    <h1>Implementation Plan</h1>
    <p>Review this plan and annotate anything that should change.</p>
  </body>
</html>
HTML
```

Open it for review:

```sh
interplan /tmp/plan.html
```

Wait for feedback:

```sh
interplan poll /tmp/plan.html
```

Use JSON output if needed:

```sh
interplan poll /tmp/plan.html --json
```

End a session from the CLI:

```sh
interplan end /tmp/plan.html
```

Check the installed binary version:

```sh
interplan --version
```

## CLI commands

```sh
interplan                         # list sessions
interplan <file.html>             # open or resume a browser review session
interplan open <file.html>        # same as above
interplan poll <file.html>        # wait for browser feedback
interplan end <file.html>         # end a session from the agent side
interplan server                  # run the local server in the foreground
interplan --version               # print the binary version
```

Useful flags:

```sh
--json                            # print JSON instead of TOON
--no-open                         # do not open the browser automatically
--expose-external                 # bind to 0.0.0.0 and return a LAN URL
--reopen                          # reopen a user-ended session
--timeout-ms 30000                # bound a poll wait
--agent-reply "Updated."          # send an agent status message before polling
--version                         # print the binary version
```

## What the browser provides

The browser review UI currently supports:

- local iframe preview of the artifact
- element annotation by clicking the page
- general comments
- Send and Send & End actions
- local state-backed feedback delivery
- automatic live reload when the artifact file changes
- scroll preservation across reloads

The source HTML file remains portable. Interplan injects review behavior only when serving the artifact through the local review server.

## Local state

Interplan stores session state locally.

macOS:

```text
~/Library/Application Support/interplan/state.json
```

Linux:

```text
${XDG_STATE_HOME:-~/.local/state}/interplan/state.json
```

Windows:

```text
%LocalAppData%\interplan\state.json
```

Feedback is queued in local state until `interplan poll` delivers it to the agent.

## Development

Run tests:

```sh
go test ./...
```

Build locally:

```sh
go build -o ./bin/interplan ./cmd/interplan
```

Local builds without release flags report `interplan dev`. Release binaries embed the GitHub release tag at build time, so `interplan --version` reports values like `interplan v0.2.0`.

Manual browser demo:

```sh
./scripts/manual-complex-demo.sh
```

## Releases

Interplan uses GitHub Actions, Conventional Commits, and Release Please for automated versioning and release binaries.

See [`doc/RELEASES.md`](doc/RELEASES.md).

## Roadmap

The remaining feature specification is tracked in [`doc/REMAINING_FEATURES_SPEC.md`](doc/REMAINING_FEATURES_SPEC.md).
