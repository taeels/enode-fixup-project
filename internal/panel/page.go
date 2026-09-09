package panel

import "net/http"

// handleIndex 는 제어판 화면을 낸다.
//
// 시안 design/enode-ux.pen 의 제어판 화면(S3 도는 노드 · S5 멈춤 · C3 drain 건 뒤)에
// 맞춘 다크 2단(사이드바 + 본문) 레이아웃이다. 색·폰트·반경은 그 파일의 디자인
// 토큰을 그대로 옮겼다(bg #0B0D10 · surface #14181D · 보라 강조 #B98CFF 등). 값과
// 동작은 /api/* 가 지고, 이 페이지는 그것을 시안 모양으로 그린다. 버튼은 CP4 가
// 세는 것을 전부 낸다 — status(새로고침) · start · stop · logs + drain 걸기 · 모드 ·
// 풀기 (scene-gates §2.1). 장식 문자를 문자열에 넣지 않는다(glyphscan) — 배지 점 ·
// 진행 표시는 CSS 로 그린다. 가운뎃점만 구분자로 쓴다(glyphscan 면제).
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(indexHTML))
}

const indexHTML = `<!doctype html>
<html lang="ko">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>enode 호스트 제어판</title>
<style>
  :root{
    --bg:#0B0D10; --surface:#14181D; --surface-2:#1B2027; --border:#262D36; --border-strong:#39434F;
    --tp:#E6EAF0; --ts:#98A4B3; --td:#5F6B7A;
    --idle:#3FBF87; --leased:#4B9CFF; --drain:#B98CFF; --queued:#E2A33C; --stopped:#F0574B;
    --radius:10px;
    --mono:'JetBrains Mono', ui-monospace, SFMono-Regular, Menlo, monospace;
    --body:'Inter', system-ui, -apple-system, Segoe UI, Roboto, sans-serif;
  }
  *{ box-sizing:border-box; }
  body{ margin:0; background:var(--bg); color:var(--tp); font-family:var(--body); display:flex; min-height:100vh; }
  a{ color:inherit; }

  aside{ width:248px; flex:none; background:var(--surface); border-right:1px solid var(--border); padding:20px 16px; }
  aside h1{ font-size:14px; letter-spacing:.02em; margin:0 0 16px; color:var(--tp); }
  .nodeitem{ background:var(--surface-2); border:1px solid var(--border); border-radius:var(--radius); padding:10px 12px; }
  .nodeitem.sel{ border-color:var(--border-strong); }
  .nodeitem .nm{ font-family:var(--mono); font-size:13px; }
  .nodeitem .sub{ color:var(--ts); font-size:11px; margin-top:3px; }
  aside .note{ color:var(--td); font-size:11px; line-height:1.5; margin-top:16px; }

  main{ flex:1; padding:24px 28px; max-width:1180px; }
  .head{ display:flex; align-items:center; gap:12px; }
  .dot{ width:9px; height:9px; border-radius:50%; flex:none; }
  .title{ font-family:var(--mono); font-size:18px; }
  .badge{ font-size:11px; padding:3px 9px; border-radius:999px; border:1px solid var(--border-strong); color:var(--ts); }
  .badge .dot{ display:inline-block; margin-right:6px; vertical-align:middle; }
  .grow{ flex:1; }
  .subline{ color:var(--td); font-family:var(--mono); font-size:12px; margin:8px 0 0; word-break:break-all; }

  button{ font-family:var(--body); font-size:13px; padding:7px 14px; border-radius:8px; border:1px solid var(--border-strong);
    background:var(--surface-2); color:var(--tp); cursor:pointer; }
  button:hover{ border-color:var(--ts); }
  button.primary{ background:var(--drain); border-color:var(--drain); color:#160c24; font-weight:600; }
  button.danger{ border-color:var(--stopped); color:var(--stopped); }
  .actions{ display:flex; gap:8px; }

  .card{ background:var(--surface); border:1px solid var(--border); border-radius:var(--radius); padding:18px 20px; margin-top:16px; }
  .card h2{ font-size:12px; text-transform:uppercase; letter-spacing:.08em; color:var(--ts); margin:0 0 14px; font-weight:600; }
  .muted{ color:var(--td); }

  .runid{ font-family:var(--mono); font-size:22px; }
  .lease{ font-family:var(--mono); font-size:20px; color:var(--leased); }
  dl{ display:grid; grid-template-columns:8rem 1fr; gap:8px 12px; margin:0; }
  dt{ color:var(--ts); font-size:12px; }
  dd{ margin:0; font-family:var(--mono); font-size:13px; word-break:break-all; }

  .modes{ display:flex; flex-direction:column; gap:10px; margin:6px 0 14px; }
  .mode{ display:flex; gap:10px; align-items:flex-start; background:var(--surface-2); border:1px solid var(--border); border-radius:8px; padding:10px 12px; cursor:pointer; }
  .mode.on{ border-color:var(--drain); }
  .mode input{ margin-top:3px; accent-color:var(--drain); }
  .mode .mt{ font-family:var(--mono); font-size:13px; }
  .mode .md{ color:var(--ts); font-size:12px; margin-top:2px; }

  .chips{ display:flex; flex-wrap:wrap; gap:8px; }
  .chip{ background:var(--surface-2); border:1px solid var(--border); border-radius:6px; padding:5px 9px; font-family:var(--mono); font-size:12px; }
  .chip .k{ color:var(--ts); }

  pre{ background:var(--bg); border:1px solid var(--border); border-radius:8px; padding:12px; overflow:auto; max-height:22rem; font-family:var(--mono); font-size:12px; color:var(--ts); }
</style>
</head>
<body>
<aside>
  <h1>호스트 제어판</h1>
  <div class="nodeitem sel">
    <div class="nm" id="side-name"></div>
    <div class="sub" id="side-state"></div>
  </div>
  <div class="note">이 기계에 접속한 것이 소유의 증거다. 127.0.0.1 은 인증 없이 열린다.
    LAN 으로 열려면 정책 파일의 panel_token 이 필요하다.</div>
</aside>

<main>
  <div class="head">
    <span class="dot" id="head-dot"></span>
    <span class="title" id="head-title"></span>
    <span class="badge" id="head-badge"></span>
    <span class="grow"></span>
    <span class="actions" id="head-actions"></span>
  </div>
  <div class="subline" id="subline"></div>

  <div class="card">
    <h2>현재 작업</h2>
    <div id="work"></div>
  </div>

  <div class="card">
    <h2>자원 회수 (drain)</h2>
    <div id="drain"></div>
  </div>

  <div class="card">
    <h2>탐지 능력 (읽기 전용)</h2>
    <div id="caps"></div>
  </div>

  <div class="card">
    <h2>데몬 로그</h2>
    <div><button onclick="showLogs()">로그 불러오기</button></div>
    <pre id="logs" style="display:none"></pre>
  </div>

  <div class="subline" id="mediator"></div>
</main>

<script>
function esc(v){ return String(v==null?"":v).replace(/[&<>]/g,function(c){return {"&":"&amp;","<":"&lt;",">":"&gt;"}[c];}); }
function fmt(t){ return t ? new Date(t).toLocaleString() : ""; }
function el(id){ return document.getElementById(id); }

function render(st){
  var running = st.process && st.process.running;
  var id = st.identity || {};

  el("side-name").textContent = st.node || "";
  el("side-state").textContent = running ? ("도는 중 · pid " + st.process.pid) : "멈춤";

  el("head-dot").style.background = running ? "var(--idle)" : "var(--stopped)";
  el("head-title").textContent = id.label || st.node || "";
  var badge = el("head-badge");
  badge.innerHTML = "<span class='dot' style='background:" + (running?"var(--idle)":"var(--stopped)") + "'></span>" + (running?"도는 중":"멈춤");

  var a = "";
  a += "<button onclick='load()'>새로고침</button>";
  a += "<button onclick='showLogs()'>로그</button>";
  if(running){
    a += "<button onclick='doRestart()'>재시작</button>";
    a += "<button class='danger' onclick='doStop()'>정지</button>";
  } else {
    a += "<button class='primary' onclick='doStart()'>시작</button>";
  }
  el("head-actions").innerHTML = a;

  el("subline").textContent = [
    "node_id " + (id.node_id||"?"),
    "instance " + (id.instance||"(멈춤/만료)"),
    id.principal||""
  ].join("  ·  ");

  var wk = st.work || {};
  if(wk.has_lease){
    el("work").innerHTML =
      "<div class='head'><span class='runid'>" + esc(wk.run_id) + "</span><span class='grow'></span>" +
      (wk.not_after ? "<span class='lease'>" + esc(fmt(wk.not_after)) + " 까지</span>" : "") + "</div>" +
      "<dl style='margin-top:12px'><dt>단계</dt><dd>" + esc(wk.step||"?") + (wk.attempt?(" · 회차 "+wk.attempt):"") + "</dd>" +
      "<dt>시작</dt><dd>" + esc(fmt(wk.started_at)) + "</dd></dl>";
  } else {
    el("work").innerHTML = "<span class='muted'>지금 도는 작업 없음</span>";
  }

  var dr = el("drain");
  if(st.drain){
    dr.innerHTML =
      "<div class='head'><span class='title' style='font-size:14px'>되찾은 상태</span>" +
      "<span class='grow'></span><span class='badge' style='color:var(--drain);border-color:var(--drain)'>draining · " + esc(st.drain) + "</span></div>" +
      "<p class='muted' style='margin:10px 0 14px'>도는 단계까지 마친 뒤 이 노드는 후보에서 빠져 있다. 소유자가 풀어야 후보로 돌아온다.</p>" +
      "<button class='primary' onclick='doUndrain()'>drain 풀기 · 형태로 돌려주기</button>";
  } else {
    dr.innerHTML =
      "<div class='modes'>" +
      "<label class='mode on'><input type='radio' name='mode' value='graceful' checked><div><div class='mt'>graceful</div><div class='md'>도는 단계까지 마치고 임대를 놓는다. 놀라움이 적은 기본값.</div></div></label>" +
      "<label class='mode'><input type='radio' name='mode' value='at-boundary'><div><div class='mt'>at-boundary</div><div class='md'>단계 경계에서 임대를 놓고 취소 경로로 종료한다. 남은 단계는 새 Run 으로.</div></div></label>" +
      "</div><button class='primary' onclick='doDrain()'>drain 걸기</button>";
    for(var m of dr.querySelectorAll("input[name=mode]")){
      m.addEventListener("change", function(){
        for(var lab of dr.querySelectorAll(".mode")) lab.classList.remove("on");
        this.closest(".mode").classList.add("on");
      });
    }
  }

  var caps = st.caps || {};
  if(!caps.known){
    el("caps").innerHTML = "<span class='muted'>아직 모름 — 데몬이 상태 파일을 아직 안 썼다</span>";
  } else {
    var chips = (caps.caps||[]).map(function(c){
      var attrs = Object.keys(c.attrs||{}).map(function(k){ return "<span class='k'>"+esc(k)+"</span>="+esc(c.attrs[k]); }).join("  ");
      return "<span class='chip'>" + esc(c.capability) + (attrs?("  "+attrs):"") + "</span>";
    }).join("");
    el("caps").innerHTML = "<div class='chips'>" + chips + "</div>" +
      "<div class='muted' style='margin-top:12px;font-size:12px'>측정 시각 " + esc(fmt(caps.at)) + "</div>";
  }

  var md = st.mediator || {};
  el("mediator").textContent = md.reachable
    ? ("Mediator 도달 가능 · 마지막 응답 " + fmt(md.last_response))
    : ("Mediator 불통 · 마지막 응답 " + (md.last_response?fmt(md.last_response):"없음") + " · drain 통보가 늦어진다");
}

function load(){ fetch("/api/state").then(function(r){return r.json();}).then(render).catch(function(e){ alert("상태를 못 불러왔습니다: "+e); }); }
function post(p){ return fetch(p,{method:"POST"}).then(function(r){return r.json();}); }

function doDrain(){
  var mode="graceful", r=document.querySelector("input[name=mode]:checked");
  if(r) mode=r.value;
  post("/api/drain?mode="+encodeURIComponent(mode)).then(function(){ load(); });
}
function doUndrain(){ post("/api/undrain").then(function(){ load(); }); }

function stopMsg(res){
  if(res.cancelled) return "도는 작업을 취소하고 노드를 껐습니다.";
  if(res.mediator_reachable===false) return "노드를 껐습니다. Mediator 에 못 닿아 그 작업은 임대 만료로 죽고, 소유자가 껐다는 기록이 남지 않습니다.";
  return "노드를 껐습니다.";
}
function doStop(){
  if(!confirm("이 노드를 정지합니다. 도는 작업이 있으면 먼저 cancel 해서 소유자가 껐다는 기록이 남게 합니다. Mediator 에 못 닿으면 임대 만료로 죽습니다. 계속할까요?")) return;
  post("/api/stop").then(function(res){ alert(stopMsg(res)); load(); });
}
function doStart(){ post("/api/start").then(function(res){ alert(res.running?"노드가 떴습니다.":"노드가 안 떴습니다. 로그를 확인하세요."); load(); }); }
function doRestart(){
  if(!confirm("재시작: 도는 작업을 먼저 cancel 하고 정지한 뒤 다시 띄웁니다. 계속할까요?")) return;
  post("/api/stop").then(function(){ return post("/api/start"); }).then(function(res){ alert(res.running?"다시 떴습니다.":"안 떴습니다. 로그를 확인하세요."); load(); });
}

function showLogs(){
  var pre=el("logs"); pre.style.display="block";
  fetch("/api/logs").then(function(r){return r.text();}).then(function(t){ pre.textContent=t; });
}

load();
setInterval(load, 5000);
</script>
</body>
</html>
`
