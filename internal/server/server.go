package server

import (
	"encoding/json"
	"errors"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/markthebault/interplan/internal/protocol"
	"github.com/markthebault/interplan/internal/session"
)

func Serve(addr string, store *session.Store) error {
	return http.ListenAndServe(addr, Handler(store))
}

func Handler(store *session.Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"name":"interplan","protocol_version":2}`))
	})
	mux.HandleFunc("/session/", func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/session/")
		if key == "" || strings.Contains(key, "/") {
			http.NotFound(w, r)
			return
		}
		sess, err := store.GetByKey(key)
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeChrome(w, sess)
	})
	mux.HandleFunc("/artifact/", func(w http.ResponseWriter, r *http.Request) {
		serveArtifact(w, r, store)
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/sessions" && r.Method == http.MethodGet {
			handleListSessions(w, r, store)
			return
		}
		if r.URL.Path == "/api/sessions" && r.Method == http.MethodPost {
			handleOpen(w, r, store)
			return
		}
		if r.URL.Path == "/api/poll" && r.Method == http.MethodGet {
			handlePoll(w, r, store)
			return
		}
		if r.URL.Path == "/api/end" && r.Method == http.MethodPost {
			handleEnd(w, r, store)
			return
		}
		if r.URL.Path == "/api/agent-reply" && r.Method == http.MethodPost {
			handleAgentReply(w, r, store)
			return
		}
		key, ok := promptKey(r.URL.Path)
		if ok && r.Method == http.MethodPost {
			handlePrompts(w, r, store, key)
			return
		}
		if key, ok := keyedAction(r.URL.Path, "layout-warnings"); ok && r.Method == http.MethodPost {
			handleLayoutWarnings(w, r, store, key)
			return
		}
		if key, ok := keyedAction(r.URL.Path, "end"); ok && r.Method == http.MethodPost {
			handleKeyEnd(w, r, store, key)
			return
		}
		http.NotFound(w, r)
	})
	return mux
}

func handleListSessions(w http.ResponseWriter, r *http.Request, store *session.Store) {
	state, err := store.Load()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out := protocol.SessionListResponse{NextStep: "Run `interplan <artifact.html>` to open a review session."}
	for _, s := range state.Sessions {
		out.Sessions = append(out.Sessions, protocol.SessionInfo{File: s.File, URL: s.URL, Status: s.Status})
	}
	writeJSON(w, http.StatusOK, out)
}

func handleOpen(w http.ResponseWriter, r *http.Request, store *session.Store) {
	var req protocol.SessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.File == "" {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	file, err := session.CanonicalPath(req.File)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	key := session.Key(file)
	sess, err := store.Open(file, session.URLFor(key, requestPort(r)), req.Reopen)
	var ended session.UserEndedError
	if errors.As(err, &ended) {
		writeJSON(w, http.StatusConflict, protocol.SessionResponse{
			Session:  protocol.SessionInfo{File: ended.Session.File, URL: ended.Session.URL, Status: "user-ended"},
			NextStep: "Review was ended by the user. Pass --reopen only if the user asked to continue.",
		})
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, protocol.SessionResponse{
		Session:  protocol.SessionInfo{File: sess.File, URL: sess.URL, Status: "opened"},
		NextStep: "Run `interplan poll " + sess.File + "`.",
	})
}

func handlePoll(w http.ResponseWriter, r *http.Request, store *session.Store) {
	rawFile := r.URL.Query().Get("file")
	if rawFile == "" {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	file, err := session.CanonicalPath(rawFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	timeout := parseTimeout(r.URL.Query().Get("timeoutMs"))
	deadline := time.Now().Add(timeout)
	for {
		out, err := store.Poll(file)
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if out.Session.Status != "waiting" || timeout == 0 || time.Now().After(deadline) {
			writeJSON(w, http.StatusOK, out)
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func handleEnd(w http.ResponseWriter, r *http.Request, store *session.Store) {
	var req struct {
		File    string `json:"file"`
		EndedBy string `json:"ended_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.File == "" {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	file, err := session.CanonicalPath(req.File)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	endedBy := req.EndedBy
	if endedBy == "" {
		endedBy = "agent"
	}
	sess, err := store.End(file, endedBy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, protocol.SessionResponse{
		Session:  protocol.SessionInfo{File: sess.File, URL: sess.URL, Status: sess.Status},
		NextStep: "Session ended by " + endedBy + ".",
	})
}

func handleAgentReply(w http.ResponseWriter, r *http.Request, store *session.Store) {
	var req struct {
		File    string `json:"file"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.File == "" {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	file, err := session.CanonicalPath(req.File)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := store.AppendAgentReply(file, req.Message); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handlePrompts(w http.ResponseWriter, r *http.Request, store *session.Store, key string) {
	var post session.PromptPost
	if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sess, err := store.AddPrompts(key, post)
	if errors.Is(err, os.ErrNotExist) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"pending_prompts": sess.PendingPrompts,
		"session_ended":   sess.Status == "ended",
	})
}

func handleLayoutWarnings(w http.ResponseWriter, r *http.Request, store *session.Store, key string) {
	var post session.LayoutWarningPost
	if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, err := store.AddLayoutWarnings(key, post); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleKeyEnd(w http.ResponseWriter, r *http.Request, store *session.Store, key string) {
	sess, err := store.GetByKey(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := store.End(sess.File, "user"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func promptKey(path string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[0] != "api" || parts[2] != "prompts" || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func keyedAction(path, action string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[0] != "api" || parts[2] != action || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func serveArtifact(w http.ResponseWriter, r *http.Request, store *session.Store) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/artifact/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	key := parts[0]
	sess, err := store.GetByKey(key)
	if errors.Is(err, os.ErrNotExist) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	root := filepath.Dir(sess.File)
	target := sess.File
	if len(parts) > 1 && parts[1] == "assets" {
		rel := filepath.Clean(strings.Join(parts[2:], string(filepath.Separator)))
		target, err = filepath.Abs(filepath.Join(root, rel))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rootAbs, err := filepath.Abs(root)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if target != rootAbs && !strings.HasPrefix(target, rootAbs+string(filepath.Separator)) {
			http.Error(w, "asset path escapes artifact directory", http.StatusForbidden)
			return
		}
		http.ServeFile(w, r, target)
		return
	}
	if parts[1] != "index.html" {
		http.NotFound(w, r)
		return
	}
	data, err := os.ReadFile(target)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(injectSDK(data, key))
}

func injectSDK(data []byte, key string) []byte {
	script := []byte(`<script>
(function(){
  const key = ` + strconv.Quote(key) + `;
  function queuePrompt(prompt){
    return fetch("/api/"+key+"/prompts",{method:"POST",headers:{"content-type":"application/json"},body:JSON.stringify({prompts:[prompt]})});
  }
  window.interplan = {key, queuePrompt};
  window.parent.postMessage({type:"interplan:ready"},"*");
})();
</script>`)
	lower := strings.ToLower(string(data))
	idx := strings.LastIndex(lower, "</body>")
	if idx < 0 {
		return append(data, script...)
	}
	out := make([]byte, 0, len(data)+len(script))
	out = append(out, data[:idx]...)
	out = append(out, script...)
	out = append(out, data[idx:]...)
	return out
}

func writeChrome(w http.ResponseWriter, sess *session.Session) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	body := `<!doctype html>
<meta charset="utf-8">
<title>Interplan Review</title>
<style>
body{margin:0;font:14px system-ui,sans-serif;display:grid;grid-template-columns:minmax(0,1fr) 340px;grid-template-rows:48px minmax(0,1fr);height:100vh;color:#1f2933;background:#f6f7f9}
header{grid-column:1 / -1;display:flex;align-items:center;gap:10px;padding:0 12px;border-bottom:1px solid #d6dbe1;background:#fff}
header strong{font-size:14px}.spacer{flex:1}
iframe{width:100%;height:100%;border:0;background:white}
main{min-width:0;min-height:0}
aside{border-left:1px solid #d6dbe1;background:#fff;padding:16px;display:flex;flex-direction:column;gap:12px;min-height:0;overflow:auto}
h1{font-size:16px;margin:0}.meta{font-size:12px;color:#657282;word-break:break-all}
textarea{min-height:160px;resize:vertical;font:inherit;padding:10px;border:1px solid #bbc4cf;border-radius:6px}
.actions{display:flex;gap:8px;flex-wrap:wrap}button{padding:8px 10px;border:1px solid #9aa7b5;background:#f8fafc;border-radius:6px;cursor:pointer}button.primary{background:#174ea6;color:white;border-color:#174ea6}button.active{background:#111827;color:#fff;border-color:#111827}
#status{font-size:12px;color:#52606d}
body.annotating iframe{cursor:crosshair}
.chips{display:flex;flex-direction:column;gap:8px}.chip{border:1px solid #d6dbe1;border-radius:6px;padding:8px;background:#f8fafc}.chip strong{display:block;font-size:12px}.chip code{font-size:11px;color:#52606d;word-break:break-all}.chip p{margin:6px 0 0}
.modal-backdrop{position:fixed;inset:0;background:rgba(15,23,42,.28);display:none;align-items:center;justify-content:center;z-index:10}.modal-backdrop.open{display:flex}
.modal{width:min(620px,calc(100vw - 32px));background:#0b1017;color:#f8fafc;border:1px solid #273241;border-radius:12px;box-shadow:0 20px 60px rgba(0,0,0,.35);padding:18px;display:flex;flex-direction:column;gap:14px}
.modal h2{font-size:18px;margin:0}.modal textarea{background:#071015;color:#f8fafc;border-color:#273241;min-height:120px}.modal .target{font-size:12px;color:#a7b2c1;word-break:break-all}.modal button.primary{background:#f6c84c;color:#111827;border-color:#f6c84c;font-weight:700}
</style>
<header><strong>Interplan</strong><button id="annotate">Annotate</button><button id="reload">Reload</button><span class="spacer"></span><button id="endOnly">End Session</button></header>
<main><iframe id="artifact" src="/artifact/` + html.EscapeString(sess.Key) + `/index.html"></iframe></main>
<aside>
<h1>Interplan Review</h1>
<div class="meta">` + html.EscapeString(sess.File) + `</div>
<div class="chips" id="chips"></div>
<textarea id="prompt" placeholder="Comment for the agent"></textarea>
<div class="actions"><button class="primary" id="send">Send</button><button id="end">Send & End</button></div>
<div id="status"></div>
</aside>
<div class="modal-backdrop" id="modal">
  <div class="modal">
    <h2 id="modalTitle">Annotate element</h2>
    <div class="target" id="modalTarget"></div>
    <textarea id="annotationText" placeholder="Annotation for this UI element"></textarea>
    <div class="actions"><button id="cancelAnnotation">Cancel</button><button class="primary" id="queueAnnotation">Queue</button></div>
  </div>
</div>
<script>
const key = ` + strconv.Quote(sess.Key) + `;
const frame = document.getElementById("artifact");
const prompt = document.getElementById("prompt");
const statusEl = document.getElementById("status");
const annotateBtn = document.getElementById("annotate");
const chips = document.getElementById("chips");
const modal = document.getElementById("modal");
const annotationText = document.getElementById("annotationText");
const modalTitle = document.getElementById("modalTitle");
const modalTarget = document.getElementById("modalTarget");
let annotate = false;
let pendingTarget = null;
let queued = [];
let frameClickHandler = null;
function cssEscape(value){
  if(window.CSS && CSS.escape) return CSS.escape(value);
  return String(value).replace(/[^a-zA-Z0-9_-]/g, "\\$&");
}
function selectorFor(el, doc){
  if(el.id && doc.querySelectorAll("#"+cssEscape(el.id)).length === 1) return "#"+cssEscape(el.id);
  for(const attr of ["data-review-id","data-testid","data-id"]){
    const value = el.getAttribute(attr);
    if(value){
      const safe = value.replace(/\\/g, "\\\\").replace(/"/g, "\\\"");
      const sel = el.tagName.toLowerCase()+"["+attr+"=\""+safe+"\"]";
      if(doc.querySelectorAll(sel).length === 1) return sel;
    }
  }
  const parts = [];
  let node = el;
  while(node && node.nodeType === 1 && node !== doc.documentElement){
    const tag = node.tagName.toLowerCase();
    let index = 1;
    let sib = node;
    while((sib = sib.previousElementSibling)){
      if(sib.tagName.toLowerCase() === tag) index++;
    }
    parts.unshift(tag+":nth-of-type("+index+")");
    node = node.parentElement;
    if(parts.length >= 6) break;
  }
  return parts.join(" > ");
}
function textFor(el){
  return (el.innerText || el.textContent || "").replace(/\s+/g, " ").trim().slice(0, 240);
}
function pickAnnotationTarget(start){
  if(!start || !start.closest) return start;
  return start.closest("[data-review-id],[data-testid],[data-id]") ||
    start.closest("article,section,aside,header,main,nav,div,li,tr,td,th,button,a,label") ||
    start;
}
function attachFrameAnnotation(){
  let doc;
  try { doc = frame.contentDocument; } catch { doc = null; }
  if(!doc || !doc.body) {
    statusEl.textContent = "Annotation unavailable for this artifact frame.";
    return;
  }
  if(frameClickHandler) doc.removeEventListener("click", frameClickHandler, true);
  frameClickHandler = event => {
    if(!annotate) return;
    const el = pickAnnotationTarget(event.target);
    if(!el) return;
    event.preventDefault();
    event.stopPropagation();
    event.stopImmediatePropagation();
    openAnnotation({
      kind:"element",
      tag:el.tagName.toLowerCase(),
      selector:selectorFor(el, doc),
      text:textFor(el)
    });
  };
  doc.addEventListener("click", frameClickHandler, true);
  doc.documentElement.dataset.interplanAnnotate = annotate ? "true" : "false";
}
function setAnnotate(next){
  annotate = next;
  annotateBtn.classList.toggle("active", annotate);
  document.body.classList.toggle("annotating", annotate);
  annotateBtn.textContent = annotate ? "Annotating" : "Annotate";
  attachFrameAnnotation();
  statusEl.textContent = annotate ? "Annotation mode: click an element in the artifact." : "";
}
function renderChips(){
  chips.innerHTML = "";
  queued.forEach((item, index) => {
    const chip = document.createElement("div");
    chip.className = "chip";
    chip.innerHTML = "<strong>"+item.target.tag+" annotation</strong><code>"+item.selector+"</code><p>"+item.prompt+"</p>";
    const remove = document.createElement("button");
    remove.textContent = "Remove";
    remove.onclick = () => { queued.splice(index,1); renderChips(); };
    chip.appendChild(remove);
    chips.appendChild(chip);
  });
}
function openAnnotation(target){
  pendingTarget = target;
  modalTitle.textContent = "Annotate <"+target.tag+">";
  modalTarget.textContent = target.selector + (target.text ? " - " + target.text : "");
  annotationText.value = "";
  modal.classList.add("open");
  setTimeout(() => annotationText.focus(), 0);
}
function closeAnnotation(){
  modal.classList.remove("open");
  pendingTarget = null;
}
async function send(endSession){
  const text = prompt.value.trim();
  const prompts = queued.slice();
  if(text) prompts.push({tag:"message",prompt:text});
  if(!prompts.length && !endSession){ statusEl.textContent = "Type a comment or queue an annotation first."; return; }
  const res = await fetch("/api/"+key+"/prompts",{method:"POST",headers:{"content-type":"application/json"},body:JSON.stringify({prompts,domSnapshot:"",endSession})});
  statusEl.textContent = res.ok ? (endSession ? "Sent and ended." : "Sent.") : "Send failed.";
  if(res.ok){ prompt.value = ""; queued = []; renderChips(); }
}
window.addEventListener("message", event => {
  if(event.source !== frame.contentWindow) return;
  if(event.data && event.data.type === "interplan:ready") attachFrameAnnotation();
});
annotateBtn.onclick = () => setAnnotate(!annotate);
frame.addEventListener("load", () => attachFrameAnnotation());
document.getElementById("reload").onclick = () => frame.contentWindow.location.reload();
document.getElementById("send").onclick = () => send(false);
document.getElementById("end").onclick = () => send(true);
document.getElementById("endOnly").onclick = async () => {
  const res = await fetch("/api/"+key+"/end",{method:"POST"});
  statusEl.textContent = res.ok ? "Ended." : "End failed.";
};
document.getElementById("cancelAnnotation").onclick = closeAnnotation;
document.getElementById("queueAnnotation").onclick = () => {
  const text = annotationText.value.trim();
  if(!text || !pendingTarget) return;
  queued.push({
    tag:"element",
    prompt:text,
    text:pendingTarget.text || "",
    selector:pendingTarget.selector,
    target:{kind:"element", selector:pendingTarget.selector, tag:pendingTarget.tag, text:pendingTarget.text || ""}
  });
  renderChips();
  closeAnnotation();
};
modal.addEventListener("click", event => { if(event.target === modal) closeAnnotation(); });
document.addEventListener("keydown", event => { if(event.key === "Escape" && modal.classList.contains("open")) closeAnnotation(); });
</script>`
	_, _ = w.Write([]byte(body))
}

func parseTimeout(raw string) time.Duration {
	if raw == "" {
		return 24 * time.Hour
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms <= 0 {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

func requestPort(r *http.Request) int {
	_, port, ok := strings.Cut(r.Host, ":")
	if !ok {
		return 37917
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		return 37917
	}
	return p
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
