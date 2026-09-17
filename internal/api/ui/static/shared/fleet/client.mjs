import { parseObservation, observationClock } from './model.mjs';

export function retryDelay(value, wallNow = Date.now()) {
  if (value !== null && /^\d+(?:\.\d+)?$/.test(value.trim())) return Math.max(0, Number(value) * 1000);
  const date = Date.parse(value); return Number.isFinite(date) ? Math.max(0, date - wallNow) : 1000;
}

// 아래 셋은 이 파일 밖에서도 쓴다 — 트랜스크립트 카드의 폴러가 같은 규율을
// 진다 (business-rules R50). 카드는 타이머가 다르고(2초) 자원도 다르지만
// 429 · 제한시간 · 연속 실패의 계산은 같아야 한다. 두 벌로 두면 한쪽만
// 고쳐지고, 그때 갈리는 것은 "Mediator 에 못 닿을 때 화면이 무엇을 하나" 다.
export const REQUEST_TIMEOUT_MS = 4000;
export const FREEZE_AFTER = 3;
const TIMEOUT_MESSAGE = '조회 제한시간 초과';

// timeLimit 은 제한시간 하나를 걸고 만료되면 controller 를 중단한다.
//
// timedOut 을 객체 필드로 두는 것이 값이다 — 거절된 뒤 catch 에서 "중단된
// 것인가 제한시간인가" 를 다시 읽어야 하는데, 값으로 넘기면 그 시점의
// 복사본이라 언제나 거짓이다.
export function timeLimit(controller, timers = globalThis, ms = REQUEST_TIMEOUT_MS) {
  const limit = { timedOut: false, timer: null };
  limit.promise = new Promise((_, reject) => {
    limit.timer = timers.setTimeout(() => { limit.timedOut = true; controller.abort(); reject(new Error(TIMEOUT_MESSAGE)); }, ms);
  });
  limit.clear = () => timers.clearTimeout(limit.timer);
  return limit;
}

// describeFailure 는 사람이 읽을 한 줄을 고른다. 계약 위반과 잘못된 JSON 과
// 네트워크 실패가 화면에서 안 합쳐진다.
export function describeFailure(error, timedOut) {
  if (timedOut) return TIMEOUT_MESSAGE;
  if (error?.name === 'ContractError') return error.message;
  if (error instanceof SyntaxError) return '잘못된 관측 JSON';
  const message = error?.message;
  if (message?.startsWith('조회') || message?.startsWith('관측') || message?.startsWith('공개') || message?.startsWith('사건')) return message;
  return '네트워크 조회 실패';
}
export class ObservationClient {
  constructor({ mode, token = '', fetcher = (...args) => fetch(...args), now = () => performance.now(), wallNow = () => Date.now(), timers = globalThis, onChange = () => {}, onUnauthorized = () => {} }) {
    this.mode = mode; this.token = mode === 'fleet' ? token : ''; this.fetcher = fetcher; this.now = now; this.wallNow = wallNow; this.timers = timers;
    this.onChange = onChange; this.onUnauthorized = onUnauthorized; this.resources = new Map(); this.generation = 0; this.retryAt = 0; this.selectedRun = null; this.stopped = false;
    for (const key of mode === 'fleet' ? ['nodes', 'runs', 'asks'] : ['nodes', 'runs']) this.add(key);
  }
  add(key) { if (!this.resources.has(key)) this.resources.set(key, { data: null, failures: 0, error: '', receivedAt: null, pending: false }); return this.resources.get(key); }
  emit() { if (!this.stopped) this.onChange(this.resources, this.now()); }
  syncDetails() {
    const ids = new Set((this.resources.get('nodes')?.data?.nodes || []).flatMap(n => n.lease ? [n.lease.run_id] : []));
    if (this.selectedRun) ids.add(this.selectedRun);
    for (const [key, r] of this.resources) if (key.startsWith('detail:') && !ids.has(key.slice(7))) { r.controller?.abort(); this.resources.delete(key); }
    for (const id of ids) this.add(`detail:${id}`);
  }
  selectRun(id) { this.selectedRun = id; this.syncDetails(); this.emit(); if (id) return this.request(`detail:${id}`); }
  async request(key) {
    const r = this.resources.get(key);
    if (this.stopped || !r || r.pending || this.now() < this.retryAt) return;
    const generation = this.generation, controller = new AbortController(); r.controller = controller; r.pending = true;
    const current = () => !this.stopped && generation === this.generation && this.resources.get(key) === r && r.controller === controller;
    const kind = key.startsWith('detail:') ? 'detail' : key, id = kind === 'detail' ? key.slice(7) : null;
    const path = kind === 'detail' ? `/v1/runs/${encodeURIComponent(id)}` : `/v1/${key}`;
    const limit = timeLimit(controller, this.timers);
    try {
      // 제한시간은 헤더뿐 아니라 응답 본문을 읽는 시간까지 포함한다.
      const work = async () => {
        const response = await this.fetcher(path, { method: 'GET', headers: this.mode === 'fleet' ? { Authorization: `Bearer ${this.token}` } : {}, credentials: 'omit', cache: 'no-store', redirect: 'error', signal: controller.signal });
        if (!current() || limit.timedOut) return null;
        if (response.status === 401 && this.mode === 'fleet') { this.stop(); this.onUnauthorized(); return null; }
        if (response.status === 429) { if (this.mode === 'demo') this.retryAt = Math.max(this.retryAt, this.now() + retryDelay(response.headers.get('Retry-After'), this.wallNow())); throw new Error('조회 한도 초과 · 잠시 후 다시 조회'); }
        if (!response.ok) throw new Error(response.status === 404 ? '관측 대상이 없습니다 (404)' : response.status === 401 ? '공개 데모 조회가 열려 있지 않습니다 (401)' : `조회 실패 (${response.status})`);
        return parseObservation(kind, await response.json(), id);
      };
      const data = await Promise.race([work(), limit.promise]);
      if (!current() || !data) return;
      if (kind === 'detail') {
        if (r.data?.requires !== undefined) r.previousRequires = r.data.requires;
        if (data.requires !== undefined) r.previousRequires = data.requires;
      }
      const receivedAt = this.now();
      const anchorResource = ['nodes', 'runs'].map(k => this.resources.get(k)).filter(r => r?.data?.observed_at).sort((a, b) => b.receivedAt - a.receivedAt)[0];
      Object.assign(r, { data, failures: 0, error: '', receivedAt, frozenAt: null, clockAnchor: observationClock(anchorResource, receivedAt) });
      if (kind === 'nodes') this.syncDetails();
    } catch (error) {
      if (!current() || (controller.signal.aborted && !limit.timedOut)) return;
      r.failures++; r.error = describeFailure(error, limit.timedOut);
      if (r.failures >= FREEZE_AFTER && r.frozenAt == null) r.frozenAt = this.now();
    } finally {
      limit.clear();
      if (current()) { r.pending = false; r.controller = null; this.emit(); }
    }
  }
  async refresh(onlyFailed = false) {
    if (this.stopped) return;
    const keys = [...this.resources].filter(([, r]) => !onlyFailed || r.error || !r.data || this.now() - r.receivedAt >= 15000).map(([k]) => k);
    await Promise.allSettled(keys.map(key => this.request(key)));
    // nodes 응답으로 새로 발견한 현재 임대 상세만 가져온다.
    await Promise.allSettled([...this.resources].filter(([k, r]) => k.startsWith('detail:') && !r.data && !r.error && !keys.includes(k)).map(([k]) => this.request(k)));
  }
  start() { this.interval = this.timers.setInterval(() => this.refresh(), 5000); return this.refresh(); }
  stop() {
    this.stopped = true; this.generation++; this.token = ''; this.timers.clearInterval(this.interval);
    for (const r of this.resources.values()) r.controller?.abort();
    this.resources.clear();
  }
}
