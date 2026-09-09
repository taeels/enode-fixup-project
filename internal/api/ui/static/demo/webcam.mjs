import { element, button } from '../shared/fleet/view.mjs';
export function parseWebcamSettings(settings) {
  if (!settings || typeof settings !== 'object' || Array.isArray(settings) || !Object.hasOwn(settings, 'webcam')) throw new Error('Invalid webcam settings');
  if (settings.webcam === null) return null;
  const raw = settings.webcam?.embedUrl;
  if (typeof raw !== 'string' || !raw.startsWith('https://') || /[^\x21-\x7e]|\\|#|%(?![0-9a-f]{2})/i.test(raw) || raw.length > 4096) throw new Error('Invalid webcam URL');
  const url = new URL(raw), host = url.hostname;
  if ((url.port && Number(url.port) < 1) || url.search.includes(';')) throw new Error('Invalid webcam URL');
  if (url.protocol !== 'https:' || url.username || url.password || url.hash || !/^[a-z0-9]+(?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9]+(?:[a-z0-9-]*[a-z0-9])?)+$/.test(host) || /^\d+(?:\.\d+){3}$/.test(host) || /\.(?:localhost|local|internal)$/.test(host)) throw new Error('Invalid webcam URL');
  if ([...url.searchParams.keys()].some(k => /token|secret|password|authorization|signature|credential|api.?key|^sig$|^key$|^auth$/i.test(k))) throw new Error('Private webcam credentials are not allowed');
  return url;
}
export class WebcamState {
  constructor() { this.size = 360; this.zoom = 1; this.x = 0; this.y = 0; this.moving = false; this.maximum = 720; }
  bounds(width, height) { this.maximum = Math.max(1, Math.min(720, width - 48, height - 48)); this.resize(this.size); }
  resize(size) { this.size = Math.max(Math.min(200, this.maximum), Math.min(this.maximum, size)); }
  scale(zoom) { this.zoom = Math.max(1, Math.min(4, zoom)); this.pan(this.x, this.y); }
  pan(x, y) { const limit = (this.zoom - 1) / 2; this.x = Math.max(-limit, Math.min(limit, x)); this.y = Math.max(-limit, Math.min(limit, y)); }
  fit() { this.zoom = 1; this.x = this.y = 0; }
  minimap() { return { width: 1 / this.zoom, height: 1 / this.zoom, left: .5 - .5 / this.zoom - this.x / this.zoom, top: .5 - .5 / this.zoom - this.y / this.zoom }; }
}
export class WebcamWindow {
  constructor(panel, { onVisibilityChange = () => {} } = {}) {
    this.panel = panel; this.onVisibilityChange = onVisibilityChange; this.state = new WebcamState(); this.pointers = new Map(); this.settingsGeneration = 0;
    this.root = element('section', 'webcam-window'); this.root.dataset.testid = 'demo-webcam-region'; this.root.setAttribute('aria-label', '웹캠 방송');
    const header = element('div', 'webcam-header'); header.append(element('strong', '', '웹캠'));
    this.status = element('span', 'webcam-status', '방송 준비 중'); this.status.setAttribute('role', 'status');
    this.handle = button('↗', 'demo-webcam-resize-handle', () => {}, 'webcam-resize'); this.handle.setAttribute('aria-label', '방송 크기 조절 · 방향키 사용'); header.append(this.status, this.handle);
    this.closeButton = button('×', 'demo-webcam-close-button', () => this.setVisible(false), 'webcam-close'); this.closeButton.setAttribute('aria-label', '웹캠 닫기'); header.append(this.closeButton);
    this.frame = element('div', 'webcam-frame'); this.placeholder = element('p', 'webcam-placeholder', '방송 준비 중'); this.frame.append(this.placeholder);
    this.input = element('div', 'webcam-input'); this.input.tabIndex = 0; this.input.dataset.testid = 'demo-webcam-pan-surface'; this.input.setAttribute('aria-label', '화면 이동 · 방향키와 확대 키, Escape로 종료'); this.input.hidden = true; this.frame.append(this.input);
    this.map = element('div', 'webcam-minimap'); this.map.setAttribute('aria-label', '확대 화면의 위치'); this.mapViewport = element('div'); this.map.append(this.mapViewport); this.frame.append(this.map);
    const controls = element('div', 'webcam-controls'); this.zoomLabel = element('output');
    this.panButton = button('화면 이동', 'demo-webcam-pan-button', () => this.setMoving(!this.state.moving));
    this.reconnect = button('다시 연결', 'demo-webcam-reconnect-button', () => this.load());
    controls.append(this.panButton, button('−', 'demo-webcam-zoom-out-button', () => { this.state.scale(this.state.zoom - .25); this.render(); }), this.zoomLabel,
      button('+', 'demo-webcam-zoom-in-button', () => { this.state.scale(this.state.zoom + .25); this.render(); }), button('맞춤', 'demo-webcam-fit-button', () => { this.state.fit(); this.render(); }), this.reconnect);
    this.root.append(header, this.frame, controls); panel.append(this.root);
    this.resizeObserver = new ResizeObserver(() => { this.state.bounds(panel.clientWidth, panel.clientHeight); this.render(); }); this.resizeObserver.observe(panel);
    this.handle.addEventListener('pointerdown', e => { e.preventDefault(); this.handle.setPointerCapture(e.pointerId); this.resizing = { id: e.pointerId, x: e.clientX, y: e.clientY, size: this.state.size }; });
    this.handle.addEventListener('pointermove', e => { if (this.resizing?.id !== e.pointerId) return; this.state.resize(this.resizing.size + (e.clientX - this.resizing.x - e.clientY + this.resizing.y) / 2); this.render(); });
    for (const name of ['pointerup', 'pointercancel', 'lostpointercapture']) this.handle.addEventListener(name, () => { this.resizing = null; });
    this.handle.addEventListener('keydown', e => { const delta = { ArrowRight: 20, ArrowUp: 20, ArrowLeft: -20, ArrowDown: -20 }[e.key]; if (delta !== undefined || ['Home', 'End'].includes(e.key)) { e.preventDefault(); this.state.resize(e.key === 'Home' ? 200 : e.key === 'End' ? this.state.maximum : this.state.size + delta); this.render(); } });
    this.input.addEventListener('pointerdown', e => { this.input.setPointerCapture(e.pointerId); this.pointers.set(e.pointerId, { x: e.clientX, y: e.clientY }); this.pinchDistance = this.distance(); });
    this.input.addEventListener('pointermove', e => {
      const previous = this.pointers.get(e.pointerId); if (!previous) return;
      this.pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
      if (this.pointers.size === 2) { const distance = this.distance(); if (this.pinchDistance) this.state.scale(this.state.zoom * distance / this.pinchDistance); this.pinchDistance = distance; }
      else this.state.pan(this.state.x + (e.clientX - previous.x) / this.frame.clientWidth, this.state.y + (e.clientY - previous.y) / this.frame.clientHeight);
      this.render();
    });
    for (const name of ['pointerup', 'pointercancel', 'lostpointercapture']) this.input.addEventListener(name, e => { this.pointers.delete(e.pointerId); this.pinchDistance = this.distance(); });
    this.input.addEventListener('wheel', e => { e.preventDefault(); this.state.scale(this.state.zoom + (e.deltaY > 0 ? -.25 : .25)); this.render(); }, { passive: false });
    this.input.addEventListener('dblclick', () => { this.state.scale(this.state.zoom === 1 ? 2 : 1); this.render(); });
    this.input.addEventListener('keydown', e => {
      if (e.key === 'Escape') { e.preventDefault(); this.setMoving(false); this.panButton.focus(); return; }
      const movement = { ArrowLeft: [.1, 0], ArrowRight: [-.1, 0], ArrowUp: [0, .1], ArrowDown: [0, -.1] }[e.key];
      if (movement) { e.preventDefault(); this.state.pan(this.state.x + movement[0], this.state.y + movement[1]); }
      else if (['+', '=', '-'].includes(e.key)) { e.preventDefault(); this.state.scale(this.state.zoom + (e.key === '-' ? -.25 : .25)); }
      this.render();
    });
    this.state.bounds(panel.clientWidth, panel.clientHeight); this.render(); this.load();
  }
  distance() { const p = [...this.pointers.values()]; return p.length === 2 ? Math.hypot(p[0].x - p[1].x, p[0].y - p[1].y) : 0; }
  setVisible(visible) {
    if (visible === !this.root.hidden) return;
    this.root.hidden = !visible;
    if (visible) { this.render(); this.load(); }
    else {
      this.settingsGeneration++; this.settingsController?.abort(); clearTimeout(this.playerTimer);
      this.player?.remove(); this.player = null; this.resizing = null;
      this.state.moving = false; this.pointers.clear(); this.pinchDistance = 0; this.render();
    }
    this.onVisibilityChange(visible);
  }
  setMoving(value) { this.state.moving = value; this.pointers.clear(); this.render(); if (value) this.input.focus(); }
  render() {
    this.root.style.width = `${this.state.size}px`; this.root.style.height = `${this.state.size}px`;
    this.handle.setAttribute('aria-label', `방송 크기 ${Math.round(this.state.size)}px · 방향키로 조절`);
    if (this.player) this.player.style.transform = `translate(${this.state.x * this.frame.clientWidth}px,${this.state.y * this.frame.clientHeight}px) scale(${this.state.zoom})`;
    this.input.hidden = !this.state.moving; this.panButton.setAttribute('aria-pressed', String(this.state.moving)); this.zoomLabel.textContent = `${this.state.zoom.toFixed(2)}×`;
    this.map.hidden = this.state.zoom === 1; const m = this.state.minimap();
    Object.assign(this.mapViewport.style, { width: `${m.width * 100}%`, height: `${m.height * 100}%`, left: `${m.left * 100}%`, top: `${m.top * 100}%` });
  }
  async load() {
    if (this.root.hidden) return;
    const generation = ++this.settingsGeneration; this.settingsController?.abort(); this.settingsController = new AbortController();
    const controller = this.settingsController, timer = setTimeout(() => controller.abort(), 4000);
    this.status.textContent = '방송 설정 확인 중';
    try {
      const response = await fetch('/ui/demo/settings.json', { cache: 'no-store', credentials: 'omit', redirect: 'error', signal: controller.signal }); if (!response.ok) throw new Error('Webcam settings unavailable');
      const url = parseWebcamSettings(await response.json()); if (generation !== this.settingsGeneration) return;
      this.player?.remove(); this.player = null; clearTimeout(this.playerTimer);
      if (!url) { this.placeholder.hidden = false; this.placeholder.textContent = '방송 준비 중'; this.status.textContent = '방송 준비 중'; return; }
      const player = element('iframe'); player.title = '데모 방송 플레이어'; player.src = url.href; player.allow = 'autoplay; fullscreen; picture-in-picture'; player.referrerPolicy = 'strict-origin-when-cross-origin';
      player.addEventListener('load', () => { if (this.player !== player) return; clearTimeout(this.playerTimer); this.status.textContent = '플레이어 표시 · 방송 상태 미확인'; });
      player.addEventListener('error', () => { if (this.player === player) this.status.textContent = '플레이어 연결을 확인하세요'; });
      this.player = player; this.frame.prepend(player); this.placeholder.hidden = true; this.status.textContent = '플레이어 연결 중';
      this.playerTimer = setTimeout(() => { if (this.player === player) this.status.textContent = '연결 지연 · 다시 연결해 보세요'; }, 10000); this.render();
    } catch { if (generation === this.settingsGeneration) { this.player?.remove(); this.player = null; this.placeholder.hidden = false; this.placeholder.textContent = '방송 설정을 확인할 수 없습니다.'; this.status.textContent = '방송 설정 오류'; } }
    finally { clearTimeout(timer); }
  }
  destroy() { this.settingsGeneration++; this.settingsController?.abort(); clearTimeout(this.playerTimer); this.resizeObserver.disconnect(); this.root.remove(); }
}
