import { RUN_STATES } from '../shared/fleet/model.mjs';
import { retryDelay } from '../shared/fleet/client.mjs';
import { element, button, dismissOnBackdrop } from '../shared/fleet/view.mjs';
import { trapFocus } from './tour.mjs';
export const PENDING_KEY = 'enode.demo.pending.v1';
export const SCENARIOS = ['led-toggle', 'welcome-audio'];
const validName = name => typeof name === 'string' && /^[가-힣]{1,12} [가-힣]{1,12}$/.test(name);
const legacyName = name => typeof name === 'string' && /^guest-[a-z]{1,24}-[a-z]{1,24}$/.test(name);
const validID = id => typeof id === 'string' && /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(id);
// LAN의 HTTP 데모에서는 randomUUID가 없을 수 있다. CSPRNG의 동일한 v4 형식을 쓴다.
export function requestID(cryptography = globalThis.crypto) {
  if (typeof cryptography?.randomUUID === 'function') return cryptography.randomUUID();
  const bytes = cryptography.getRandomValues(new Uint8Array(16));
  bytes[6] = (bytes[6] & 15) | 64; bytes[8] = (bytes[8] & 63) | 128;
  const hex = [...bytes].map(b => b.toString(16).padStart(2, '0')).join('');
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}
export class Submission {
  constructor({ submitter, storage, fetcher = (...args) => fetch(...args), uuid = requestID, now = () => Date.now(), timers = globalThis, onChange = () => {}, onAccepted = () => {}, onRefresh = () => {} }) {
    this.submitter = submitter; this.storage = storage; this.fetcher = fetcher; this.uuid = uuid; this.now = now; this.timers = timers;
    this.onChange = onChange; this.onAccepted = onAccepted; this.onRefresh = onRefresh; this.generation = 0;
    this.state = { phase: 'idle', intent: null, message: '', retryAt: 0, run: null };
    try {
      const saved = JSON.parse(storage?.getItem(PENDING_KEY) || 'null');
      if (saved && SCENARIOS.includes(saved.scenario_id) && (validName(saved.submitter) || legacyName(saved.submitter)) && validID(saved.request_id) && Number.isFinite(saved.started_at) && saved.started_at >= 0) {
        this.state = { ...this.state, phase: 'uncertain', intent: { scenario_id: saved.scenario_id, submitter: saved.submitter, request_id: saved.request_id }, startedAt: saved.started_at, message: '이전에 보낸 요청의 접수 결과가 미확인입니다. 같은 요청으로 다시 확인할 수 있습니다.' };
      } else if (saved) this.clearSaved();
    } catch { this.clearSaved(); }
  }
  emit() { this.onChange(this.state); }
  clearSaved() { try { this.storage?.removeItem(PENDING_KEY); } catch { /* 현재 세션의 메모리 상태는 유지한다. */ } }
  canStart() { return !['pending', 'uncertain', 'limited'].includes(this.state.phase); }
  start(scenario) {
    if (!this.canStart() || !SCENARIOS.includes(scenario)) return;
    if (!validName(this.submitter)) { this.state.message = 'Guest 이름을 확인할 수 없습니다. 페이지를 새로 열어 주세요.'; this.emit(); return; }
    let requestID; try { requestID = this.uuid(); } catch { /* 안전한 ID가 없으면 제출하지 않는다. */ }
    if (!validID(requestID)) { this.state.message = '요청 ID를 만들 수 없습니다. 브라우저 지원을 확인하세요.'; this.emit(); return; }
    this.state = { phase: 'idle', intent: { scenario_id: scenario, submitter: this.submitter, request_id: requestID }, startedAt: this.now(), retryAt: 0, message: '', run: null };
    return this.send();
  }
  newIntent() {
    if (this.state.phase === 'pending') return;
    this.generation++; this.controller?.abort(); this.clearSaved();
    this.state = { phase: 'idle', intent: null, retryAt: 0, run: null, message: '앞선 요청은 이미 접수됐을 수 있습니다. 다음 선택은 별도의 새 작업입니다.' }; this.emit();
  }
  retry() { if (['uncertain', 'limited'].includes(this.state.phase) && this.now() >= this.state.retryAt) return this.send(); }
  async send() {
    if (!this.state.intent || this.state.phase === 'pending') return;
    const generation = ++this.generation, intent = { ...this.state.intent }, controller = new AbortController(); this.controller = controller;
    this.state.phase = 'pending'; this.state.message = '접수 확인 중… 창을 닫아도 요청은 계속됩니다.';
    try { this.storage?.setItem(PENDING_KEY, JSON.stringify({ ...intent, started_at: this.state.startedAt })); } catch { /* 저장 불가여도 현재 의도의 ID는 메모리에 유지한다. */ }
    this.emit(); let timeout;
    const timeLimit = new Promise((_, reject) => { timeout = this.timers.setTimeout(() => { controller.abort(); reject(new Error('Submission timed out')); }, 10000); });
    try {
      const work = async () => {
        const response = await this.fetcher('/v1/demo/runs', { method: 'POST', credentials: 'omit', redirect: 'error', cache: 'no-store', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(intent), signal: controller.signal });
        const body = [200, 201, 202].includes(response.status) ? await response.json() : null;
        return { response, body };
      };
      const { response, body } = await Promise.race([work(), timeLimit]);
      if (generation !== this.generation) return;
      const status = response.status;
      if ([200, 201, 202].includes(status)) {
        if (!body || typeof body.run_id !== 'string' || !body.run_id || !RUN_STATES.includes(body.state)) throw new Error('Invalid submission acknowledgement');
        this.state.phase = 'accepted'; this.state.run = { run_id: body.run_id, state: body.state }; this.clearSaved();
        this.state.message = `${body.run_id} · ${body.state}${body.state === 'QUEUED' ? ' · 접수됨, 배정을 기다리는 중' : body.state === 'FAILED' ? ' · 기존 요청이 실패 상태입니다' : ' · 접수 확인됨'}`;
        this.emit(); this.onAccepted(this.state.run); return;
      }
      if ([400, 413, 415, 422].includes(status)) { this.state.phase = 'rejected'; this.state.message = `요청이 거절되었습니다 (${status}).${status === 422 ? ' 관측 목록을 다시 확인합니다.' : ' 새 작업으로 다시 시작할 수 있습니다.'}`; this.clearSaved(); if (status === 422) this.onRefresh(); }
      else if (status === 404) { this.state.phase = 'unavailable'; this.state.message = '데모 제출이 아직 제공되지 않습니다 (404).'; this.clearSaved(); }
      else if (status === 409) { this.state.phase = 'conflict'; this.state.message = '기존 요청과 충돌했습니다 (409). 대기 접수로 확인된 응답이 아닙니다.'; this.clearSaved(); }
      else if (status === 429) { this.state.phase = 'limited'; this.state.retryAt = this.now() + retryDelay(response.headers.get('Retry-After'), this.now()); this.state.message = '제출 한도에 도달했습니다. 대기 후 같은 요청으로 다시 시도하세요.'; }
      else throw new Error('Unconfirmed submission');
    } catch {
      if (generation !== this.generation) return;
      this.state.phase = 'uncertain'; this.state.message = '접수 결과를 확인하지 못했습니다. 이미 접수됐을 수 있으므로 같은 요청으로 다시 확인하세요.';
    } finally { this.timers.clearTimeout(timeout); if (generation === this.generation) this.emit(); }
  }
  destroy() { this.generation++; this.controller?.abort(); }
}
export class SubmissionDialog {
  constructor(root, { submission, canOpen = () => true, onGallery }) {
    this.submission = submission; this.canOpen = canOpen;
    this.dialog = element('dialog', 'task-dialog'); this.dialog.setAttribute('aria-labelledby', 'demo-task-title');
    const title = element('h2', '', '새 작업'); title.id = 'demo-task-title';
    const close = button('×', 'demo-task-close-button', () => this.close(), 'task-close'); close.setAttribute('aria-label', '새 작업 닫기');
    this.intro = element('p', 'muted');
    this.led = button('LED Toggle', 'demo-task-led-button', () => submission.start('led-toggle'), 'scenario-button'); this.led.append(element('span', '', '보드의 LED 시나리오 요청'));
    this.audio = button('사운드 재생', 'demo-task-audio-button', () => submission.start('welcome-audio'), 'scenario-button'); this.audio.append(element('span', '', '음원 시나리오 요청'));
    const scenarios = element('div', 'scenario-choices'); scenarios.append(this.led, this.audio);
    if (onGallery) { const gallery = button('해커톤에 의견 남기기', 'demo-task-gallery-button', () => { this.close(); onGallery(); }, 'scenario-button gallery-card'); gallery.append(element('span', '', '격리된 Claude와 댓글을 작성하고 참가팀에 기여하세요')); scenarios.append(gallery); }
    this.status = element('p', 'task-status'); this.status.setAttribute('role', 'status');
    this.retry = button('같은 요청으로 다시 확인', 'demo-task-retry-button', () => submission.retry(), 'primary');
    this.newRequest = button('다른 작업 시작', 'demo-task-new-intent-button', () => submission.newIntent());
    this.warning = element('p', 'muted', '다른 작업을 시작하면 별도의 요청이 됩니다. 앞선 요청은 이미 접수됐을 수 있습니다.');
    const actions = element('div', 'dialog-actions'); actions.append(this.newRequest, this.retry);
    this.dialog.append(close, title, this.intro, scenarios, this.status, this.warning, actions); root.append(this.dialog);
    this.dialog.addEventListener('cancel', e => { e.preventDefault(); this.close(); }); this.dialog.addEventListener('keydown', e => trapFocus(this.dialog, e));
    dismissOnBackdrop(this.dialog, this.dialog, () => this.close());
    this.timer = setInterval(() => this.update(submission.state), 250); this.update(submission.state);
  }
  get open() { return this.dialog.open; }
  show() { if (!this.canOpen() || this.open) return; this.returnFocus = document.activeElement; this.update(this.submission.state); this.dialog.showModal(); this.dialog.querySelector('[data-testid="demo-task-close-button"]').focus(); }
  close() { this.dialog.close(); if (this.returnFocus?.isConnected) this.returnFocus.focus({ preventScroll: true }); }
  update(state) {
    this.intro.textContent = state.intent && state.intent.submitter !== this.submission.submitter && !this.submission.canStart() ? '이전 요청의 제출자 정보로 다시 확인합니다.' : `${this.submission.submitter} 이름으로 요청합니다. 시나리오를 누르면 바로 제출됩니다.`;
    this.led.disabled = this.audio.disabled = !this.submission.canStart();
    const unresolved = ['uncertain', 'limited'].includes(state.phase);
    this.retry.hidden = this.newRequest.hidden = this.warning.hidden = !unresolved;
    this.retry.disabled = this.submission.now() < state.retryAt;
    const wait = this.retry.disabled ? ` (${Math.ceil((state.retryAt - this.submission.now()) / 1000)}초)` : '';
    const text = state.message + wait; if (this.status.textContent !== text) this.status.textContent = text;
  }
  destroy() { clearInterval(this.timer); this.dialog.remove(); }
}
