import test from 'node:test';
import { machineKind } from '../static/shared/fleet/scene.mjs';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { parseObservation, graphLayout, nodeFacts, matches, sandboxValues, observationClock, dependentClock, isStale } from '../static/shared/fleet/model.mjs';
const fixture = async name => JSON.parse(await readFile(new URL(`../testdata/obs-contract/${name}.json`, import.meta.url)));
test('the merged observation contract accepts nested requirements and explicit null leases', async () => {
  const nodes = parseObservation('nodes', await fixture('nodes-states')).nodes;
  const run = parseObservation('detail', await fixture('run-queued-nested'), 'run-queued');
  assert.equal(nodes.length, 6); assert.equal(nodes.filter(n => matches(n, run.requires[0])).length, 1);
});
test('wrong wire types cannot be silently normalized into successful observations', async () => {
  for (const name of ['nodes-invalid-draining', 'nodes-lease-omitted']) {
    const data = await fixture(name); assert.throws(() => parseObservation('nodes', data));
  }
  const flat = await fixture('run-queued-flat'); assert.throws(() => parseObservation('detail', flat, flat.run_id));
  const data = await fixture('run-branch'); delete data.steps[0].chosen;
  assert.throws(() => parseObservation('detail', data, data.run_id));
  const rows = await fixture('runs-states'); delete rows.runs[0].submitter;
  assert.throws(() => parseObservation('runs', rows));
});
test('a partial detail keeps Run state and warnings without inventing requirements', async () => {
  const data = await fixture('run-requires-unavailable');
  const result = parseObservation('detail', data, data.run_id);
  assert.equal(result.state, 'RUNNING'); assert.equal(result.requires, undefined); assert.equal(result.warnings.length, 1);
});
test('ASKED survives an expired lease and remains independent of draining and expiry', async () => {
  const { nodes } = await fixture('nodes-states'), detail = await fixture('run-asked');
  const node = nodes.find(n => n.lease?.run_id === detail.run_id); node.draining = 'graceful';
  const facts = nodeFacts(node, new Map([[detail.run_id, detail]]), [], Date.parse('2026-09-08T12:00:00Z'));
  assert.equal(facts.asked, true); assert.equal(facts.tone, 'asked');
  assert.equal(sandboxValues(node)[0].value, '미제공');
});
test('graph layout preserves branches and refuses invented edges or cycles', async () => {
  const { steps } = await fixture('run-branch'); const layout = graphLayout(steps);
  assert.ok(layout[1].x > layout[0].x); assert.equal(layout[1].x, layout[2].x); assert.notEqual(layout[1].y, layout[2].y);
  for (const edit of [s => s[0].needs.push('missing'), s => s[0].needs.push(s[1].id), s => s.push(s[0])]) {
    const copy = structuredClone(steps); edit(copy); assert.throws(() => graphLayout(copy));
  }
});
test('clock freezes at freshness limit without inventing a third failed request', () => {
  const resource = { data: { observed_at: '2026-09-08T12:00:00Z' }, receivedAt: 100, failures: 2 };
  assert.equal(observationClock(resource, 20100), Date.parse(resource.data.observed_at) + 15000);
  assert.equal(isStale(resource, 15100), true); assert.equal(resource.failures, 2);
});

test('device geometry uses explicit advertisement attributes, never label or harness', () => {
  assert.equal(machineKind({ label: 'board-01-workstation', capabilities: [{ capability: 'agent.reason', attrs: { harness: 'claude' } }] }), 'generic');
  assert.equal(machineKind({ capabilities: [{ attrs: { board: 'stm32' } }] }), 'generic');
  assert.equal(machineKind({ capabilities: [{ attrs: { board: 'stm32', os: 'windows' } }] }), 'workstation');
  assert.equal(machineKind({ capabilities: [{ attrs: { device_type: 'board' } }] }), 'board');
  assert.equal(machineKind({ capabilities: [{ attrs: { device_type: 'workstation' } }] }), 'workstation');
});


test('related clocks stay frozen while an unrelated observation continues refreshing', () => {
  const base = Date.parse('2026-09-08T12:00:00Z');
  const detail = { data: { state: 'RUNNING' }, clockAnchor: base, receivedAt: 0, failures: 2 };
  const nodes = { data: { observed_at: '2026-09-08T12:00:20Z' }, receivedAt: 20000, failures: 0 };
  assert.equal(dependentClock(nodes, [detail], 20000), base + 15000);
  detail.failures = 3; detail.frozenAt = 19000;
  assert.equal(dependentClock(nodes, [detail], 22000), base + 15000);
  assert.equal(dependentClock(nodes, [{ data: null }], 22000), null);
});

// 관측은 통째로 검사하므로 Run 하나가 틀리면 목록 전체가 안 뜬다.
// 그때 어느 Run 인지를 말해야 한다 - 안 말하면 보는 사람이 화면으로 못 좁히고
// 서버 응답을 직접 떠서 뒤져야 한다. 실제로 그랬다 (mac-shot-20260901 의
// verdict.checks 가 null 이었고, 화면은 "Invalid observation: verdict.checks"
// 한 줄만 냈다).
test('a bad verdict names the run it came from', () => {
  const runs = {
    observed_at: '2026-09-16T12:00:00Z',
    runs: [
      { run_id: 'good-1', state: 'SUCCEEDED', work_id: '', submitter: '',
        created_at: '2026-09-16T11:00:00Z', ended_at: '2026-09-16T11:30:00Z',
        assigned: [], verdict: { state: 'SUCCEEDED', checks: [] } },
      { run_id: 'mac-shot-20260901', state: 'SUCCEEDED', work_id: '', submitter: '',
        created_at: '2026-09-16T11:00:00Z', ended_at: '2026-09-16T11:30:00Z',
        assigned: [], verdict: { state: 'SUCCEEDED', checks: null } },
    ],
  };
  assert.throws(() => parseObservation('runs', runs), /verdict\.checks \(run mac-shot-20260901\)/);
});

// 빈 배열은 틀린 것이 아니다 - 판정 조건이 0 개인 계약이 그 모양이다.
test('an empty checks array is valid', () => {
  const runs = {
    observed_at: '2026-09-16T12:00:00Z',
    runs: [{ run_id: 'r1', state: 'SUCCEEDED', work_id: '', submitter: '',
      created_at: '2026-09-16T11:00:00Z', ended_at: '2026-09-16T11:30:00Z',
      assigned: [], verdict: { state: 'SUCCEEDED', checks: [] } }],
  };
  assert.equal(parseObservation('runs', runs).runs.length, 1);
});
