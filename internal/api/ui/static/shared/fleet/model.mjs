// obs 관측과 표시 모델. 합성 응답이나 배정 판단을 만들지 않는다.
export const COLORS = { idle: '#3FBF87', leased: '#4B9CFF', drain: '#B98CFF', asked: '#E2A33C', expiring: '#7A8698', failed: '#F0574B' };
export const RUN_STATES = ['QUEUED', 'RUNNING', 'VERIFYING', 'SUCCEEDED', 'FAILED'];
export class ContractError extends Error {
  constructor(field) { super(`Invalid observation: ${field}`); this.name = 'ContractError'; }
}
const check = (ok, field) => { if (!ok) throw new ContractError(field); };
const object = (v, field) => { check(v !== null && typeof v === 'object' && !Array.isArray(v), field); return v; };
const string = (v, field, empty = false) => check(typeof v === 'string' && (empty || v.length > 0), field);
const array = (v, field) => { check(Array.isArray(v), field); return v; };
const unique = (items, key, field) => check(new Set(items.map(key)).size === items.length, field);
const date = (v, field) => check(typeof v === 'string' && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/.test(v) && Number.isFinite(Date.parse(v)), field);
function attrs(v, field) {
  object(v, field);
  for (const value of Object.values(v)) string(value, field, true);
}
function assigned(v) {
  for (const role of array(v, 'assigned')) {
    object(role, 'assigned'); string(role.as, 'assigned.as');
    for (const n of array(role.nodes, 'assigned.nodes')) { object(n, 'assigned.node'); string(n.node, 'assigned.node'); string(n.label, 'assigned.label', true); }
  }
}
function verdict(v) {
  if (v == null) return;
  object(v, 'verdict'); string(v.state, 'verdict.state');
  for (const c of array(v.checks, 'verdict.checks')) {
    object(c, 'verdict.check'); check(typeof c.ok === 'boolean', 'verdict.check.ok');
    if (c.note !== undefined) string(c.note, 'verdict.check.note', true);
  }
}
export function parseObservation(kind, input, expectedID) {
  const v = structuredClone(object(input, kind));
  if (kind === 'nodes' || kind === 'runs') date(v.observed_at, 'observed_at');
  if (kind === 'nodes') {
    for (const n of array(v.nodes, 'nodes')) {
      object(n, 'node'); string(n.node_id, 'node_id'); string(n.label, 'label', true); string(n.instance, 'instance', true);
      date(n.seen_at, 'seen_at'); date(n.expires_at, 'expires_at');
      check(['', 'graceful', 'at-boundary'].includes(n.draining), 'draining');
      for (const c of array(n.capabilities, 'capabilities')) {
        object(c, 'capability');
        string(c.capability, 'capability');
        // contract.Capability의 nil 속성 맵은 null로 직렬화된다.
        if (c.attrs === null) c.attrs = {};
        attrs(c.attrs, 'capability.attrs');
      }
      check(Object.hasOwn(n, 'lease'), 'lease');
      if (n.lease !== null) { object(n.lease, 'lease'); string(n.lease.run_id, 'lease.run_id'); date(n.lease.not_after, 'not_after'); }
    }
    unique(v.nodes, n => n.node_id, 'duplicate node_id');
  } else if (kind === 'runs') {
    for (const r of array(v.runs, 'runs')) {
      object(r, 'run'); string(r.run_id, 'run_id'); string(r.state, 'state'); string(r.work_id, 'work_id', true);
      string(r.submitter, 'submitter', true); date(r.created_at, 'created_at');
      check(Object.hasOwn(r, 'ended_at') && Object.hasOwn(r, 'verdict'), 'run terminal fields');
      if (r.ended_at !== null) date(r.ended_at, 'ended_at');
      assigned(r.assigned); verdict(r.verdict);
    }
    unique(v.runs, r => r.run_id, 'duplicate run_id');
  } else if (kind === 'detail') {
    string(v.run_id, 'run_id'); check(v.run_id === expectedID, 'run_id mismatch'); string(v.state, 'state');
    if (v.warnings !== undefined) array(v.warnings, 'warnings').forEach(w => string(w, 'warning'));
    if (v.requires === undefined) check(v.warnings?.length > 0, 'requires');
    else {
      for (const r of array(v.requires, 'requires')) {
        object(r, 'requires');
        string(r.as, 'requires.as'); string(r.capability, 'requires.capability'); attrs(r.attrs, 'requires.attrs');
        if (r.count !== undefined) check(Number.isInteger(r.count) && r.count >= 0, 'requires.count');
      }
      unique(v.requires, r => r.as, 'duplicate requires.as');
    }
    if (v.assigned !== undefined) assigned(v.assigned);
    if (v.steps !== undefined) {
      for (const s of array(v.steps, 'steps')) {
        object(s, 'step');
        string(s.id, 'step.id'); check(Number.isInteger(s.seq) && s.seq > 0, 'step.seq');
        string(s.state, 'step.state'); string(s.uses, 'step.uses', true);
        check(typeof s.chosen === 'boolean', 'step.chosen');
        array(s.needs, 'step.needs').forEach(n => string(n, 'step.needs'));
        if (s.node !== undefined) string(s.node, 'step.node');
        if (s.attempt !== undefined) check(Number.isInteger(s.attempt) && s.attempt >= 0, 'step.attempt');
        for (const key of ['started_at', 'ended_at']) if (s[key] !== undefined) date(s[key], key);
      }
      try { graphLayout(v.steps); } catch (e) { v.graphError = e.message; }
    }
    verdict(v.verdict);
  } else if (kind === 'asks') {
    for (const a of array(v.asks, 'asks')) {
      object(a, 'ask'); string(a.run_id, 'ask.run_id'); check(Number.isInteger(a.seq) && a.seq > 0, 'ask.seq');
      string(a.prompt, 'ask.prompt', true);
      if (a.answerers !== undefined) array(a.answerers, 'answerers').forEach(x => string(x, 'answerer'));
      for (const key of ['asked_at', 'deadline']) if (a[key] != null) date(a[key], key);
    }
    unique(v.asks, a => JSON.stringify([a.run_id, a.seq]), 'duplicate ask');
  } else throw new ContractError('resource kind');
  return v;
}

// unit/ux-runtime@8985694의 level/lane 배치를 사용하되 순환/누락을 거부한다.
export function graphLayout(steps, iso = false) {
  unique(steps, s => s.id, 'duplicate step.id'); unique(steps, s => s.seq, 'duplicate step.seq');
  const byID = new Map(steps.map(s => [s.id, s])), levels = new Map(), indegree = new Map(), children = new Map();
  for (const s of steps) {
    unique(s.needs, x => x, 'duplicate needs'); indegree.set(s.id, s.needs.length); levels.set(s.id, 0);
    for (const id of s.needs) {
      check(byID.has(id), 'missing dependency');
      if (!children.has(id)) children.set(id, []);
      children.get(id).push(s.id);
    }
  }
  const queue = steps.filter(s => !s.needs.length).map(s => s.id);
  for (let i = 0; i < queue.length; i++) {
    const id = queue[i];
    for (const child of children.get(id) || []) {
      levels.set(child, Math.max(levels.get(child), levels.get(id) + 1));
      indegree.set(child, indegree.get(child) - 1);
      if (indegree.get(child) === 0) queue.push(child);
    }
  }
  check(queue.length === steps.length, 'dependency cycle');
  const lanes = new Map();
  return steps.map(s => {
    const col = levels.get(s.id), row = lanes.get(col) || 0; lanes.set(col, row + 1);
    return { ...s, x: 65 + col * (iso ? 235 : 240) + row * (iso ? 40 : 0), y: 70 + row * 180 + col * (iso ? 80 : 0) };
  });
}
export const seconds = (value, now) => Math.max(0, Math.ceil((Date.parse(value) - now) / 1000));
// Run의 관측 상태만 설명한다. 요청 전달률이나 Mediator의 건강 상태를 추정하지 않는다.
export function runFlowFacts(state, steps = [], stale = false) {
  const descriptions = {
    QUEUED: ['배정 대기', '실행 가능한 enode를 기다리고 있어요', 'asked'],
    RUNNING: ['작업 진행 중', '실행 단계의 상태를 확인하고 있어요', 'leased'],
    VERIFYING: ['결과 검증 중', '실행 결과를 확인하고 있어요', 'asked'],
    SUCCEEDED: ['작업 완료', '모든 실행 결과가 확인됐어요', 'idle'],
    FAILED: ['작업 실패', '단계와 결과에서 원인을 확인하세요', 'failed'],
  };
  let [label, description, tone] = descriptions[state] || ['상태 확인 중', '작업 관측을 기다리고 있어요', 'expiring'];
  if (state === 'RUNNING' && steps.some(s => s.state === 'ASKED')) {
    [label, description, tone] = ['사람 응답 대기', '질문에 답하면 실행이 이어져요', 'asked'];
  } else if (state === 'RUNNING' && steps.some(s => s.state === 'CLAIMED')) {
    [label, description] = ['실행 중', '배정된 enode에서 작업하고 있어요'];
  }
  const activeSteps = !stale && state === 'RUNNING' ? steps.filter(s => s.state === 'CLAIMED').map(s => s.id) : [];
  return { label, description, tone, activeSteps };
}
export function nodeFacts(node, details, asks, now) {
  const runID = node.lease?.run_id;
  const asked = !!runID && ((details.get(runID)?.steps || []).some(s => s.state === 'ASKED') || asks.some(a => a.run_id === runID));
  const expiring = seconds(node.expires_at, now) <= 30;
  return { asked, expiring, tone: asked ? 'asked' : node.draining ? 'drain' : node.lease ? 'leased' : 'idle',
    label: asked ? '사람을 기다림' : node.draining ? `draining · ${node.lease ? '진행 중' : '대기 중'}` : node.lease ? '임대 중' : '임대 없음' };
}
export function matches(node, requirement) {
  return node.capabilities.some(c => c.capability === requirement.capability && Object.entries(requirement.attrs).every(([k, v]) => c.attrs?.[k] === v));
}
export const sandboxValues = node => node.capabilities.map(c => ({ capability: c.capability, value: c.attrs?.sandbox ?? '미제공' }));
export function observationClock(resource, now) {
  if (!resource?.data) return null;
  const anchor = resource.data.observed_at ? Date.parse(resource.data.observed_at) : resource.clockAnchor;
  if (!Number.isFinite(anchor)) return null;
  const elapsed = Math.max(0, now - resource.receivedAt);
  return anchor + Math.min(elapsed, 15000, resource.frozenAt == null ? 15000 : Math.max(0, resource.frozenAt - resource.receivedAt));
}
export function dependentClock(resource, dependencies, now) {
  const clocks = [resource, ...dependencies].map(r => observationClock(r, now));
  return clocks.some(c => c === null) ? null : Math.min(...clocks);
}
export function isStale(resource, now) { return !!resource?.data && (resource.failures >= 3 || now - resource.receivedAt >= 15000); }
