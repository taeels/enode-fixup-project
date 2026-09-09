package panel

import "net/http"

// handleIndex 는 제어판 화면을 낸다.
//
// 레이아웃과 꾸밈은 시안(design/*.pen · 진행자)이 정한다 — 여기서는 화면이
// 읽고 쓰는 값과 버튼이 전부 서는 기능판을 낸다. CP4 는 버튼을 전부 센다
// (status · start/restart · stop · logs + drain 걸기 · 모드 · 풀기 ·
// scene-gates §2.1). 문자열은 사용자에게 나가는 출력이라 영어로 쓰고, 장식
// 문자를 넣지 않는다 (glyphscan 이 이 리터럴도 읽는다).
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

const indexHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>enode control panel</title>
<style>
  body { font-family: system-ui, sans-serif; margin: 1.5rem; max-width: 52rem; }
  h1 { font-size: 1.2rem; }
  .card { border: 1px solid #ccc; border-radius: 6px; padding: 1rem; margin: 0.8rem 0; }
  .card h2 { font-size: 1rem; margin: 0 0 0.5rem; }
  .badge { display: inline-block; padding: 0.1rem 0.5rem; border-radius: 4px; background: #eee; font-size: 0.85rem; }
  .badge.draining { background: #fde68a; }
  .badge.stopped { background: #fecaca; }
  .badge.running { background: #bbf7d0; }
  button { margin: 0.2rem 0.3rem 0.2rem 0; padding: 0.35rem 0.7rem; }
  .muted { color: #666; }
  pre { background: #f6f6f6; padding: 0.7rem; overflow: auto; max-height: 22rem; }
  dl { display: grid; grid-template-columns: 9rem 1fr; gap: 0.2rem 0.5rem; margin: 0; }
  dt { color: #555; }
</style>
</head>
<body>
<h1>enode control panel &mdash; <span id="node"></span></h1>

<div class="card" id="identity-card">
  <h2>identity</h2>
  <dl id="identity"></dl>
</div>

<div class="card" id="process-card">
  <h2>process <span id="process-badge" class="badge"></span></h2>
  <div id="process-actions"></div>
</div>

<div class="card" id="caps-card">
  <h2>detected capabilities <span class="muted">(read-only)</span></h2>
  <div id="caps"></div>
</div>

<div class="card" id="work-card">
  <h2>current work</h2>
  <div id="work"></div>
</div>

<div class="card" id="drain-card">
  <h2>drain</h2>
  <div id="drain"></div>
</div>

<div class="card" id="mediator-card">
  <h2>mediator</h2>
  <div id="mediator"></div>
</div>

<div class="card" id="logs-card" style="display:none">
  <h2>daemon log</h2>
  <pre id="logs"></pre>
</div>

<script>
function esc(v){ return String(v == null ? "" : v).replace(/[&<>]/g, function(c){ return {"&":"&amp;","<":"&lt;",">":"&gt;"}[c]; }); }
function fmt(t){ return t ? new Date(t).toLocaleString() : ""; }

function render(st){
  document.getElementById("node").textContent = st.node || "";

  var id = st.identity || {};
  document.getElementById("identity").innerHTML =
    "<dt>node_id</dt><dd>" + esc(id.node_id) + "</dd>" +
    "<dt>label</dt><dd>" + esc(id.label) + "</dd>" +
    "<dt>principal</dt><dd>" + esc(id.principal) + "</dd>" +
    "<dt>instance</dt><dd>" + (id.instance ? esc(id.instance) : "<span class='muted'>(stopped or expired)</span>") + "</dd>";

  var pb = document.getElementById("process-badge");
  var running = st.process && st.process.running;
  pb.textContent = running ? ("running pid=" + st.process.pid) : "stopped";
  pb.className = "badge " + (running ? "running" : "stopped");

  // process actions: status(refresh) + logs always; start when stopped (S5),
  // restart when running (stop then start). stop when running.
  var pa = "";
  pa += "<button onclick='load()'>refresh status</button>";
  pa += "<button onclick='showLogs()'>logs</button>";
  if (running){
    pa += "<button onclick='doStop()'>stop</button>";
    pa += "<button onclick='doRestart()'>restart</button>";
  } else {
    pa += "<button onclick='doStart()'>start</button>";
  }
  document.getElementById("process-actions").innerHTML = pa;

  var caps = st.caps || {};
  if (!caps.known){
    document.getElementById("caps").innerHTML = "<span class='muted'>not known yet (the daemon has not written the status file)</span>";
  } else {
    var rows = (caps.caps || []).map(function(c){
      var attrs = Object.keys(c.attrs || {}).map(function(k){ return k + "=" + c.attrs[k]; }).join(" ");
      return "<li>" + esc(c.capability) + (attrs ? " <span class='muted'>" + esc(attrs) + "</span>" : "") + "</li>";
    }).join("");
    document.getElementById("caps").innerHTML =
      "<ul>" + rows + "</ul><div class='muted'>detected at " + esc(fmt(caps.at)) + "</div>";
  }

  var wk = st.work || {};
  if (wk.has_lease){
    document.getElementById("work").innerHTML =
      "<dl><dt>run</dt><dd>" + esc(wk.run_id) + "</dd>" +
      "<dt>step</dt><dd>" + esc(wk.step) + (wk.attempt ? " (attempt " + wk.attempt + ")" : "") + "</dd>" +
      "<dt>started</dt><dd>" + esc(fmt(wk.started_at)) + "</dd>" +
      "<dt>lease until</dt><dd>" + esc(fmt(wk.not_after)) + "</dd></dl>";
  } else {
    document.getElementById("work").innerHTML = "<span class='muted'>no run right now</span>";
  }

  var dr = document.getElementById("drain");
  if (st.drain){
    // built (S3b): current mode badge + undrain
    dr.innerHTML = "<span class='badge draining'>draining: " + esc(st.drain) + "</span> " +
      "<button onclick='doUndrain()'>undrain (return to candidates)</button>";
  } else {
    // before drain (S3): drain button + mode select
    dr.innerHTML =
      "<select id='mode'><option value='graceful'>graceful (finish the running step)</option>" +
      "<option value='at-boundary'>at-boundary (release at the step boundary)</option></select> " +
      "<button onclick='doDrain()'>drain</button>";
  }

  var md = st.mediator || {};
  document.getElementById("mediator").innerHTML = md.reachable
    ? "<span class='muted'>reachable. last response " + esc(fmt(md.last_response)) + "</span>"
    : "<span class='muted'>unreachable. last response " + (md.last_response ? esc(fmt(md.last_response)) : "never") + ". drain notices to the fleet are delayed.</span>";
}

function load(){ fetch("/api/state").then(function(r){ return r.json(); }).then(render).catch(function(e){ alert("cannot load state: " + e); }); }

function post(path){ return fetch(path, {method:"POST"}).then(function(r){ return r.json(); }); }

function doDrain(){
  var mode = document.getElementById("mode").value;
  post("/api/drain?mode=" + encodeURIComponent(mode)).then(function(){ load(); });
}
function doUndrain(){ post("/api/undrain").then(function(){ load(); }); }

function doStop(){
  if (!confirm("Stop this node. The running run, if any, is cancelled first so the record shows you stopped it. If the mediator is unreachable it dies as lease-expired instead. Continue?")) return;
  post("/api/stop").then(function(res){ if (res.message) alert(res.message); load(); });
}
function doStart(){ post("/api/start").then(function(res){ if (res.message) alert(res.message); load(); }); }
function doRestart(){
  if (!confirm("Restart: stop (cancelling the running run first) then start. Continue?")) return;
  post("/api/stop").then(function(){ return post("/api/start"); }).then(function(res){ if (res.message) alert(res.message); load(); });
}

function showLogs(){
  document.getElementById("logs-card").style.display = "block";
  fetch("/api/logs").then(function(r){ return r.text(); }).then(function(t){ document.getElementById("logs").textContent = t; });
}

load();
setInterval(load, 5000);
</script>
</body>
</html>
`
