#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HTML_FILE="${INTERPLAN_DEMO_HTML:-/tmp/interplan-complex-demo.html}"
PORT_FILE="${INTERPLAN_DEMO_PORT_FILE:-/tmp/interplan-complex-demo.port}"
BIN="${INTERPLAN_BIN:-$ROOT/bin/interplan}"

if [[ -z "${INTERPLAN_PORT:-}" ]]; then
  for candidate in $(seq 37918 38018); do
    if ! lsof -nP -iTCP:"$candidate" -sTCP:LISTEN >/dev/null 2>&1; then
      export INTERPLAN_PORT="$candidate"
      break
    fi
  done
fi
export INTERPLAN_PORT="${INTERPLAN_PORT:-37918}"
printf '%s\n' "$INTERPLAN_PORT" > "$PORT_FILE"

cat > "$HTML_FILE" <<'HTML'
<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Interplan Complex Demo</title>
  <style>
    body { margin: 0; font: 15px system-ui, sans-serif; background: #f4f6f8; color: #1f2933; }
    header { padding: 24px 32px; background: #172033; color: white; }
    main { display: grid; grid-template-columns: 1.4fr .9fr; gap: 20px; padding: 24px 32px; }
    section, aside { background: white; border: 1px solid #d8dee6; border-radius: 8px; padding: 18px; }
    h1, h2, h3 { margin-top: 0; }
    .cards { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; }
    .card { border: 1px solid #d8dee6; border-radius: 8px; padding: 14px; background: #fbfcfd; }
    .card strong { display: block; font-size: 18px; margin-top: 6px; }
    .flow { display: grid; gap: 10px; margin-top: 14px; }
    .step { display: flex; gap: 12px; align-items: flex-start; border: 1px solid #e1e6ed; padding: 12px; border-radius: 8px; }
    .num { width: 28px; height: 28px; border-radius: 50%; background: #174ea6; color: white; display: grid; place-items: center; font-weight: 700; }
    table { width: 100%; border-collapse: collapse; margin-top: 12px; }
    th, td { border-bottom: 1px solid #e1e6ed; text-align: left; padding: 10px; }
    label { display: block; margin: 10px 0 4px; font-weight: 600; }
    input, select, textarea { width: 100%; box-sizing: border-box; padding: 9px; border: 1px solid #bbc4cf; border-radius: 6px; font: inherit; }
    button { margin-top: 12px; padding: 9px 12px; border: 0; border-radius: 6px; background: #174ea6; color: white; font-weight: 700; }
    .warning { border-color: #f4c542; background: #fff9db; }
    .tag { display: inline-block; padding: 2px 7px; border-radius: 999px; background: #e8f0fe; color: #174ea6; font-size: 12px; }
  </style>
</head>
<body>
  <header data-review-id="hero">
    <span class="tag">Draft plan</span>
    <h1>Interplan rollout proposal</h1>
    <p>Review the interaction model, implementation phases, and risks before this becomes the default planning flow.</p>
  </header>

  <main>
    <section data-review-id="overview">
      <h2>Initial technical approach</h2>

      <div class="cards">
        <div class="card" data-testid="browser-review-card">
          <span class="tag">UI</span>
          <strong>Browser review</strong>
          <p>Users can inspect the artifact, annotate UI elements, and send feedback to the agent.</p>
        </div>

        <div class="card warning" data-testid="long-poll-card">
          <span class="tag">Agent loop</span>
          <strong>Long-poll API</strong>
          <p>The agent waits for comments, layout warnings, or the user ending the session.</p>
        </div>

        <div class="card" data-testid="state-store-card">
          <span class="tag">State</span>
          <strong>Local session store</strong>
          <p>Session state is persisted locally so feedback survives CLI restarts.</p>
        </div>
      </div>

      <div class="flow" data-review-id="flow">
        <div class="step">
          <div class="num">1</div>
          <div>
            <h3>Create artifact</h3>
            <p>The agent writes a complete HTML plan.</p>
          </div>
        </div>

        <div class="step">
          <div class="num">2</div>
          <div>
            <h3>Open review</h3>
            <p>The CLI starts the local server and opens the browser shell.</p>
          </div>
        </div>

        <div class="step">
          <div class="num">3</div>
          <div>
            <h3>Apply feedback</h3>
            <p>The agent receives structured prompts and updates the source artifact.</p>
          </div>
        </div>
      </div>
    </section>

    <aside data-review-id="decision-panel">
      <h2>Decision inputs</h2>

      <label for="priority">Priority</label>
      <select id="priority" name="priority">
        <option>Ship minimal browser annotation</option>
        <option>Add Mermaid node picking</option>
        <option>Build export/share first</option>
      </select>

      <label for="risk">Main risk</label>
      <textarea id="risk" rows="4">The current implementation may under-handle iframe coordination edge cases.</textarea>

      <button data-interplan-action="confirm-priority"
              data-interplan-prompt="User confirmed the current implementation priority.">
        Confirm priority
      </button>

      <h2 style="margin-top: 24px;">Phase table</h2>
      <table data-review-id="phase-table">
        <thead>
          <tr>
            <th>Phase</th>
            <th>Scope</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>Browser shell</td>
            <td>Iframe, comments, send/end</td>
            <td>Started</td>
          </tr>
          <tr>
            <td>Annotations</td>
            <td>Click target, selector, prompt</td>
            <td>In progress</td>
          </tr>
          <tr>
            <td>Layout gate</td>
            <td>Overflow and text clipping warnings</td>
            <td>Later</td>
          </tr>
        </tbody>
      </table>
    </aside>
  </main>
</body>
</html>
HTML

echo "Building $BIN"
(cd "$ROOT" && go build -o "$BIN" ./cmd/interplan)

echo "Wrote demo artifact: $HTML_FILE"
echo "Using Interplan port: $INTERPLAN_PORT"
echo "Wrote port file: $PORT_FILE"
echo
echo "Opening Interplan review..."
"$BIN" "$HTML_FILE"
echo
echo "Manual test steps:"
echo "1. Click Annotate in the browser."
echo "2. Click the yellow Long-poll API card."
echo "3. Enter: Rename this to Poll bridge; it is more precise."
echo "4. Click Queue."
echo "5. Click Send & End."
echo
echo "Then poll for feedback:"
echo "  INTERPLAN_PORT=$INTERPLAN_PORT $BIN poll $HTML_FILE --timeout-ms 1000"
