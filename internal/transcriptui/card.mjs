// 트랜스크립트 카드를 그리는 한 벌. 제어판과 현황판이 같은 바이트를 로드한다.
//
// 앞 판은 이 함수들이 internal/panel/page.go 의 문자열 상수 안에 살았고,
// 현황판은 같은 규칙을 .mjs 로 한 번 더 지을 참이었다. 두 화면이 같은 모양을
// 두 언어로 두 번 짓는 것이 첫째 값이고, 제어판 JS 를 재는 하네스가 없어
// DOM 규칙 다섯(R14 · R19 · R20 · R21 · R30)의 유일한 검사가 사람이었던 것이
// 둘째다. 한 장으로 빼면 둘 다 닫힌다.
//
// 임포트가 0 이다. 두 서버가 각자의 라우트로 이 바이트를 내는데, 임포트를
// 하나라도 걸면 그 경로를 두 서버가 똑같이 낼 수 있어야 한다 - 잎이 아니면
// 접은 두 벌이 의존을 통해 뒤로 돌아온다.
//
// 부수효과도 0 이다. window 를 안 건드리고 시계도 네트워크도 안 읽는다
// (business-rules R37 · R38). 제어판은 인라인 모듈 한 줄로 window 에 걸고,
// 현황판은 DashboardView 에 주입으로 넘긴다.

// 그리는 종류는 파서의 일곱뿐이다. 모르는 type 은 파서가 raw 로 준다 (R41).
const KINDS = ['init', 'text', 'tool_use', 'tool_result', 'result', 'raw', 'capped'];

// ContractError 는 name 으로 갈린다. client.mjs 의 오류 사상이 그 이름을 읽고
// "잘못된 관측 JSON" 과 계약 위반을 가른다 - 같은 이름을 쓰는 이유다.
class ContractError extends Error {
  constructor(what) { super(`the event contract was broken: ${what}`); this.name = 'ContractError'; }
}
const check = (ok, what) => { if (!ok) throw new ContractError(what); };
const count = (v, what) => { check(Number.isInteger(v) && v >= 0, what); return v; };

// parseEvents 는 as=events 의 몸통(transcript.Result)을 검증한다.
//
// 이 검사가 이 저장소에 처음이다 - model.mjs 의 parseObservation 이 nodes ·
// runs · detail · asks 넷을 필드마다 재는데 사건 배열은 그 표에 없었다.
// 본문을 텍스트로만 그리는 것(R54)이 XSS 를 막지만 모양이 틀린 응답은 못 막고,
// 못 막으면 카드가 빈 채로 서서 "에이전트가 아무 말도 안 했다" 로 읽힌다.
//
// 모르는 키는 안 막는다. 파서가 나중에 필드를 더할 수 있고, 그때 화면이
// 통째로 거부하면 새 필드를 더하는 쪽이 화면을 먼저 고쳐야 한다.
export function parseEvents(body) {
  check(body !== null && typeof body === 'object' && !Array.isArray(body), 'the body is not an object');
  check(Array.isArray(body.events), 'events is not an array');
  for (const e of body.events) {
    check(e !== null && typeof e === 'object' && !Array.isArray(e), 'an event is not an object');
    check(KINDS.includes(e.kind), `unknown kind: ${String(e.kind)}`);
    count(e.line, 'line');
    for (const key of ['sub', 'text', 'name', 'id']) if (e[key] !== undefined) check(typeof e[key] === 'string', key);
    if (e.ok !== undefined) check(typeof e.ok === 'boolean', 'ok');
    if (e.shell !== undefined) check(typeof e.shell === 'boolean', 'shell');
    if (e.cut !== undefined) count(e.cut, 'cut');
    if (e.info !== undefined) check(e.info !== null && typeof e.info === 'object', 'info');
  }
  for (const key of ['raw', 'lines', 'head', 'partial']) if (body[key] !== undefined) count(body[key], key);
  return body;
}

// statusLine 은 카드의 상태 줄 한 줄을 낸다. 도는 단계가 아니면 null 이다.
//
// live 가 불리언인 것이 이 함수가 한 벌로 남는 조건이다. 현황판은
// step.state === 'CLAIMED' 를 넣고 제어판은 임대 여부를 넣는다 — 무엇이
// 「돈다」인지는 화면마다 다르지만 그 답은 양쪽 다 예/아니오다.
//
// 이 문이 없으면 끝난 단계에 짝 없는 tool_use 가 남았을 때 (도구가 도는
// 중에 단계가 죽으면 그렇다) 화면이 끝난 것을 도는 것으로 그린다.
export function statusLine(events, live) {
  if (!live) return null;
  const tool = activeTool(events);
  return tool ? `running ${tool.name}` : 'thinking';
}

// activeTool 은 마지막 tool_use 에 짝이 되는 tool_result 가 아직 없으면 그
// 도구를 낸다. 짝이 다 있으면 null 이다.
//
// 마지막 사건의 종류만 보면 안 된다 - 결과가 온 뒤에도 그 뒤에 text 가
// 붙으면 "쓰는 중" 이 그대로 남는다. 짝짓기는 파서가 이미 해 뒀다
// (Event.ID 가 tool_use_id 다).
//
// 이 판정이 도는 단계인지까지 보지는 않는다 - 그 문은 statusLine 이 진다
// (R56). 둘을 가른 이유는 짝짓기와 문이 다른 것을 재기 때문이다: 짝짓기는
// 사건 열만 보고, 문은 화면이 아는 것을 본다.
export function activeTool(events) {
  if (!Array.isArray(events)) return null;
  for (let i = events.length - 1; i >= 0; i--) {
    const e = events[i];
    if (e.kind !== 'tool_use') continue;
    if (!e.id) return { name: e.name || '' };
    for (let j = i + 1; j < events.length; j++) {
      if (events[j].kind === 'tool_result' && events[j].id === e.id) return null;
    }
    return { name: e.name || '' };
  }
  return null;
}

function label(e) {
  if (e.kind === 'text') return e.sub === 'thinking' ? 'Thinking' : 'Agent';
  if (e.kind === 'tool_use') return 'Tool';
  if (e.kind === 'tool_result') return e.ok === false ? 'Result (failed)' : 'Result';
  if (e.kind === 'init') return 'Start';
  if (e.kind === 'result') return 'End';
  if (e.kind === 'capped') return 'Capped';
  return 'raw';
}

// 인자의 기본 값이 먼저 선다. 도구마다 그 하나가 사람이 읽는 것이다 —
// Bash 는 command 이고 Read 는 file_path 다.
const PRIMARY = ['command', 'file_path', 'path', 'pattern', 'query', 'url', 'prompt'];

// 완결된 "키": 값 짝 하나. 잘린 JSON 에서도 앞쪽 짝들은 온전하다.
const PAIR = /"([A-Za-z_][\w.]*)"\s*:\s*("(?:[^"\\]|\\.)*"|-?\d+(?:\.\d+)?|true|false|null)/g;

function short(v) {
  const t = typeof v === 'string' ? v : JSON.stringify(v);
  return t.length > 80 ? t.slice(0, 80) + '…' : t;
}

function fromPairs(pairs) {
  const first = PRIMARY.find(k => k in pairs);
  const out = [];
  if (first) out.push(short(pairs[first]));
  for (const k of Object.keys(pairs)) if (k !== first) out.push(`${k}=${short(pairs[k])}`);
  return out.join(' · ');
}

// toolArgs 는 도구 인자를 사람이 읽는 한 줄로 만든다.
//
// 원문 JSON 을 그대로 그리면 중괄호와 따옴표가 화면의 절반을 먹는다. 그리고
// 파서가 인자를 표시 상한에서 자르므로 JSON.parse 가 대체로 실패한다 - 그래서
// 짝 단위로 훑는 길을 함께 둔다. 훑어서 아무것도 못 찾으면 원문 그대로다.
function toolArgs(text) {
  if (!text) return '';
  try {
    const o = JSON.parse(text);
    if (o && typeof o === 'object' && !Array.isArray(o)) return fromPairs(o);
  } catch { /* 잘린 JSON 이다. 아래에서 짝을 훑는다 */ }
  const pairs = {};
  for (const m of text.matchAll(PAIR)) {
    try { pairs[m[1]] = JSON.parse(m[2]); } catch { /* 이 짝은 건너뛴다 */ }
  }
  return Object.keys(pairs).length ? fromPairs(pairs) : text;
}

function summary(e) {
  if (e.kind === 'init') {
    const i = e.info || {};
    return [i.model, i.version, i.tools ? `${i.tools} tools` : ''].filter(Boolean).join(' · ');
  }
  if (e.kind === 'result') {
    const r = e.info || {};
    // 달러는 자리를 묶는다 - 부동소수가 그대로 나오면 $0.016518900000000003 이
    // 되어 카드 한 줄의 절반을 먹는다. 네 자리면 한 단계의 비용이 다 들어간다.
    const cost = r.cost_usd ? `$${Number(r.cost_usd).toFixed(4)}` : '';
    return [r.reason, r.turns ? `${r.turns} turns` : '', cost].filter(Boolean).join(' · ');
  }
  if (e.kind === 'capped') return `the progress file reached its limit (${(e.info || {}).bytes || 0} bytes)`;
  // raw 는 한 줄이다. 본문 JSON 을 여기 그리면 카드가 장부가 된다 - CB1 에서
  // 도는 동안 사건 29 중 16 이 raw 였다 (system/thinking_tokens 가 토큰
  // 델타마다 한 줄씩 온다). 버리지는 않는다. 줄 전체는 원문 토글이 낸다 (R40).
  if (e.kind === 'raw') return [e.sub, `${(e.text || '').length} bytes`].filter(Boolean).join(' · ');
  if (e.kind === 'tool_use') return toolArgs(e.text);
  return e.text || '';
}

// drawable 은 그릴 값이 있는 사건인가다 (ADR-071).
//
// 셋을 안 그린다. 버리는 것이 아니라 카드에 안 올리는 것이고, 줄 전체는
// 원문 토글이 언제나 낸다.
//
//   생각                  ADR-071 이 봉인에서 안 남기기로 한 것이다. 도는 동안에만
//                         보였다가 봉인 뒤 사라지면 같은 카드가 두 얼굴이 된다
//   본문 없는 말           생각만 한 턴이 남기는 빈 줄이다. 읽을 것이 0 인데 자리를 먹는다
//   system/thinking_tokens 토큰 델타마다 한 줄씩 온다. 세는 값이지 읽는 값이 아니다
function drawable(e) {
  if (e.kind === 'text') return e.sub !== 'thinking' && !!e.text;
  if (e.kind === 'raw') return e.sub !== 'system/thinking_tokens';
  return true;
}

// row 는 사건 하나를 줄 하나로 만든다.
//
// 본문은 textContent 로만 들어간다. innerHTML 에 하네스 바이트가 0 번 닿는다
// (R54 · SECURITY-05) - 제어판의 CSP 에 'unsafe-inline' 이 붙어 있으므로
// 현황판보다 제어판 쪽에서 이 줄이 더 유일한 방어다.
function row(e, open, onToggle) {
  const el = (tag, cls, text) => {
    const n = document.createElement(tag);
    if (cls) n.className = cls;
    if (text !== undefined) n.textContent = text;
    return n;
  };
  const line = el('div', 'ev' + (e.kind === 'raw' ? ' raw' : ''));
  line.append(el('span', 'k', label(e)));
  if (e.name) line.append(el('span', 'tool', e.name));

  // 도구 결과는 접어 둔다. 펼침의 열쇠가 tool_use_id 인 것이 값이다 - line
  // 번호는 파서가 창 안에서 1 부터 세므로 창이 밀리면 전부 밀린다 (R39).
  const foldable = e.kind === 'tool_result' && e.id;
  if (foldable) {
    const shown = open.has(e.id);
    const fold = el('span', 'fold', shown ? ' [fold]' : ' [unfold]');
    fold.addEventListener('click', () => onToggle(e.id, !shown));
    line.append(fold);
    if (!shown) return line;
  }
  line.append(el('div', 'body', summary(e)));
  if (e.cut) line.append(el('span', 'cut', `${e.cut} bytes were cut`));
  return line;
}

// renderEvents 는 사건 열을 container 안에 통째로 다시 짓는다.
//
// 인자가 사건 배열 하나인 것이 이 모듈이 한 벌로 남는 조건이다 (R36).
// 원문 문자열까지 받게 하면 함수 안에 갈림이 서고, 두 화면이 그 갈림의 다른
// 가지만 쓴다 - 한쪽 가지는 영영 안 밟힌다.
//
// open 은 펼친 tool_use_id 의 Set 이고 onToggle(id, next) 는 부르는 쪽이 자기
// 상태를 고쳐 다시 그린다. 이 둘이 인자인 이유는 앞 판에서 제어판의 전역
// txOpen 과 drawTranscript 에 직접 붙어 있었기 때문이다.
export function renderEvents(container, events, { open = new Set(), onToggle = () => {}, empty = 'nothing to read yet' } = {}) {
  // 바닥에 있었는지를 그리기 전에 잰다. 갈고 나서 재면 언제나 바닥이 아니다.
  // 위로 올려 읽는 중이면 따라가지 않는다 (R53 · U5 의 R21) - 앞 판은
  // 무조건 따라가서 도는 동안 스크롤백을 읽을 수 없었다.
  const stuck = container.scrollHeight - container.scrollTop - container.clientHeight < 24;
  const list = (Array.isArray(events) ? events : []).filter(drawable);
  const rows = list.map(e => row(e, open, onToggle));
  if (!rows.length) {
    const none = document.createElement('div');
    none.className = 'muted';
    none.textContent = empty;
    rows.push(none);
  }
  container.replaceChildren(...rows);
  if (stuck) container.scrollTop = container.scrollHeight;
}
