import test from 'node:test';
import assert from 'node:assert/strict';
import { GalleryHistory, GALLERY_HISTORY_KEY } from '../static/shared/gallery-history.mjs';
import { GalleryDemo, GALLERY_KEY } from '../static/demo/gallery.mjs';
const proof = '04995a39-1fba-4c16-b2cd-dc80ff21257e';
const id = 'gallery-' + 'a'.repeat(64), other = 'gallery-' + 'b'.repeat(64);
const result = { outcome: 'answered', message: 'Found the team', transcript: [{ role: 'assistant', text: 'Saved reply' }] };
const body = (runID = id) => ({ run: { run_id: runID, state: 'SUCCEEDED' }, result });
const response = (data = body(), status = 200) => ({ ok: status === 200, status, json: async () => data });
const timers = { setTimeout: () => 1, clearTimeout: () => {} };
function storage() {
  const values = new Map();
  return { getItem: k => values.get(k), setItem: (k, v) => values.set(k, v), removeItem: k => values.delete(k) };
}
test('accepted session migrates proof and history survives reset and browser reopen', async () => {
  const session = storage(), persistent = storage();
  session.setItem(GALLERY_KEY, JSON.stringify({ intent: { prompt: 'Old prompt', request_id: proof }, run: { run_id: id }, result }));
  const history = new GalleryHistory({ storage: persistent });
  const demo = new GalleryDemo({ storage: session, history, timers, fetcher: async () => response() });
  await demo.poll(); demo.reset();
  assert.equal(demo.state.intent, null);
  assert.deepEqual(JSON.parse(persistent.getItem(GALLERY_HISTORY_KEY)), [[id, proof]]);
  assert.ok(!persistent.getItem(GALLERY_HISTORY_KEY).includes('Old prompt'));
  const calls = [];
  const reopened = new GalleryHistory({ storage: persistent, timers, fetcher: async (path, options) => { calls.push({ path, ...options }); return response(); } });
  await reopened.read(id.replace('gallery-', 'gallery-post-'));
  assert.equal(reopened.state.data.result.transcript[0].text, 'Saved reply');
  assert.equal(calls[0].path, `/v1/demo/gallery/runs/${id}`);
  assert.equal(calls[0].headers['X-Gallery-Request-ID'], proof);
  assert.equal(calls[0].method, 'GET'); assert.equal(calls[0].body, undefined);
});
test('guest without proof never fetches someone else conversation', async () => {
  let calls = 0;
  const history = new GalleryHistory({ timers, fetcher: async () => { calls++; return response(); } });
  await history.read(id);
  assert.equal(history.state.phase, 'unavailable'); assert.equal(calls, 0);
});
test('operator reads existing runs without a browser proof and drops token on destroy', async () => {
  const calls = [];
  const history = new GalleryHistory({ token: 'fixture-token', timers, fetcher: async (path, options) => { calls.push({ path, ...options }); return response(); } });
  await history.read(id);
  assert.equal(calls[0].path, `/v1/demo/gallery/history/${id}`);
  assert.deepEqual(calls[0].headers, { Authorization: 'Bearer fixture-token' });
  assert.equal(calls[0].method, 'GET'); assert.equal(calls[0].cache, 'no-store'); assert.equal(calls[0].credentials, 'omit'); assert.equal(calls[0].redirect, 'error');
  history.destroy(); assert.equal(history.token, ''); assert.deepEqual(history.state, { phase: 'idle' });
});
test('a delayed previous selection or closed dialog cannot replace the current conversation', async () => {
  const pending = [];
  const history = new GalleryHistory({ token: 'fixture-token', timers, fetcher: () => new Promise(resolve => pending.push(resolve)) });
  const first = history.read(id), second = history.read(other);
  pending[1](response(body(other))); await second;
  pending[0](response()); await first;
  assert.equal(history.state.id, other);
  const third = history.read(id); history.cancel(); pending[2](response()); await third;
  assert.equal(history.state.phase, 'loading'); assert.equal(history.state.data, undefined);
});
test('malformed and unavailable responses do not show a different run or trigger a write', async () => {
  for (const [data, status, phase] of [[body(other), 200, 'error'], [{ ...body(), publication: { run: { run_id: other } } }, 200, 'error'], [null, 401, 'error'], [null, 404, 'error'], [null, 429, 'error'], [null, 503, 'error'], [{ run: { run_id: id, state: 'FAILED' }, result: null }, 200, 'ready'], [{ run: { run_id: id, state: 'RUNNING' }, result: null }, 200, 'ready']]) {
    const history = new GalleryHistory({ token: 'fixture-token', timers, fetcher: async (_, options) => { assert.equal(options.method, 'GET'); return response(data, status); } });
    await history.read(id); assert.equal(history.state.phase, phase); assert.ok(history.state.message);
  }
});
test('proof archive is bounded and merges another tab while ignoring malformed entries', () => {
  const persistent = storage();
  persistent.setItem(GALLERY_HISTORY_KEY, JSON.stringify([[null, proof], ['../../secret', proof], [id, 'invalid']]));
  const first = new GalleryHistory({ storage: persistent }), second = new GalleryHistory({ storage: persistent });
  first.remember(id, proof); second.remember(other, proof);
  first.restore(); assert.equal(first.proofs.size, 2);
  for (let n = 0; n < 201; n++) first.remember('gallery-' + n.toString(16).padStart(64, '0'), proof);
  assert.equal(first.proofs.size, 200);
  assert.equal(JSON.parse(persistent.getItem(GALLERY_HISTORY_KEY)).length, 200);
  const blocked = new GalleryHistory({ storage: { getItem() { throw Error('blocked'); }, setItem() { throw Error('blocked'); } } });
  blocked.remember(id, proof); assert.equal(blocked.proofs.get(id), proof);
});
