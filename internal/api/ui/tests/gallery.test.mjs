import test from 'node:test';
import assert from 'node:assert/strict';
import { GalleryDemo, GALLERY_KEY } from '../static/demo/gallery.mjs';
const proof = '04995a39-1fba-4c16-b2cd-dc80ff21257e';
const id = 'gallery-' + 'a'.repeat(64), postID = 'gallery-post-' + 'a'.repeat(64);
const response = (body, status = 200) => ({ ok: status >= 200 && status < 300, status, json: async () => body });
const draft = { outcome: 'draft_ready', message: '초안입니다.', body: '좋은 프로젝트!', transcript: [{ role: 'assistant', text: '초안입니다.' }] };
function create(fetcher, saved) {
  const values = new Map(saved ? [[GALLERY_KEY, JSON.stringify(saved)]] : []);
  const storage = { getItem: k => values.get(k), setItem: (k, v) => values.set(k, v), removeItem: k => values.delete(k) };
  const timers = { setTimeout: () => 1, clearTimeout: () => {} };
  const demo = new GalleryDemo({ submitter: '밝은 수달', storage, uuid: () => proof, fetcher, timers });
  demo.state.projects = [{ id: 'project', title: 'Project', teamName: 'Team' }]; return { demo, values, storage };
}
test('draft does not publish until explicit confirmation; body is frozen across uncertain retry', async () => {
  const calls = []; let publishAttempts = 0, published = false;
  const { demo } = create(async (path, options) => {
    calls.push({ path, ...options });
    if (path.endsWith('/publish')) { publishAttempts++; if (publishAttempts === 1) throw Error('connection lost'); published = true; return response({ run_id: postID, state: 'RUNNING' }); }
    if (options.method === 'POST') return response({ run_id: id, state: 'RUNNING' }, 201);
    return response({ run: { run_id: id, state: 'SUCCEEDED' }, result: draft, ...(published ? { publication: { run: { run_id: postID, state: 'SUCCEEDED' }, result: { outcome: 'posted', message: '게시 완료' } } } : {}) });
  });
  await demo.start('project', '댓글 써줘');
  assert.equal(demo.state.phase, 'draft_ready'); assert.equal(publishAttempts, 0);
  await demo.publish('내가 수정한 댓글'); assert.equal(demo.state.phase, 'uncertain');
  await demo.publish('다른 댓글'); assert.equal(publishAttempts, 1);
  await demo.retry(); assert.equal(demo.state.phase, 'posted'); assert.equal(publishAttempts, 2);
  const posts = calls.filter(c => c.path.endsWith('/publish'));
  assert.deepEqual(posts.map(c => JSON.parse(c.body)), Array(2).fill({ request_id: proof, body: '내가 수정한 댓글' }));
  assert.ok(calls.every(c => c.credentials === 'omit' && c.redirect === 'error'));
  assert.equal(calls.find(c => c.method === 'GET').headers['X-Gallery-Request-ID'], proof);
});
test('refused request cannot publish', async () => {
  let posts = 0;
  const { demo } = create(async (path, options) => {
    if (options.method === 'POST') { posts++; return response({ run_id: id, state: 'RUNNING' }); }
    return response({ run: { run_id: id, state: 'SUCCEEDED' }, result: { outcome: 'refused', message: '외부 사이트 방문은 거절합니다.', transcript: [] } });
  });
  await demo.start('project', '다른 사이트 방문해'); await demo.publish('댓글');
  assert.equal(demo.state.phase, 'refused'); assert.equal(posts, 1);
});
test('lost draft acknowledgement persists original UUID and retries same intent', async () => {
  const bodies = [];
  const { demo, values } = create(async (path, options) => { bodies.push(options.body); throw Error('offline'); });
  await demo.start('project', '댓글'); await demo.retry();
  assert.equal(bodies.length, 2); assert.equal(bodies[0], bodies[1]);
  assert.equal(JSON.parse(values.get(GALLERY_KEY)).intent.request_id, proof);
  demo.reset(); assert.equal(demo.state.phase, 'uncertain');
});
test('reload only reads accepted request; never automatically posts', async () => {
  const intent = { project_id: 'project', prompt: '댓글', request_id: proof, submitter: '밝은 수달' };
  const methods = [];
  const { demo } = create(async (path, options) => { methods.push(options.method); return response({ run: { run_id: id, state: 'QUEUED' }, result: null }); }, { intent, run: { run_id: id, state: 'RUNNING' }, confirmedBody: null });
  assert.deepEqual(methods, []); await demo.retry(); assert.deepEqual(methods, ['GET']); assert.equal(demo.state.phase, 'running');
});
test('invalid selection and duplicate click do not create requests', async () => {
  let count = 0;
  const { demo } = create(async () => { count++; throw Error('offline'); });
  await demo.start('https://example.com', 'comment'); assert.equal(count, 0);
  await Promise.all([demo.start('project', 'comment'), demo.start('project', 'comment')]); assert.equal(count, 1);
});
test('stale callback after destruction cannot update UI', async () => {
  let resolve; const { demo } = create(() => new Promise(r => { resolve = r; }));
  const pending = demo.start('project', 'comment'); demo.destroy(); resolve(response({ run_id: id, state: 'RUNNING' })); await pending;
  assert.equal(demo.state.run, null);
});
