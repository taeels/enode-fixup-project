import { requestID } from './submission.mjs';
export const GALLERY_KEY = 'enode.demo.gallery.v1';
export const GALLERY_ORIGIN = 'https://main.d3gkmtkue9o7ly.amplifyapp.com';
const terminal = state => ['SUCCEEDED', 'FAILED', 'CANCELLED'].includes(state);
const validID = value => typeof value === 'string' && /^gallery-[0-9a-f]{64}$/.test(value);
const validProof = value => typeof value === 'string' && /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(value);
export class GalleryDemo {
  constructor({ submitter, storage, fetcher = (...args) => fetch(...args), uuid = requestID, timers = globalThis, onChange = () => {}, onAccepted = () => {} }) {
    Object.assign(this, { submitter, storage, fetcher, uuid, timers, onChange, onAccepted });
    this.state = { phase: 'idle', projects: [], message: '', intent: null, run: null, result: null, publication: null, confirmedBody: null };
    this.editBody = ''; this.generation = 0;
    try {
      const saved = JSON.parse(storage?.getItem(GALLERY_KEY) || 'null');
      if (saved && validProof(saved.intent?.request_id) && (!saved.run || validID(saved.run.run_id))) {
        Object.assign(this.state, saved, { projects: [], phase: 'uncertain', message: '이전 요청을 같은 ID로 확인할 수 있습니다.' });
        this.editBody = saved.result?.body || '';
      }
    } catch { /* A corrupt session never authorizes a write. */ }
  }
  emit() { this.onChange(this.state); }
  save() {
    const { projects, message, ...saved } = this.state;
    try { this.storage?.setItem(GALLERY_KEY, JSON.stringify(saved)); } catch { /* Keep current intent in memory. */ }
  }
  async request(path, body, proof) {
    const controller = new AbortController(), timeout = this.timers.setTimeout(() => controller.abort(), 15000);
    try {
      const response = await this.fetcher('/v1/demo/gallery' + path, { method: body ? 'POST' : 'GET', credentials: 'omit', redirect: 'error', cache: 'no-store', signal: controller.signal,
        headers: { ...(body ? { 'Content-Type': 'application/json' } : {}), ...(proof ? { 'X-Gallery-Request-ID': proof } : {}) }, ...(body ? { body: JSON.stringify(body) } : {}) });
      if (!response.ok) { const error = new Error(response.status === 429 ? '요청이 많습니다. 잠시 후 같은 요청으로 확인해 주세요.' : response.status === 404 ? '갤러리 데모가 준비되지 않았거나 요청을 찾지 못했습니다.' : `결과를 확인하지 못했습니다 (${response.status}).`); error.status = response.status; throw error; }
      return await response.json();
    } finally { this.timers.clearTimeout(timeout); }
  }
  async catalog() {
    try { const data = await this.request('/projects'); if (!Array.isArray(data.projects)) throw new Error('프로젝트 목록이 올바르지 않습니다.'); this.state.projects = data.projects; this.state.message = ''; }
    catch (error) { this.state.message = error.message; }
    this.emit();
  }
  async start(prompt) {
    if (this.state.intent || !prompt.trim() || [...prompt].length > 1000) return;
    const proof = this.uuid(); if (!validProof(proof)) return;
    this.state.intent = { prompt, submitter: this.submitter, request_id: proof }; this.save();
    await this.send(false);
  }
  async send(publish) {
    if (this.state.phase === 'sending') return;
    const generation = this.generation;
    this.state.phase = 'sending'; this.state.message = publish ? '댓글을 게시하고 있습니다…' : 'Claude에게 메시지를 전달하고 있습니다…'; this.save(); this.emit();
    try {
      const run = await this.request(publish ? `/runs/${this.state.run.run_id}/publish` : this.state.intent.project_id ? '/runs' : '/comments', publish ? { request_id: this.state.intent.request_id, body: this.state.confirmedBody } : this.state.intent);
      if (generation !== this.generation) return;
      if (!(publish ? /^gallery-post-[0-9a-f]{64}$/ : /^gallery-[0-9a-f]{64}$/).test(run?.run_id) || typeof run.state !== 'string') throw new Error('접수 응답을 확인하지 못했습니다.');
      if (publish) this.state.publication = { run, result: null }; else this.state.run = run;
      this.onAccepted(run); this.save(); await this.poll();
    } catch (error) {
      if (generation !== this.generation) return;
      this.state.phase = 'uncertain'; this.state.message = error.message + ' 이미 접수됐을 수 있습니다.'; this.save(); this.emit();
    }
  }
  async poll() {
    if (!this.state.run || this.stopped) return;
    const generation = this.generation;
    try {
      const data = await this.request(`/runs/${this.state.run.run_id}`, null, this.state.intent.request_id);
      if (generation !== this.generation) return;
      if (data.run?.run_id !== this.state.run.run_id) throw new Error('결과가 요청과 일치하지 않습니다.');
      Object.assign(this.state, { run: data.run, result: data.result, publication: data.publication || null });
      const active = data.publication || data;
      if (active.result) {
        this.state.phase = active.result.outcome; this.state.message = active.result.message;
        if (this.state.phase === 'draft_ready') this.editBody = active.result.body;
      } else if (terminal(active.run.state) && active.run.state !== 'SUCCEEDED') {
        this.state.phase = 'error'; this.state.message = `실행이 ${active.run.state} 상태로 끝났습니다.`;
      } else {
        this.state.phase = 'running'; this.state.message = active.run.state === 'QUEUED' ? '메시지를 받았습니다. 앞선 작업이 끝나면 이어서 진행합니다.' : 'Claude가 요청을 처리하고 있습니다…';
        this.timer = this.timers.setTimeout(() => this.poll(), 3000);
      }
      this.save(); this.emit();
    } catch (error) {
      if (generation !== this.generation) return;
      this.state.phase = 'uncertain'; this.state.message = error.message; this.save(); this.emit();
    }
  }
  publish(body) {
    if (this.state.phase !== 'draft_ready' || this.state.confirmedBody !== null || !body.trim() || [...body].length > 500) return;
    this.state.confirmedBody = body; this.save(); return this.send(true);
  }
  retry() {
    if (['sending', 'running'].includes(this.state.phase)) return;
    if (this.state.confirmedBody !== null && !this.state.publication) return this.send(true);
    if (this.state.run) return this.poll();
    if (this.state.intent) return this.send(false);
    return this.catalog();
  }
  reset() {
    if (['sending', 'running', 'uncertain'].includes(this.state.phase)) return;
    this.generation++; this.timers.clearTimeout(this.timer);
    const projects = this.state.projects;
    this.state = { phase: 'idle', projects, message: '', intent: null, run: null, result: null, publication: null, confirmedBody: null }; this.editBody = '';
    try { this.storage?.removeItem(GALLERY_KEY); } catch { /* Memory reset is sufficient in this tab. */ }
    this.emit();
  }
  destroy() { this.stopped = true; this.generation++; this.timers.clearTimeout(this.timer); }
}
