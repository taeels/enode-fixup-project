import test from 'node:test';
import assert from 'node:assert/strict';
import { renderEvents, activeTool, statusLine, parseEvents } from '../../../transcriptui/card.mjs';

// 이 파일이 카드 규칙의 첫 자동 검사다. 앞 판에서 그리는 함수들은
// internal/panel/page.go 의 문자열 상수 안에 살았고 - 임포트가 안 되므로 -
// R19 · R20 · R21 · R30 의 유일한 검사가 사람이었다 (CB1). U8 이 그 함수들을
// internal/transcriptui 의 .mjs 한 장으로 빼면서 이 하네스 안으로 들어온다.
//
// DOM 은 여기서 가짜다. 노드에 브라우저가 없고, 이 저장소에 DOM 을 재는
// 하네스가 0 이라 만들어 쓴다. 가짜라서 못 재는 것을 이름으로 적는다 -
// 실제 배치 · 실제 스크롤 · CSS 는 CB4 와 CB1 이 사람 눈으로 잰다.
// 여기서 재는 것은 규칙이다: 무엇을 그리나, 무엇을 안 그리나, 어디에 넣나.
function element(tag) {
  const node = {
    tag, className: '', children: [], listeners: {},
    scrollTop: 0, scrollHeight: 0, clientHeight: 0,
    append(...kids) { node.children.push(...kids); },
    replaceChildren(...kids) { node.children = [...kids]; },
    addEventListener(type, fn) { (node.listeners[type] ??= []).push(fn); },
    click() { for (const fn of node.listeners.click || []) fn(); },
  };
  let text = '';
  Object.defineProperty(node, 'textContent', {
    get: () => (node.children.length ? node.children.map(k => k.textContent).join('') : text),
    set: v => { text = String(v); node.children = []; },
  });
  // innerHTML 에 하네스 바이트가 0 번 닿는다 (R54 · SECURITY-05). 대입이
  // 일어나면 그 자리에서 터뜨린다 - 「안 썼다」를 사람이 읽어서 확인하는
  // 것과 시험이 재는 것은 다르다.
  Object.defineProperty(node, 'innerHTML', {
    get: () => { throw new Error('innerHTML must not be read by the card renderer'); },
    set: () => { throw new Error('innerHTML must not be written by the card renderer'); },
  });
  return node;
}
globalThis.document = { createElement: element };

const ev = (kind, rest = {}) => ({ kind, line: 1, ...rest });
const text = node => node.children.map(k => k.textContent).join(' | ');

test('every parser kind is drawn and an unknown type never reaches the card', () => {
  const box = element('div');
  renderEvents(box, [
    ev('init', { info: { model: 'claude', version: '2.1', tools: 7 } }),
    ev('text', { text: '읽는 중' }),
    ev('tool_use', { name: 'Read', id: 't1', text: '{"path":"a"}' }),
    ev('tool_result', { name: 'Read', id: 't1', text: '결과 본문' }),
    ev('result', { info: { reason: 'success', turns: 3, cost_usd: 0.12 } }),
    ev('capped', { info: { bytes: 10485760 } }),
    ev('raw', { sub: 'system', text: '{"type":"system"}' }),
  ], {});
  assert.equal(box.children.length, 7);
  const drawn = text(box);
  for (const want of ['Start', 'Agent', 'Tool', 'Result', 'End', 'Capped', 'raw']) assert.ok(drawn.includes(want), want);
  assert.ok(drawn.includes('claude'), 'init carries its values');
  assert.ok(drawn.includes('3 turns'), 'result carries its values');
});

// 읽을 값이 없는 셋은 카드에 안 오른다 (ADR-071).
//
// 지우는 것이 아니라 안 그리는 것이다 - 원문 토글이 그 줄을 언제나 낸다.
// 사건 배열을 그대로 두고 그리는 자리에서만 거르는 것이 그 뜻이다.
test('what has nothing to read does not take a row', () => {
  const box = element('div');
  renderEvents(box, [
    ev('text', { sub: 'thinking', text: '생각한 것' }),
    ev('text', { text: '' }),
    ev('raw', { sub: 'system/thinking_tokens', text: '{"estimated_tokens":50}' }),
    ev('text', { text: '남는 줄' }),
  ], {});
  assert.equal(box.children.length, 1, 'only the line with something to read is drawn');
  const drawn = text(box);
  assert.ok(drawn.includes('남는 줄'), 'the readable line survived');
  assert.ok(!drawn.includes('생각한 것'), 'thinking is not drawn');
  assert.ok(!drawn.includes('thinking_tokens'), 'the token counter is not drawn');
});

// 도구 인자는 JSON 원문이 아니라 사람이 읽는 한 줄이다.
test('tool arguments are read as values, not as json', () => {
  const box = element('div');
  renderEvents(box, [ev('tool_use', { name: 'Bash', text: '{"command":"date","description":"print it"}' })], {});
  const drawn = text(box);
  assert.ok(drawn.includes('date'), 'the primary value stands first');
  assert.ok(drawn.includes('description=print it'), 'the rest keeps its name');
  assert.ok(!drawn.includes('{"'), 'no json punctuation reaches the card');
});

// 잘린 인자도 읽힌다 - 파서가 표시 상한에서 자르므로 이쪽이 흔한 경로다.
test('an argument cut in the middle still reads as values', () => {
  const box = element('div');
  renderEvents(box, [ev('tool_use', { name: 'Write', text: '{"file_path":"/tmp/x.json","content":"{\\"a' })], {});
  const drawn = text(box);
  assert.ok(drawn.includes('/tmp/x.json'), 'the complete pair was read');
  assert.ok(!drawn.includes('{"file_path"'), 'the raw json did not fall through');
});

test('raw stays one line and its body never reaches the card', () => {
  const box = element('div');
  const body = '{"type":"system","subtype":"thinking_tokens","delta":{"thinking":17}}';
  renderEvents(box, [ev('raw', { sub: 'system', text: body })], {});
  const drawn = text(box);
  assert.ok(drawn.includes(`${body.length} bytes`), 'the byte count stands in for the body');
  assert.ok(!drawn.includes('thinking_tokens'), 'the raw body is not drawn');
});

test('tool results fold on tool_use_id, not on the line number', () => {
  const box = element('div');
  const events = [ev('tool_result', { name: 'Read', id: 't1', text: '결과 본문' })];
  renderEvents(box, events, { open: new Set() });
  assert.ok(!text(box).includes('결과 본문'), 'a folded result hides its body');
  renderEvents(box, events, { open: new Set(['t1']) });
  assert.ok(text(box).includes('결과 본문'), 'the open key is the tool_use_id');

  // 줄 번호가 밀려도 펼침이 따라 밀리면 안 된다. 창이 앞에서 잘리면 파서가
  // 창 안에서 1 부터 다시 세므로 line 은 같은 사건을 가리키지 않는다.
  const moved = [ev('tool_result', { name: 'Read', id: 't1', text: '결과 본문', line: 41 })];
  renderEvents(box, moved, { open: new Set(['t1']) });
  assert.ok(text(box).includes('결과 본문'), 'a shifted line number does not lose the open state');
});

test('toggling a fold reports the identifier and the next state to the caller', () => {
  const box = element('div');
  const seen = [];
  renderEvents(box, [ev('tool_result', { name: 'Read', id: 't1', text: '본문' })], { open: new Set(), onToggle: (id, on) => seen.push([id, on]) });
  box.children[0].children.find(k => k.className === 'fold').click();
  assert.deepEqual(seen, [['t1', true]]);
});

test('the view follows the bottom only when it was already at the bottom', () => {
  const box = element('div');
  box.scrollHeight = 900; box.clientHeight = 300; box.scrollTop = 600; // 바닥이다
  renderEvents(box, [ev('text', { text: 'a' })], {});
  assert.equal(box.scrollTop, 900, 'a reader at the bottom keeps following');

  // 위로 올려 읽는 중. 앞 판은 무조건 따라가서 도는 동안 스크롤백을 읽을 수
  // 없었고, CB1 이 그 자리를 잡았다.
  box.scrollHeight = 900; box.clientHeight = 300; box.scrollTop = 10;
  renderEvents(box, [ev('text', { text: 'a' })], {});
  assert.equal(box.scrollTop, 10, 'a reader scrolled up is not dragged down');
});

test('an empty window says so instead of leaving the card blank', () => {
  const box = element('div');
  renderEvents(box, [], {});
  assert.equal(box.children.length, 1);
  assert.equal(box.children[0].className, 'muted');
});

test('the active tool is the last unmatched tool_use, whatever follows it', () => {
  assert.equal(activeTool([ev('text', { text: 'a' })]), null);
  assert.deepEqual(activeTool([ev('tool_use', { name: 'Read', id: 't1' })]), { name: 'Read' });
  assert.equal(activeTool([ev('tool_use', { name: 'Read', id: 't1' }), ev('tool_result', { id: 't1' })]), null);
  // 마지막 사건의 종류만 보면 여기서 틀린다 - 결과가 온 뒤에 말이 붙었다.
  assert.equal(activeTool([ev('tool_use', { name: 'Read', id: 't1' }), ev('tool_result', { id: 't1' }), ev('text', { text: '다 읽었다' })]), null);
  assert.deepEqual(activeTool([ev('tool_use', { name: 'Read', id: 't1' }), ev('tool_result', { id: 't1' }), ev('tool_use', { name: 'Write', id: 't2' })]), { name: 'Write' });
});

test('a wrongly shaped events body is refused instead of drawn half way', () => {
  assert.doesNotThrow(() => parseEvents({ events: [], raw: 0, lines: 0, head: 0, partial: 0 }));
  for (const bad of [null, [], { events: {} }, { events: [{ kind: 'nope', line: 1 }] }, { events: [{ kind: 'text', line: -1 }] }, { events: [{ kind: 'text', line: 1, text: 7 }] }, { events: [], raw: 'many' }]) {
    assert.throws(() => parseEvents(bad), e => e.name === 'ContractError', JSON.stringify(bad));
  }
  // 모르는 키는 안 막는다 - 파서가 필드를 더할 때 화면이 먼저 고쳐져야 하면
  // 그 순서가 뒤집힌다.
  assert.doesNotThrow(() => parseEvents({ events: [{ kind: 'text', line: 1, text: 'a', futureField: 1 }] }));
});

test('the status line has a gate: a step that is not running gets no line', () => {
  const running = [ev('tool_use', { name: 'Read', id: 't1' })];
  assert.equal(statusLine(running, true), 'running Read');
  assert.equal(statusLine([ev('text', { text: 'a' })], true), 'thinking');
  // 도구가 도는 중에 단계가 죽으면 짝 없는 tool_use 가 남는다. 문이 없으면
  // 화면이 끝난 것을 도는 것으로 그린다.
  assert.equal(statusLine(running, false), null);
  assert.equal(statusLine([], false), null);
});
