import { RUN_STATES, nodeFacts, observationClock, dependentClock, isStale, matches } from './model.mjs';
import { fleetScene, runScene } from './scene.mjs';
import { formatTimestamp, elapsedTime, remainingTime, leaseIdentity, capabilityName, attributeName, attributeValue, technicalAttribute, drainDescription } from './format.mjs';

export function element(tag, className, text) {
  const el = document.createElement(tag);
  if (className) el.className = className;
  if (text !== undefined) el.textContent = text;
  return el;
}
export function button(text, testid, action, className = '') {
  const el = element('button', className, text); el.type = 'button'; el.dataset.testid = testid;
  el.addEventListener('click', action); return el;
}
// 배경에서 시작해 배경에서 끝난 클릭만 닫는다. 카드 내부 드래그는 읽기 조작이다.
export function dismissOnBackdrop(dialog, content, close) {
  let startedOutside = false;
  const outside = e => {
    const r = content.getBoundingClientRect();
    return !content.contains(e.target) || e.clientX < r.left || e.clientX > r.right || e.clientY < r.top || e.clientY > r.bottom;
  };
  dialog.addEventListener('pointerdown', e => { startedOutside = e.button === 0 && outside(e); });
  dialog.addEventListener('pointercancel', () => { startedOutside = false; });
  dialog.addEventListener('click', e => { const dismiss = startedOutside && outside(e); startedOutside = false; if (dismiss) close(); });
}
export function replaceContents(target, children, signature) {
  if (signature !== undefined && target._signature === signature) return;
  target._signature = signature;
  const active = target.contains(document.activeElement) ? document.activeElement : null;
  const identity = active ? { ...active.dataset } : null, scroll = target.scrollTop;
  const openDetails = [...target.querySelectorAll('details[data-testid][open]')].map(el => el.dataset.testid);
  target.replaceChildren(...children);
  for (const el of target.querySelectorAll('details[data-testid]')) el.open = openDetails.includes(el.dataset.testid);
  target.scrollTop = scroll;
  if (identity) [...target.querySelectorAll('[data-testid]')].find(el => Object.entries(identity).every(([k, v]) => el.dataset[k] === v))?.focus({ preventScroll: true });
}
function pair(list, name, value) { list.append(element('dt', '', name), element('dd', '', value ?? '미제공')); }
function message(text) { return element('p', 'muted', text); }
function timestamp(value) { const el = element('time', '', formatTimestamp(value)); if (value) { el.dateTime = value; el.title = value; } return el; }
function timePair(list, name, value, context) {
  const description = element('dd'); description.append(timestamp(value));
  if (context) description.append(element('span', 'time-context', context));
  list.append(element('dt', '', name), description);
}
const shellArgument = value => "'" + value.replaceAll("'", "'\\''") + "'";
export class DashboardView {
  constructor(root, { mode, identity, onRetry, onLogout, onRunSelection = () => {} }) {
    this.root = root; this.mode = mode; this.onRunSelection = onRunSelection;
    this.state = { scene: 'fleet', representation: mode === 'demo' ? '3d' : '2d', run: null, node: null, step: null, filter: '' };
    this.resources = new Map(); this.positions = new Map(); this.now = performance.now();
    root.classList.add('dashboard');
    const header = element('header', 'dashboard-header');
    const brand = element('div', 'brand'); brand.append(element('strong', '', `enode ${mode === 'demo' ? '데모' : '함대'} 현황판`), element('span', 'muted', mode === 'demo' ? '공개 데모' : 'Mediator · 관측'));
    this.headerActions = element('div', 'header-actions');
    this.fleetButton = button('함대', `${mode}-${mode === 'demo' ? 'fleet' : 'grid'}-button`, () => this.changeScene('fleet'));
    this.runButton = button('작업 그래프', `${mode}-${mode === 'demo' ? 'run' : 'graph'}-button`, () => this.changeScene('run'));
    this.twoD = button('2D', `${mode}-2d-button`, () => this.changeRepresentation('2d'));
    this.threeD = button('3D', `${mode}-3d-button`, () => this.changeRepresentation('3d'));
    this.headerActions.append(this.fleetButton, this.runButton, this.twoD, this.threeD, element('span', 'identity', identity));
    if (onLogout) this.headerActions.append(button('인증 종료', `${mode}-logout-button`, onLogout));
    header.append(brand, this.headerActions);
    this.connection = element('div', 'connection'); this.connection.setAttribute('role', 'status');
    this.connectionText = element('span'); this.retry = button('다시 조회', `${mode}-retry-button`, onRetry);
    this.connection.append(this.connectionText, this.retry);
    const layout = element('main', 'dashboard-layout');
    this.panel = element('section', 'scene-panel'); this.panel.dataset.testid = `${mode}-fleet-region`; this.panel.setAttribute('aria-label', '함대와 작업 장면');
    const sceneHeader = element('div', 'scene-header'); this.title = element('h1', '', '함대'); this.subtitle = element('span', 'muted');
    this.backButton = button('← 함대로 돌아가기', `${mode}-graph-back-button`, () => { this.changeScene('fleet'); this.fleetButton.focus({ preventScroll: true }); }, 'scene-back');
    this.sceneContext = element('p', 'scene-context');
    const heading = element('div', 'scene-heading'); heading.append(this.title, this.subtitle, this.sceneContext);
    sceneHeader.append(this.backButton, heading);
    this.viewport = element('div', 'scene-viewport'); this.viewport.tabIndex = 0; this.viewport.dataset.testid = `${mode}-scene-viewport`; this.viewport.setAttribute('aria-label', '장면 탐색 영역');
    this.canvas = element('div', 'scene-canvas'); this.viewport.append(this.canvas);
    this.inspector = element('aside', 'inspector'); this.inspector.setAttribute('aria-label', '선택 상세'); this.inspector.hidden = true;
    this.navigation = element('div', 'scene-navigation');
    this.zoomLabel = element('output', '', '100%');
    this.navigation.append(button('−', `${mode}-graph-zoom-out-button`, () => this.setZoom(this.zoom - .15)), this.zoomLabel,
      button('+', `${mode}-graph-zoom-in-button`, () => this.setZoom(this.zoom + .15)),
      button('화면 맞춤', `${mode}-graph-fit-button`, () => this.fit()), button('100%', `${mode}-graph-actual-size-button`, () => this.setZoom(1)));
    this.zoomHint = element('span', 'zoom-hint', '휠로 확대·축소');
    this.navigation.prepend(this.zoomHint);
    this.panel.append(sceneHeader, this.viewport, this.inspector, this.navigation);
    this.sidebar = element('aside', 'run-sidebar'); this.sidebar.dataset.testid = `${mode}-run-list-region`; this.sidebar.setAttribute('aria-label', '현재 작업 목록');
    this.sidebar.append(element('h2', '', '현재 작업 목록')); this.newTaskSlot = element('div', 'new-task-slot'); this.sidebar.append(this.newTaskSlot);
    const filterLabel = element('label', 'filter-label', '상태');
    this.filter = element('select'); this.filter.dataset.testid = `${mode}-run-state-filter`;
    for (const value of ['', ...RUN_STATES]) { const option = element('option', '', value || '전체 상태'); option.value = value; this.filter.append(option); }
    this.filter.addEventListener('change', () => { this.closeInspector(); this.state.filter = this.filter.value; this.render(); }); filterLabel.append(this.filter);
    this.runStatus = message('첫 관측을 기다리는 중'); this.runList = element('div', 'run-list');
    this.submissionNotice = element('p', 'submission-notice'); this.submissionNotice.setAttribute('role', 'status'); this.submissionNotice.hidden = true;
    this.requirements = element('section', 'requirements'); this.stepList = element('details', 'step-list');
    this.askList = element('section', 'requirements'); this.askList.hidden = mode !== 'fleet';
    this.sidebar.append(filterLabel, this.runStatus, this.submissionNotice, this.runList, message('QUEUED 작업은 노드에 배치하지 않습니다.'), this.askList, this.requirements, this.stepList);
    layout.append(this.panel, this.sidebar); root.replaceChildren(header, this.connection, layout);
    this.zoom = 1; this.dimensions = { width: 880, height: 620 }; this.autoFit = true;
    this.resizeObserver = new ResizeObserver(() => { if (this.autoFit) this.fit(); }); this.resizeObserver.observe(this.viewport);
    this.dismissOutside = e => {
      if (this.inspector.hidden || this.inspector.contains(e.target)) return;
      if (e.type !== 'wheel' && e.target.closest?.(`[data-testid="${mode}-node-select-button"], [data-testid="${mode}-step-select-button"]`)) return;
      this.closeInspector();
    };
    this.dismissKey = e => {
      if (this.inspector.hidden || this.root.querySelector('dialog[open]')) return;
      if (e.key === 'Escape') { e.preventDefault(); this.closeInspector(true); }
      else if (['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown', 'PageUp', 'PageDown', 'Home', 'End'].includes(e.key) && !this.inspector.contains(e.target)) this.closeInspector();
    };
    for (const name of ['pointerdown', 'click', 'focusin']) document.addEventListener(name, this.dismissOutside, true);
    document.addEventListener('wheel', this.dismissOutside, { capture: true, passive: true });
    document.addEventListener('keydown', this.dismissKey, true);
    this.wheelZoom = e => {
      if (!this.canvas.querySelector('svg') || e.shiftKey || Math.abs(e.deltaX) > Math.abs(e.deltaY) || !e.deltaY) return;
      e.preventDefault();
      const delta = e.deltaY * (e.deltaMode === 1 ? 16 : e.deltaMode === 2 ? this.viewport.clientHeight : 1);
      this.setZoom(this.zoom * Math.exp(-Math.max(-500, Math.min(500, delta)) * .002), true, { x: e.clientX, y: e.clientY });
    };
    this.viewport.addEventListener('wheel', this.wheelZoom, { passive: false });
  }
  viewKey() { return `${this.state.scene}:${this.state.scene === 'run' ? this.state.run : ''}:${this.state.representation}`; }
  savePosition() { this.positions.set(this.viewKey(), { x: this.viewport.scrollLeft, y: this.viewport.scrollTop, zoom: this.zoom, autoFit: this.autoFit }); }
  restorePosition() {
    const p = this.positions.get(this.viewKey()); this.autoFit = p?.autoFit ?? true;
    if (this.autoFit) this.fit(); else this.setZoom(p.zoom, false);
    this.viewport.scrollLeft = p?.x || 0; this.viewport.scrollTop = p?.y || 0;
  }
  changeScene(scene) { if (scene === 'run' && !this.state.run) return; this.closeInspector(); this.savePosition(); this.state.scene = scene; this.render(); this.restorePosition(); }
  changeRepresentation(value) { this.closeInspector(); this.savePosition(); this.state.representation = value; this.render(); this.restorePosition(); }
  selectRun(id, submitted = false) {
    this.closeInspector(); this.savePosition();
    this.state.run = id; this.state.scene = 'run';
    if (submitted) { this.state.representation = '3d'; this.state.filter = ''; this.filter.value = ''; }
    this.onRunSelection(id); this.render(); this.restorePosition();
  }
  selectNode(id) { this.changeScene('fleet'); this.state.node = id; this.render(); }
  selectStep(id) { this.changeScene('run'); this.state.step = id; this.render(); }
  closeInspector(restoreFocus = false) {
    const node = this.state.node, step = this.state.step;
    if (!node && !step) return;
    this.state.node = this.state.step = null; this.inspector.hidden = true; this.render();
    if (restoreFocus) [...this.root.querySelectorAll('[data-node-id], [data-step-id]')].find(el => node ? el.dataset.nodeId === node : el.dataset.stepId === step)?.focus({ preventScroll: true });
  }
  setZoom(value, manual = true, anchor) {
    if (manual) this.closeInspector();
    const old = this.zoom; this.zoom = Math.max(manual ? .15 : Number.EPSILON, Math.min(3, value)); if (manual) this.autoFit = false;
    const svg = this.canvas.querySelector('svg');
    const before = svg?.getBoundingClientRect(), viewport = this.viewport.getBoundingClientRect();
    const point = anchor || { x: viewport.left + this.viewport.clientWidth / 2, y: viewport.top + this.viewport.clientHeight / 2 };
    if (svg) { svg.style.width = `${this.dimensions.width * this.zoom}px`; svg.style.height = `${this.dimensions.height * this.zoom}px`; }
    this.zoomLabel.textContent = `${Math.round(this.zoom * 100)}%`;
    if (svg && this.zoom !== old) {
      const after = svg.getBoundingClientRect();
      this.viewport.scrollLeft += after.left + (point.x - before.left) * this.zoom / old - point.x;
      this.viewport.scrollTop += after.top + (point.y - before.top) * this.zoom / old - point.y;
    }
  }
  fit() { this.autoFit = true; if (this.viewport.clientWidth <= 20 || this.viewport.clientHeight <= 20) return; this.setZoom(Math.min(1, (this.viewport.clientWidth - 20) / this.dimensions.width, (this.viewport.clientHeight - 20) / this.dimensions.height), false); }
  update(resources, now = performance.now()) { this.resources = resources; this.now = now; this.render(); }
  resourceStatus(key) {
    const r = this.resources.get(key);
    if (!r?.data) return r?.error || '첫 관측을 기다리는 중';
    const age = Math.max(0, Math.floor((this.now - r.receivedAt) / 1000));
    return `${isStale(r, this.now) ? '갱신 지연 · 시간 정지' : r.error ? '미갱신' : '관측 중'} · ${age}초 전${r.error ? ` · ${r.error}` : ''}`;
  }
  render() {
    const nodesResource = this.resources.get('nodes'), runsResource = this.resources.get('runs');
    const nodes = nodesResource?.data?.nodes || [], runs = runsResource?.data?.runs || [];
    const details = new Map([...this.resources].filter(([k]) => k.startsWith('detail:')).map(([k, r]) => [k.slice(7), r.data]));
    const asks = this.resources.get('asks')?.data?.asks || [], now = observationClock(nodesResource, this.now) ?? 0;
    const detailResource = this.resources.get(`detail:${this.state.run}`), detail = detailResource?.data;
    this.connectionText.textContent = `함대 ${this.resourceStatus('nodes')}  /  목록 ${this.resourceStatus('runs')}${nodesResource?.data ? ` · ${formatTimestamp(nodesResource.data.observed_at)}` : ''}`;
    this.connection.classList.toggle('is-stale', [...this.resources.values()].some(r => r.error || isStale(r, this.now)));
    this.runStatus.textContent = `${runs.length}개 작업 · ${this.resourceStatus('runs')}`;
    const isFleet = this.state.scene === 'fleet', selectedRun = runs.find(r => r.run_id === this.state.run);
    this.panel.dataset.scene = this.state.scene; this.backButton.hidden = isFleet;
    this.title.textContent = isFleet ? '함대' : '작업 그래프';
    this.subtitle.textContent = isFleet ? `${nodes.length}개 노드 · 모형을 눌러 상세 확인` : `${detail?.steps?.length ?? 0}개 단계 · 화살표는 실행 의존 관계`;
    this.sceneContext.hidden = isFleet;
    this.sceneContext.textContent = isFleet ? '' : `${selectedRun?.submitter || '제출자 미확인'} · ${this.state.run} · ${detail?.state || selectedRun?.state || '조회 중'}`;
    this.sceneContext.title = this.sceneContext.textContent;
    this.runButton.disabled = !this.state.run;
    for (const [el, pressed] of [[this.fleetButton, this.state.scene === 'fleet'], [this.runButton, this.state.scene === 'run'], [this.twoD, this.state.representation === '2d'], [this.threeD, this.state.representation === '3d']]) el.setAttribute('aria-pressed', String(pressed));
    const signature = JSON.stringify([this.state.scene, this.state.representation, this.state.node, this.state.step, this.state.run, nodes, runs.map(r => [r.run_id, r.submitter]), detail, [...details], asks, nodes.map(n => nodeFacts(n, details, asks, now).expiring)]);
    if (signature !== this.canvas._signature) {
      let scene;
      if (this.state.scene === 'fleet' && nodes.length) scene = fleetScene({ nodes, runs, details, asks, now, iso: this.state.representation === '3d', mode: this.mode, selected: this.state.node, onSelect: id => this.selectNode(id) });
      else if (this.state.scene === 'run' && detail?.steps?.length && !detail.graphError) scene = runScene({ steps: detail.steps, nodes, iso: this.state.representation === '3d', mode: this.mode, selected: this.state.step, onSelect: id => this.selectStep(id) });
      if (scene) {
        const position = { x: this.viewport.scrollLeft, y: this.viewport.scrollTop };
        this.dimensions = scene; replaceContents(this.canvas, [scene.element], signature); this.setZoom(this.zoom, false);
        if (this.autoFit) this.fit();
        else { this.viewport.scrollLeft = position.x; this.viewport.scrollTop = position.y; }
      }
      else replaceContents(this.canvas, [element('div', 'scene-empty', this.state.scene === 'fleet' ? nodesResource?.data ? '현재 관측된 노드가 없습니다.' : this.resourceStatus('nodes') : detail?.graphError ? `그래프를 표시할 수 없습니다. ${detail.graphError} — 아래 단계 목록을 확인하세요.` : detail ? `${detail.state} · 아직 관측된 단계가 없습니다.` : this.resourceStatus(`detail:${this.state.run}`))], signature);
    }
    const filtered = runs.filter(r => !this.state.filter || r.state === this.state.filter);
    const rows = filtered.map(r => {
      const b = button('', `${this.mode}-run-select-button`, () => this.selectRun(r.run_id), 'run-row'); b.dataset.runId = r.run_id; b.setAttribute('aria-pressed', String(this.state.run === r.run_id));
      b.append(element('span', 'run-id', r.run_id), element('span', `state state-${r.state.toLowerCase().replace(/[^a-z]/g, '')}`, r.state),
        element('span', 'run-meta', r.submitter || '제출자 미제공'), element('span', 'run-meta', formatTimestamp(r.created_at)), element('span', 'run-meta', r.assigned.length ? r.assigned.map(a => `${a.as}: ${a.nodes.map(n => n.label || n.node).join(', ')}`).join(' / ') : '노드 미배정'));
      return b;
    });
    replaceContents(this.runList, rows.length ? rows : [message(runsResource?.data ? '해당하는 작업이 없습니다.' : '작업 목록 미확인')], JSON.stringify([filtered, this.state.run]));
    this.renderInspector(nodes, details, asks, now, detail, detailResource);
    this.renderRequirements(detail, detailResource, nodes);
    if (this.mode === 'fleet') {
      const items = [element('h3', '', '사람 응답 대기'), message(this.resourceStatus('asks'))];
      for (const a of asks) {
        const b = button(`${a.run_id} · 단계 ${a.seq}`, 'fleet-ask-run-button', () => this.selectRun(a.run_id), 'text-link'); b.dataset.runId = a.run_id;
        const askResource = this.resources.get('asks');
        const askNow = dependentClock(runsResource, [askResource], this.now);
        items.push(b, message(a.prompt), message(`응답자: ${a.answerers?.join(', ') || '미제공'}\n질문: ${formatTimestamp(a.asked_at)}${a.asked_at && askNow !== null ? ` · ${elapsedTime(a.asked_at, askNow)} (목록 관측 시각 기준)` : ''}\n기한: ${a.deadline ? formatTimestamp(a.deadline) : '기한 없음'}`), message(`runctl asks\nrunctl answer ${shellArgument(a.run_id)} ${a.seq} --set field=value`));
      }
      if (!asks.length) items.push(message(this.resources.get('asks')?.data ? '현재 인박스에 질문이 없습니다.' : '인박스 미확인'));
      replaceContents(this.askList, items, items.map(x => x.textContent).join('\n'));
    }
    const stepItems = [element('summary', '', '단계와 의존 관계')];
    for (const s of detail?.steps || []) {
      const b = button(`${s.seq}. ${s.id} · ${s.state} · needs: ${s.needs.join(', ') || '없음'}`, `${this.mode}-step-select-button`, () => this.selectStep(s.id), 'text-step'); b.dataset.stepId = s.id; stepItems.push(b);
    }
    replaceContents(this.stepList, stepItems, JSON.stringify(detail?.steps)); this.stepList.hidden = !detail;
  }
  renderRequirements(detail, resource, nodes) {
    this.requirements.hidden = !detail;
    if (!detail) return;
    const contents = [element('h3', '', '요구 능력'), message(`작업 상세 ${this.resourceStatus(`detail:${this.state.run}`)}`)];
    if (detail.warnings?.length) contents.push(element('p', 'warning', detail.warnings.join(' · ')));
    const requires = detail.requires ?? resource?.previousRequires;
    if (detail.requires === undefined) contents.push(element('p', 'warning', requires ? '요구 정보 미확인 · 이전 관측을 표시합니다.' : '요구 정보 미확인'));
    for (const r of requires || []) {
      const list = element('dl', 'attribute-list');
      for (const [key, value] of Object.entries(r.attrs)) pair(list, attributeName(key), attributeValue(key, value));
      contents.push(element('h3', '', `${r.as} · ${capabilityName(r.capability)} × ${r.count || 1}`), list, message(`일치 광고 ${nodes.filter(n => matches(n, r)).length}개 (배정 가능 여부와 별개)`));
    }
    if (detail.requires?.length === 0) contents.push(message('요구 능력 없음'));
    if (detail.verdict) contents.push(element('h3', '', `검증 · ${detail.verdict.state}`), ...detail.verdict.checks.map(c => message(`${c.ok ? '통과' : '실패'} · ${c.note || '설명 없음'}`)));
    replaceContents(this.requirements, contents, JSON.stringify([detail, resource?.previousRequires, nodes, contents[1].textContent]));
  }
  renderInspector(nodes, details, asks, now, detail, detailResource) {
    const node = nodes.find(n => n.node_id === this.state.node), step = detail?.steps?.find(s => s.id === this.state.step);
    const isNode = this.state.scene === 'fleet'; this.inspector.hidden = isNode ? !node : !step;
    if (this.inspector.hidden) return;
    const items = [], list = element('dl');
    const close = button('×', `${this.mode}-inspector-close-button`, () => this.closeInspector(true), 'inspector-close'); close.setAttribute('aria-label', '상세 닫기'); items.push(close);
    if (isNode) {
      const facts = nodeFacts(node, details, asks, now); items.push(element('h2', '', node.label || node.node_id), element('p', `node-tone-${facts.tone}`, facts.label));
      pair(list, '작업 수락', drainDescription(node.draining));
      if (node.lease) {
        const leaseResource = this.resources.get(`detail:${node.lease.run_id}`), dependent = [leaseResource, ...(this.mode === 'fleet' ? [this.resources.get('asks')] : [])];
        const freeze = dependent.filter(r => !r?.data || isStale(r, this.now));
        const leaseNow = dependentClock(this.resources.get('nodes'), dependent, this.now);
        const owner = leaseIdentity(node, this.resources.get('runs')?.data?.runs);
        pair(list, '사용 중인 작업의 제출자', owner.submitter);
        pair(list, '현재 작업', owner.runId);
        if (isStale(this.resources.get('runs'), this.now) || this.resources.get('runs')?.error) pair(list, '제출자 정보', '이전 목록 관측 · 갱신 지연');
        timePair(list, '임대 갱신 기한', node.lease.not_after, `${facts.asked ? '사람 응답 대기 중에는 임대가 만료되지 않습니다.' : remainingTime(node.lease.not_after, leaseNow)}${freeze.length ? ' · 관련 관측 미갱신' : ''}`);
        for (const s of details.get(node.lease.run_id)?.steps || []) if (s.node === node.node_id && s.state === 'CLAIMED') {
          pair(list, '실행 단계', `${s.id} · 회차 ${s.attempt ?? '미제공'}`); timePair(list, '실행 시작', s.started_at);
        }
        const b = button('사용 중인 작업 보기 →', `${this.mode}-node-run-button`, () => this.selectRun(node.lease.run_id), 'text-link'); b.dataset.nodeId = node.node_id; items.push(b);
      }
      const capabilities = element('section', 'capability-section'); capabilities.append(element('h3', '', '제공 기능'));
      for (const c of node.capabilities) {
        const group = element('section', 'capability-group'), attributes = element('dl', 'attribute-list');
        const name = element('h3', '', capabilityName(c.capability)); name.title = c.capability; group.append(name);
        for (const [key, value] of Object.entries(c.attrs)) if (!technicalAttribute(key)) pair(attributes, attributeName(key), attributeValue(key, value));
        if (!Object.hasOwn(c.attrs, 'sandbox')) pair(attributes, '실행 격리 (광고값)', '미제공');
        group.append(attributes); capabilities.append(group);
      }
      if (!node.capabilities.length) capabilities.append(message('광고된 기능이 없습니다.'));
      const freshness = element('section', 'inspector-section'), times = element('dl');
      freshness.append(element('h3', '', '연결 정보'));
      timePair(times, '마지막 광고 수신', node.seen_at, elapsedTime(node.seen_at, now));
      timePair(times, '광고 유효 기한', node.expires_at, remainingTime(node.expires_at, now));
      freshness.append(times, message('시각은 현재 브라우저 시간대 기준입니다.'));
      const technical = element('details', 'technical-details'); technical.dataset.testid = `${this.mode}-node-technical-details`;
      const ids = element('dl'); pair(ids, '노드 ID', node.node_id); pair(ids, '인스턴스 ID', node.instance); pair(ids, '회수 정책 원문', node.draining || '없음');
      for (const c of node.capabilities) for (const [key, value] of Object.entries(c.attrs)) if (technicalAttribute(key)) pair(ids, `${attributeName(key)} · ${c.capability}`, value);
      const technicalToggle = element('summary', '', '기술 상세 · ID와 원문 광고'); technicalToggle.dataset.testid = `${this.mode}-node-technical-toggle`;
      technical.append(technicalToggle, ids, element('pre', '', JSON.stringify(node, null, 2)));
      items.push(capabilities, freshness, technical);
    } else {
      items.push(element('h2', '', step.id)); pair(list, '상태', step.state); pair(list, '용도', step.uses); pair(list, '선행 단계', step.needs.join(', ') || '없음');
      pair(list, '경로 선택', `${step.chosen ? '선택됨' : '선택 안 됨'}${step.state === 'SKIPPED' ? step.chosen ? ' · 도달하지 못함' : ' · 실행하지 않는 경로' : ''}`);
      pair(list, '실행 노드', nodes.find(n => n.node_id === step.node)?.label || step.node); pair(list, '회차', step.attempt); timePair(list, '시작', step.started_at); timePair(list, '종료', step.ended_at);
      const b = button('이 단계의 노드 보기 →', `${this.mode}-step-node-button`, () => this.selectNode(step.node), 'text-link'); b.disabled = !nodes.some(n => n.node_id === step.node); items.push(b);
      if (step.state === 'ASKED') {
        const askResource = this.resources.get('asks'), ask = asks.find(a => a.run_id === detail.run_id && a.seq === step.seq);
        pair(list, '사람 응답', 'ASKED · 답변을 기다리는 중');
        if (this.mode === 'fleet') {
          pair(list, '인박스', this.resourceStatus('asks'));
          if (ask) { pair(list, '질문', ask.prompt); pair(list, '응답자', ask.answerers?.join(', ')); timePair(list, '질문 시각', ask.asked_at); if (ask.deadline) timePair(list, '기한', ask.deadline); else pair(list, '기한', '기한 없음'); }
          else pair(list, '질문 내용', askResource?.data ? '현재 인박스에 없음' : '미확인');
          pair(list, 'CLI 안내', `runctl asks\nrunctl answer ${shellArgument(detail.run_id)} ${step.seq} --set field=value`);
        }
      }
      if (detailResource?.error) pair(list, '상세 관측', detailResource.error);
    }
    items.splice(3, 0, list); replaceContents(this.inspector, items, JSON.stringify([isNode, node, step, items.map(x => x.textContent), details.get(node?.lease?.run_id), detailResource?.error]));
  }
  destroy() {
    this.viewport.removeEventListener('wheel', this.wheelZoom);
    for (const name of ['pointerdown', 'click', 'focusin', 'wheel']) document.removeEventListener(name, this.dismissOutside, true);
    document.removeEventListener('keydown', this.dismissKey, true);
    this.resizeObserver.disconnect(); this.root.replaceChildren();
  }
}
