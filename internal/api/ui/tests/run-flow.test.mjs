import test from 'node:test';
import assert from 'node:assert/strict';
import { runFlowFacts } from '../static/shared/fleet/model.mjs';

test('terminal and unknown run states never animate residual claimed steps', () => {
  const steps = [{ id: 'worker', state: 'CLAIMED' }];
  for (const state of ['QUEUED', 'VERIFYING', 'SUCCEEDED', 'FAILED', 'FUTURE_STATE', undefined]) {
    assert.deepEqual(runFlowFacts(state, steps).activeSteps, [], state);
  }
  assert.equal(steps[0].state, 'CLAIMED');
});

test('only fresh RUNNING observations animate the claimed execution branches', () => {
  const steps = [{ id: 'finished', state: 'DONE' }, { id: 'worker', state: 'CLAIMED' }, { id: 'unchosen', state: 'SKIPPED' }];
  assert.deepEqual(runFlowFacts('RUNNING', steps).activeSteps, ['worker']);
  assert.deepEqual(runFlowFacts('RUNNING', steps, true).activeSteps, []);
  assert.equal(runFlowFacts('RUNNING', steps, true).label, runFlowFacts('RUNNING', steps).label);
});

test('human waiting remains distinct while an independent branch can still run', () => {
  const facts = runFlowFacts('RUNNING', [{ id: 'question', state: 'ASKED' }, { id: 'worker', state: 'CLAIMED' }]);
  assert.equal(facts.tone, 'asked');
  assert.deepEqual(facts.activeSteps, ['worker']);
  assert.deepEqual(runFlowFacts('RUNNING', [{ id: 'question', state: 'ASKED' }]).activeSteps, []);
});

test('empty, queued, verifying and failed observations keep distinct meanings', () => {
  const states = [undefined, 'QUEUED', 'RUNNING', 'VERIFYING', 'SUCCEEDED', 'FAILED'];
  const facts = states.map(state => runFlowFacts(state));
  assert.equal(new Set(facts.map(f => f.label)).size, states.length);
  for (const fact of facts) assert.deepEqual(fact.activeSteps, []);
  assert.equal(runFlowFacts('FAILED').tone, 'failed');
});
