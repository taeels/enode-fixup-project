export const GALLERY_HISTORY_KEY = 'enode.demo.gallery.history.v1';
export const galleryParentID = id => typeof id === 'string' && /^gallery-(?:post-)?[0-9a-f]{64}$/.test(id) ? id.replace('gallery-post-', 'gallery-') : null;
const validProof = value => typeof value === 'string' && /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/.test(value);

// 대화 본문은 봉인 결과에서 읽는다. 브라우저에는 자기 요청의 조회 증명만 남긴다.
export class GalleryHistory {
  constructor({ storage, token = '', fetcher = (...args) => fetch(...args), timers = globalThis, onChange = () => {} } = {}) {
    Object.assign(this, { storage, token, fetcher, timers, onChange });
    this.proofs = new Map(); this.generation = 0; this.state = { phase: 'idle' };
    this.restore();
  }
  restore() {
    try {
      const entries = JSON.parse(this.storage?.getItem(GALLERY_HISTORY_KEY) || '[]');
      if (Array.isArray(entries)) for (const entry of entries.slice(-200)) {
        if (Array.isArray(entry) && entry[0] && galleryParentID(entry[0]) === entry[0] && validProof(entry[1])) this.proofs.set(entry[0], entry[1]);
      }
    } catch { /* Storage can be disabled or corrupt; current tab still works. */ }
    while (this.proofs.size > 200) this.proofs.delete(this.proofs.keys().next().value);
  }
  remember(id, proof) {
    id = galleryParentID(id);
    if (!id || !validProof(proof)) return;
    if (this.proofs.get(id) === proof) return;
    this.restore(); this.proofs.delete(id); this.proofs.set(id, proof);
    while (this.proofs.size > 200) this.proofs.delete(this.proofs.keys().next().value);
    try { this.storage?.setItem(GALLERY_HISTORY_KEY, JSON.stringify([...this.proofs])); } catch { /* Keep proof in memory. */ }
  }
  emit(state) { this.state = state; this.onChange(state); }
  cancel() { this.generation++; this.controller?.abort(); }
  async read(selectedID) {
    this.cancel(); this.restore();
    const generation = this.generation, id = galleryParentID(selectedID);
    if (!id) { this.emit({ phase: 'error', id: selectedID, message: '갤러리 작업이 아닙니다.' }); return; }
    const proof = this.proofs.get(id);
    if (!this.token && !proof) {
      this.emit({ phase: 'unavailable', id: selectedID, message: '이 브라우저에 이 요청의 조회 정보가 없습니다. 요청한 브라우저에서 열거나 실 함대 화면에 토큰으로 접속해 주세요.' }); return;
    }
    this.emit({ phase: 'loading', id: selectedID, message: '저장된 대화를 불러오고 있습니다…' });
    const controller = new AbortController(); this.controller = controller;
    const timeout = this.timers.setTimeout(() => controller.abort(), 15000);
    try {
      const response = await this.fetcher(`/v1/demo/gallery/${this.token ? 'history' : 'runs'}/${id}`, {
        method: 'GET', credentials: 'omit', redirect: 'error', cache: 'no-store', signal: controller.signal,
        headers: this.token ? { Authorization: `Bearer ${this.token}` } : { 'X-Gallery-Request-ID': proof },
      });
      if (!response.ok) throw new Error(response.status === 401 ? '인증이 만료되었습니다. 토큰으로 다시 접속해 주세요.' : response.status === 404 ? '대화 기록을 찾을 수 없거나 조회 권한이 없습니다.' : response.status === 429 ? '조회 요청이 많습니다. 잠시 후 다시 열어주세요.' : `기록을 불러오지 못했습니다 (${response.status}).`);
      const data = await response.json();
      if (generation !== this.generation) return;
      if (data.run?.run_id !== id || (data.publication && data.publication.run?.run_id !== id.replace('gallery-', 'gallery-post-'))) throw new Error('기록이 선택한 작업과 일치하지 않습니다.');
      const active = data.publication || data;
      const message = active.result?.message || (['FAILED', 'CANCELLED'].includes(active.run.state) ? `실행이 ${active.run.state} 상태로 끝났으며 저장된 대화가 없습니다.` : active.run.state === 'SUCCEEDED' ? '결과를 봉인하고 있습니다. 잠시 후 다시 조회해 주세요.' : '아직 실행 중입니다. 완료 후 저장된 대화를 확인할 수 있습니다.');
      this.emit({ phase: 'ready', id: selectedID, data, message });
    } catch (error) {
      if (generation === this.generation) this.emit({ phase: 'error', id: selectedID, message: controller.signal.aborted ? '조회 시간이 초과됐습니다. 다시 조회해 주세요.' : error.message });
    } finally { this.timers.clearTimeout(timeout); }
  }
  destroy() { this.cancel(); this.token = ''; this.proofs.clear(); this.state = { phase: 'idle' }; }
}
