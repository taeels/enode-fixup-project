import test from 'node:test';
import assert from 'node:assert/strict';
import { GalleryDemo, GALLERY_KEY } from '../static/demo/gallery.mjs';
const proof = '04995a39-1fba-4c16-b2cd-dc80ff21257e';
const id = 'gallery-' + 'a'.repeat(64);
const response = (body, status = 200) => ({ ok: status >= 200 && status < 300, status, json: async () => body });
const draft = { outcome: 'draft_ready', message: '초안입니다.', body: '좋은 프로젝트!', transcript: [{ role: 'assistant', text: '초안입니다.' }] };
function create(fetcher, saved) {
  const values = new Map(saved ? [[GALLERY_KEY, JSON.stringify(saved)]] : []);
  const storage = { getItem: k => values.get(k), setItem: (k, v) => values.set(k, v), removeItem: k => values.delete(k) };
  const timers = { setTimeout: () => 1, clearTimeout: () => {} };
  const demo = new GalleryDemo({ submitter: '밝은 수달', storage, uuid: () => proof, fetcher, timers });
  demo.state.projects = [{ id: 'project', title: 'Project', teamName: 'Team' }]; return { demo, values, storage };
}
test('one prompt submits one server workflow and displays the actual posted result', async () => {
  const calls = [];
  const { demo } = create(async (path, options) => {
    calls.push({ path, ...options });
    if (options.method === 'POST') return response({ run_id: id, state: 'RUNNING' }, 201);
    return response({ run: { run_id: id, state: 'SUCCEEDED' }, result: { outcome: 'posted', message: '게시 완료', project_id: 'project', body: '좋은 프로젝트!', transcript: [{ role: 'tool', tool: 'post_comment', text: '게시 완료' }] } });
  });
  await demo.start('Team 팀에 응원 댓글 달아줘');
  assert.equal(demo.state.phase, 'posted'); assert.equal(demo.state.result.project_id, 'project');
  assert.deepEqual(calls.filter(c => c.method === 'POST').map(c => c.path), ['/v1/demo/gallery/comments']);
  assert.deepEqual(JSON.parse(calls[0].body), { prompt: 'Team 팀에 응원 댓글 달아줘', request_id: proof, submitter: '밝은 수달' });
  assert.equal(demo.state.confirmedBody, null);
  assert.ok(calls.every(c => c.credentials === 'omit' && c.redirect === 'error'));
  assert.equal(calls[1].headers['X-Gallery-Request-ID'], proof);
});
test('explicit draft-only outcome and ambiguous team never cause a publication request', async () => {
  for (const outcome of ['draft_ready', 'needs_project']) {
    let writes = 0;
    const { demo } = create(async (path, options) => {
      if (options.method === 'POST') { writes++; return response({ run_id: id, state: 'RUNNING' }); }
      return response({ run: { run_id: id, state: 'SUCCEEDED' }, result: { ...draft, outcome } });
    });
    await demo.start('댓글 초안만 써줘'); assert.equal(demo.state.phase, outcome); assert.equal(writes, 1);
    demo.reset(); assert.equal(demo.state.phase, 'idle');
  }
});
test('refused request cannot publish', async () => {
  let posts = 0;
  const { demo } = create(async (path, options) => {
    if (options.method === 'POST') { posts++; return response({ run_id: id, state: 'RUNNING' }); }
    return response({ run: { run_id: id, state: 'SUCCEEDED' }, result: { outcome: 'refused', message: '외부 사이트 방문은 거절합니다.', transcript: [] } });
  });
  await demo.start('다른 사이트 방문해'); await demo.publish('댓글');
  assert.equal(demo.state.phase, 'refused'); assert.equal(posts, 1);
});
test('lost draft acknowledgement persists original UUID and retries same intent', async () => {
  const bodies = [];
  const { demo, values } = create(async (path, options) => { bodies.push(options.body); throw Error('offline'); });
  await demo.start('댓글'); await demo.retry();
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
test('blank prompt and duplicate click do not create requests', async () => {
  let count = 0;
  const { demo } = create(async () => { count++; throw Error('offline'); });
  await demo.start(' '); assert.equal(count, 0);
  await Promise.all([demo.start('comment'), demo.start('comment')]); assert.equal(count, 1);
});
test('stale callback after destruction cannot update UI', async () => {
  let resolve; const { demo } = create(() => new Promise(r => { resolve = r; }));
  const pending = demo.start('comment'); demo.destroy(); resolve(response({ run_id: id, state: 'RUNNING' })); await pending;
  assert.equal(demo.state.run, null);
});
