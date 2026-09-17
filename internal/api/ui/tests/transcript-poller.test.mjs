import test from 'node:test';
import assert from 'node:assert/strict';
import { TranscriptPoller, CARD_INTERVAL_MS } from '../static/shared/fleet/transcript-poller.mjs';
import * as card from '../../../transcriptui/card.mjs';

// 폴러는 네트워크도 시계도 안 든다 - 전부 주입이다. 그래서 이 파일이
// 가짜 타이머와 가짜 fetch 로 돌고, 재는 것이 「몇 초 뒤에 무엇을 부르나」가
// 아니라 「무엇을 부르고 언제 멈추나」다.
function harness({ responses = [], steps = [], run = 'run-1' } = {}) {
  const calls = [], timers = { handles: [] };
  timers.setInterval = () => 1;
  timers.clearInterval = () => {};
  timers.setTimeout = fn => { timers.handles.push(fn); return timers.handles.length; };
  timers.clearTimeout = () => {};
  let clock = 0;
  const fetcher = async url => {
    calls.push(url);
    const next = responses.length > 1 ? responses.shift() : responses[0];
    return next ?? reply({});
  };
  const poller = new TranscriptPoller({ render: card, token: 't', fetcher, timers, now: () => clock, wallNow: () => clock });
  poller.track(run, steps);
  return { poller, calls, tick: () => poller.tick(), advance: ms => { clock += ms; } };
}

function reply({ status = 200, bytes = '0', attempt = '0', source = 'progress', capped = null, events = [], raw = '' }) {
  const headers = new Map([['X-Enode-Log-Bytes', bytes], ['X-Enode-Log-Attempt', attempt], ['X-Enode-Log-Source', source]]);
  if (capped) headers.set('X-Enode-Log-Capped', capped);
  return { status, ok: status >= 200 && status < 300, headers: { get: k => headers.get(k) ?? null }, json: async () => ({ events, raw: 0, lines: events.length, head: 0, partial: 0 }), text: async () => raw };
}

const ev = (kind, rest = {}) => ({ kind, line: 1, ...rest });
const claimed = (seq, name) => ({ seq, name, state: 'CLAIMED' });

test('the card timer is its own, two seconds, not the five of the observation list', () => {
  assert.equal(CARD_INTERVAL_MS, 2000);
});

test('the log URL carries the step id as name and never an offset', async () => {
  const h = harness({ steps: [claimed(1, 'summarize')], responses: [reply({ bytes: '6785' })] });
  await h.tick();
  assert.equal(h.calls.length, 1);
  assert.ok(h.calls[0].includes('/v1/runs/run-1/steps/1/log'), h.calls[0]);
  assert.ok(h.calls[0].includes('name=summarize'), 'the step id is the log name');
  assert.ok(h.calls[0].includes('as=events'));
  // from= 은 경계를 가로지르는 줄을 양쪽에서 다 떨어뜨린다. 그 줄이 사라진
  // 것을 화면이 알 길이 없어서 안 쓴다.
  assert.ok(!h.calls[0].includes('from='), 'the window is never cut');
});

test('a step nobody said had finished keeps being polled', async () => {
  // CB4 가 재는 자리다 - 목록 폴링을 막으면 상태가 CLAIMED 에 멈춰 있고
  // 카드는 그대로 자라야 한다. 폴러가 스스로 끝을 판정하면 그 조각이
  // 초록인 이유가 우연이 된다.
  //
  // 바퀴가 다섯인 것이 이 시험의 값이다. 셋으로 재 봤더니 「세 번 안 움직이면
  // 끝」이라는 변이가 살아남았다 - 첫 바퀴는 총 길이가 null 에서 움직인
  // 것으로 세므로 그 규칙이 발동하는 것은 넷째부터다. 변이가 살아남으면
  // 시험이 없는 것이다.
  const h = harness({ steps: [claimed(1, 'a')], responses: [reply({ bytes: '10' })] });
  for (let i = 0; i < 5; i++) await h.tick();
  assert.equal(h.calls.length, 5, 'a still total is not an ending');
});

test('a finished step is read once and a sealed answer stops the polling', async () => {
  const done = harness({ steps: [{ seq: 1, name: 'a', state: 'DONE' }], responses: [reply({ bytes: '99' })] });
  await done.tick(); await done.tick(); await done.tick();
  assert.equal(done.calls.length, 1, 'a finished step is read exactly once');

  const sealed = harness({ steps: [claimed(1, 'a')], responses: [reply({ bytes: '99', source: 'sealed' })] });
  await sealed.tick(); await sealed.tick();
  assert.equal(sealed.calls.length, 1, 'a sealed file cannot grow');
  assert.equal(sealed.poller.card(1).source, 'sealed');
});

test('only the running step is polled when several steps are on the card list', async () => {
  const h = harness({ steps: [{ seq: 1, name: 'a', state: 'DONE' }, claimed(2, 'b')], responses: [reply({ bytes: '1' })] });
  await h.tick();
  assert.equal(h.calls.length, 2, 'the finished step takes its single read');
  h.calls.length = 0;
  await h.tick();
  assert.deepEqual(h.calls.map(u => u.includes('steps/2/')), [true], 'from then on only the running step');
});

test('no selected run means no requests at all', async () => {
  const h = harness({ steps: [claimed(1, 'a')] });
  h.poller.track(null, []);
  await h.tick();
  assert.equal(h.calls.length, 0);
});

test('the last update time moves only when the total moves', async () => {
  const h = harness({ steps: [claimed(1, 'a')], responses: [reply({ bytes: '10' }), reply({ bytes: '10' }), reply({ bytes: '40' })] });
  await h.tick();
  const first = h.poller.card(1).changedAt;
  h.advance(5000); await h.tick();
  assert.equal(h.poller.card(1).changedAt, first, 'silence does not count as an update');
  h.advance(5000); await h.tick();
  assert.equal(h.poller.card(1).changedAt, 10000, 'new bytes move the clock');
});

test('a changed attempt drops what the card held instead of splicing two files', async () => {
  const h = harness({ steps: [claimed(1, 'a')], responses: [
    reply({ bytes: '40', attempt: '0', events: [ev('text', { text: '첫 시도' })] }),
    reply({ bytes: '5', attempt: '1', events: [ev('text', { text: '재시도' })] }),
  ] });
  await h.tick();
  assert.equal(h.poller.card(1).events.length, 1);
  await h.tick();
  const c = h.poller.card(1);
  assert.equal(c.attempt, 1);
  assert.equal(c.total, 5);
  assert.equal(c.events[0].text, '재시도', 'the earlier attempt is not carried over');
});

test('a rate limit pushes the next round by the shared calculation', async () => {
  const limited = reply({ status: 429 });
  limited.headers = { get: k => (k === 'Retry-After' ? '3' : null) };
  const h = harness({ steps: [claimed(1, 'a')], responses: [limited, reply({ bytes: '1' })] });
  await h.tick();
  assert.equal(h.poller.card(1).error, '조회 한도 초과 · 잠시 후 다시 조회');
  await h.tick();
  assert.equal(h.calls.length, 1, 'the poller waits out Retry-After');
  h.advance(3000); await h.tick();
  assert.equal(h.calls.length, 2, 'and comes back after it');
});

test('a failed round keeps the last value and says the value is old', async () => {
  const h = harness({ steps: [claimed(1, 'a')], responses: [
    reply({ bytes: '40', events: [ev('text', { text: '읽은 것' })] }),
    reply({ status: 503 }),
  ] });
  await h.tick();
  await h.tick();
  const c = h.poller.card(1);
  // 빈 화면으로 떨어뜨리면 「단계가 끝났다」로 읽힌다. 실제로는 중앙에 못
  // 닿은 것이고, 그 둘이 화면에서 같은 모양이면 안 된다.
  assert.equal(c.events.length, 1, 'the last value is not wiped');
  assert.equal(c.total, 40);
  assert.ok(c.error.length > 0, 'and the card says it is stale');
});

test('a missing run folds the card instead of showing an empty one', async () => {
  const h = harness({ steps: [claimed(1, 'a')], responses: [reply({ status: 404 })] });
  await h.tick(); await h.tick();
  assert.equal(h.poller.card(1).missing, true);
  assert.equal(h.calls.length, 1, 'a 404 is not retried forever');
});

test('the raw toggle costs a second query only while it is open', async () => {
  const h = harness({ steps: [claimed(1, 'a')], responses: [reply({ bytes: '3', raw: '{"type":"system"}' })] });
  await h.tick();
  assert.equal(h.calls.length, 1, 'closed, the card carries no raw bytes');
  h.poller.setRaw(1, true);
  await h.tick();
  assert.equal(h.calls.length, 3, 'open, each round fetches both representations');
  assert.ok(h.calls[2].includes('as=raw'));
  assert.equal(h.poller.card(1).raw, '{"type":"system"}');
  h.poller.setRaw(1, false);
  assert.equal(h.poller.card(1).raw, '', 'closing drops the bytes nobody is reading');
});
