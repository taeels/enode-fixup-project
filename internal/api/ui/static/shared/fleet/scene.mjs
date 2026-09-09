import { COLORS, graphLayout, nodeFacts, runFlowFacts } from './model.mjs';
import { leaseIdentity, wrapLabel, runDisplayName, stepDisplayName } from './format.mjs';

const NS = 'http://www.w3.org/2000/svg';
function svg(tag, attributes = {}, text) {
  const el = document.createElementNS(NS, tag);
  for (const [key, value] of Object.entries(attributes)) el.setAttribute(key, value);
  if (text !== undefined) el.textContent = text;
  return el;
}
function label(parent, x, y, text, attributes = {}) {
  parent.append(svg('text', { x, y, fill: '#E6EAF0', 'font-size': 13, ...attributes }, text));
}
function selectable(el, title, testid, idKey, id, action, selected) {
  el.setAttribute('role', 'button'); el.setAttribute('tabindex', '0');
  el.setAttribute('aria-label', title); el.setAttribute('aria-pressed', String(selected));
  el.setAttribute('data-testid', testid); el.setAttribute(`data-${idKey}-id`, id);
  el.classList.add('scene-item');
  el.addEventListener('click', action);
  el.addEventListener('keydown', e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); action(); } });
}
// 모형 분류는 label이 아니라 명시적 광고 속성만 사용한다.
export function machineKind(node) {
  if (node.capabilities.some(c => typeof c.attrs?.board === 'string' && c.attrs.board.length > 0)) return 'board';
  const values = node.capabilities.flatMap(c => ['device', 'device_type', 'kind', 'model'].map(k => c.attrs?.[k]?.toLowerCase()));
  if (values.some(v => ['board', 'raspberry-pi', 'raspberry pi', 'rpi'].includes(v))) return 'board';
  if (values.some(v => ['workstation', 'desktop', 'pc'].includes(v))) return 'workstation';
  return 'generic';
}
function machine(parent, kind, color) {
  const poly = (points, fill, stroke = '#0B0D10') => parent.append(svg('polygon', { points, fill, stroke, 'stroke-width': 1.5 }));
  if (kind === 'workstation') {
    poly('-45,-46 5,-21 5,22 -45,-3', '#1B2027');
    poly('-45,-46 -37,-51 13,-26 5,-21', '#39434F');
    poly('5,-21 13,-26 13,17 5,22', '#11161D');
    poly('-39,-38 -1,-19 -1,11 -39,-8', '#172B41', color);
    poly('-22,10 -18,12 -18,26 -22,24', '#39434F');
    poly('-32,24 -20,18 -5,26 -17,32', '#262D36');
    poly('32,-24 51,-34 69,-25 51,-15', '#39434F');
    poly('32,-24 51,-15 51,32 32,23', '#1B2027');
    poly('51,-15 69,-25 69,22 51,32', '#10151B');
    parent.append(svg('circle', { cx: 39, cy: -12, r: 2, fill: color }));
  } else if (kind === 'board') {
    poly('-36,1 0,-18 37,1 0,20', '#23352F');
    poly('-36,1 0,20 0,29 -36,10', '#17251F');
    poly('0,20 37,1 37,10 0,29', '#10231C');
    poly('-13,0 0,-7 14,0 0,8', '#39434F', '#7A8698');
    for (let i = 0; i < 6; i++) parent.append(svg('circle', { cx: -25 + i * 5, cy: -1 - i * 2.5, r: 1.5, fill: '#E2A33C' }));
    parent.append(svg('circle', { cx: 16, cy: 5, r: 2, fill: color }));
  } else {
    poly('-25,-10 0,-23 26,-10 0,4', '#39434F');
    poly('-25,-10 0,4 0,31 -25,17', '#1B2027');
    poly('0,4 26,-10 26,17 0,31', '#10151B');
    parent.append(svg('circle', { cx: -12, cy: 8, r: 3, fill: color }));
  }
}
export function fleetScene({ nodes, runs = [], details, asks, now, iso, mode, selected, onSelect }) {
  const columns = Math.max(2, Math.ceil(Math.sqrt(nodes.length))), rows = Math.ceil(nodes.length / columns);
  const gapX = 220, gapY = 125;
  const width = iso ? Math.max(880, (columns + rows - 2) * gapX + 320) : Math.max(760, columns * 330 + 60);
  const height = iso ? Math.max(620, (columns + rows - 2) * gapY + 300) : Math.max(460, rows * 210 + 80);
  const originX = (width - (columns - rows) * gapX) / 2;
  const root = svg('svg', { width, height, viewBox: `0 0 ${width} ${height}`, 'aria-label': '함대 관측', role: 'group' });
  if (iso) {
    const floor = svg('g', { stroke: '#262D36', 'stroke-width': 1, opacity: .6 });
    const cx = originX, top = 75, span = Math.max(columns, rows) + 1;
    for (let i = 0; i <= span; i++) {
      floor.append(svg('path', { d: `M ${cx - i * gapX} ${top + i * gapY} l ${span * gapX} ${span * gapY}`, fill: 'none' }));
      floor.append(svg('path', { d: `M ${cx + i * gapX} ${top + i * gapY} l ${-span * gapX} ${span * gapY}`, fill: 'none' }));
    }
    root.append(floor);
  }
  nodes.forEach((node, index) => {
    const col = index % columns, row = Math.floor(index / columns);
    const x = iso ? originX + (col - row) * gapX : 30 + col * 330;
    const y = iso ? 180 + (col + row) * gapY : 40 + row * 210;
    const facts = nodeFacts(node, details, asks, now), color = COLORS[facts.tone];
    const owner = leaseIdentity(node, runs), status = `${facts.label}${owner ? ` · ${owner.submitter}` : ''}`;
    const group = svg('g', { transform: `translate(${x} ${y})`, opacity: facts.expiring ? .65 : 1 });
    selectable(group, `${node.label || node.node_id} · ${status}`, `${mode}-node-select-button`, 'node', node.node_id, () => onSelect(node.node_id), selected === node.node_id);
    group.append(svg('title', {}, `${node.label || node.node_id}\n${status}${owner ? `\n작업: ${runDisplayName(owner.runId)}\n${owner.runId}` : ''}`));
    if (iso) {
      group.append(svg('path', { d: 'M -100 30 L 0 -20 L 100 30 L 0 80 Z', fill: '#1B2027', stroke: color, 'stroke-width': selected === node.node_id ? 3 : 1.5 }));
      group.append(svg('path', { d: 'M -100 30 L 0 80 L 100 30 L 100 42 L 0 92 L -100 42 Z', fill: '#10151B' }));
      machine(group, machineKind(node), color);
      const nameLines = wrapLabel(node.label || node.node_id);
      nameLines.forEach((line, i) => label(group, 0, -68 - (nameLines.length - 1 - i) * 18, line, { 'text-anchor': 'middle', 'font-weight': 600, class: 'node-name' }));
      label(group, 0, -48, wrapLabel(status, 36, 1)[0], { 'text-anchor': 'middle', fill: color, 'font-size': 11 });
    } else {
      group.append(svg('rect', { width: 280, height: 176, rx: 10, fill: '#1B2027', stroke: selected === node.node_id ? color : '#39434F', 'stroke-width': 2 }));
      group.append(svg('circle', { cx: 17, cy: 25, r: 4, fill: color }));
      wrapLabel(node.label || node.node_id).forEach((line, i) => label(group, 30, 30 + i * 18, line, { 'font-weight': 600, class: 'node-name' }));
      label(group, 15, 76, facts.label, { fill: color });
      label(group, 15, 102, wrapLabel(node.capabilities.map(c => c.capability).join(' · '), 36, 1)[0], { fill: '#98A4B3', 'font-size': 11 });
      label(group, 15, 128, wrapLabel(`sandbox: ${[...new Set(node.capabilities.map(c => c.attrs.sandbox ?? '미제공'))].join(', ')}`, 36, 1)[0], { fill: '#98A4B3', 'font-size': 11 });
      label(group, 15, 154, owner ? wrapLabel(`작업 제출자 · ${owner.submitter}`, 36, 1)[0] : facts.expiring ? '광고 만료 임박 · 상세 확인' : '현재 임대 없음', { fill: '#98A4B3', 'font-size': 11 });
    }
    root.append(group);
  });
  return { element: root, width, height };
}
function flowLink(parent, from, to, { vertical = false, context = false, active = false, color = '#53687B' } = {}) {
  const end = vertical ? { x: to.x, y: to.y - 8 } : { x: to.x - 8, y: to.y };
  const bend = vertical ? Math.max(28, (to.y - from.y) / 2) : Math.max(28, (to.x - from.x) / 2);
  const controls = vertical ? `${from.x} ${from.y + bend}, ${end.x} ${end.y - bend}` : `${from.x + bend} ${from.y}, ${end.x - bend} ${end.y}`;
  parent.append(svg('path', { d: `M ${from.x} ${from.y} C ${controls}, ${end.x} ${end.y}`, fill: 'none', stroke: color, 'stroke-width': active ? 2.5 : 1.8, class: context ? 'flow-link flow-context-link' : 'flow-link dependency-edge', 'data-active': active }));
  parent.append(svg('path', { d: vertical ? `M ${to.x - 4} ${to.y - 10} l 4 7 l 4 -7` : `M ${to.x - 10} ${to.y - 4} l 7 4 l -7 4`, fill: 'none', stroke: color, 'stroke-width': 1.8 }));
}

function flowActor(parent, { x, y, kind, name, status, description, color, iso, mode, selected, onSelect, active }) {
  const group = svg('g', { transform: `translate(${x} ${y})`, class: `flow-actor${active ? ' is-active' : ''}` });
  const title = kind === 'guest' ? mode === 'demo' ? 'Guest' : 'Submitter' : 'Mediator';
  selectable(group, `${title} · ${name} · ${status}`, `${mode}-flow-actor-button`, 'actor', kind, () => onSelect(kind), selected === kind);
  group.append(svg('title', {}, `${title}\n${name}\n${status}\n${description}`));
  if (iso) group.append(svg('path', { d: 'M 8 168 H 270 V 8 L 282 20 V 180 H 20 Z', fill: '#101C26', stroke: '#3B5063' }));
  group.append(svg('rect', { width: 270, height: 168, rx: 16, fill: '#192631', stroke: selected === kind ? color : '#42596B', 'stroke-width': selected === kind ? 3 : 1.5 }));
  label(group, 20, 27, `${kind === 'guest' ? '01' : '02'} / ${title.toUpperCase()}`, { fill: color, 'font-size': 11, 'letter-spacing': 2 });
  group.append(svg('circle', { cx: 45, cy: 74, r: 25, fill: '#0D1A23', stroke: color, 'stroke-opacity': .5, class: 'flow-halo' }));
  if (kind === 'guest') {
    group.append(svg('circle', { cx: 45, cy: 67, r: 7, fill: '#9BD8D0' }));
    group.append(svg('path', { d: 'M 31 87 Q 31 76 45 76 Q 59 76 59 87', fill: '#9BD8D0' }));
  } else {
    group.append(svg('path', { d: 'M 45 56 L 60 65 L 60 83 L 45 92 L 30 83 L 30 65 Z M 30 65 L 45 74 L 60 65 M 45 74 V 92 M 45 56 V 64', fill: '#173449', stroke: color, 'stroke-width': 1.8 }));
  }
  wrapLabel(name, 20, 2).forEach((line, i) => label(group, 84, 69 + i * 19, line, { 'font-size': 15, 'font-weight': 600 }));
  group.append(svg('circle', { cx: 24, cy: 116, r: 3, fill: color }));
  label(group, 35, 120, status, { fill: color, 'font-size': 12 });
  label(group, 20, 146, wrapLabel(description, 39, 1)[0], { fill: '#B2C0CD', 'font-size': 11 });
  parent.append(group);
}

export function runScene({ steps = [], nodes = [], run = {}, stale = false, compact = false, iso, mode, selected, onSelect, actor, onActorSelect }) {
  const facts = runFlowFacts(run.state, steps, stale);
  const placed = graphLayout(steps).map(s => {
    const level = (s.x - 65) / 240, lane = (s.y - 70) / 180;
    return { ...s, x: compact ? 46 + lane * 242 : s.x + (iso ? lane * 24 : 0), y: compact ? 654 + level * 210 : s.y + 340 + (iso ? level * 28 : 0) };
  });
  const byID = new Map(placed.map(s => [s.id, s]));
  const width = Math.max(compact ? 360 : 840, ...placed.map(s => s.x + 250));
  const height = Math.max(compact ? 870 : 640, ...placed.map(s => s.y + 210));
  const guest = { x: compact ? (width - 270) / 2 : 55, y: compact ? 55 : 76 };
  const mediator = { x: compact ? guest.x : 485, y: compact ? 310 : 76 };
  const zoneY = compact ? 570 : 326;
  const root = svg('svg', { width, height, viewBox: `0 0 ${width} ${height}`, class: 'run-flow', 'aria-label': '요청자, Mediator, enode 실행 흐름과 단계 의존 그래프', role: 'group', 'data-paused': stale });
  root.append(svg('desc', {}, '점선은 요청과 배정의 관계, 실선은 실제 단계의 의존 관계입니다. 게스트와 Mediator를 선택하면 역할과 작업 정보를 확인할 수 있습니다.'));
  root.append(svg('rect', { x: 24, y: zoneY, width: width - 48, height: height - zoneY - 24, rx: 20, fill: '#101C26', stroke: '#314758' }));
  if (compact) {
    flowLink(root, { x: guest.x + 135, y: guest.y + 168 }, { x: mediator.x + 135, y: mediator.y }, { vertical: true, context: true, color: '#6BAEA6' });
    label(root, guest.x + 153, 269, '요청 접수', { fill: '#93BEBB', 'font-size': 11 });
  } else {
    flowLink(root, { x: guest.x + 270, y: guest.y + 84 }, { x: mediator.x, y: mediator.y + 84 }, { context: true, color: '#6BAEA6' });
    label(root, 405, 142, '요청 접수', { 'text-anchor': 'middle', fill: '#93BEBB', 'font-size': 11 });
  }
  const links = svg('g', { class: 'run-connections' });
  for (const s of placed.filter(s => !s.needs.length)) {
    flowLink(links, { x: mediator.x + 135, y: mediator.y + 168 }, { x: s.x + 100, y: s.y }, { vertical: true, context: true, active: facts.activeSteps.includes(s.id), color: facts.activeSteps.includes(s.id) ? COLORS.leased : '#526E84' });
  }
  for (const s of placed) for (const id of s.needs) {
    const from = byID.get(id), active = facts.activeSteps.includes(s.id);
    const color = active ? COLORS.leased : s.state === 'FAILED' ? COLORS.failed : s.state === 'DONE' && from.state === 'DONE' ? COLORS.idle : '#53687B';
    flowLink(links, compact ? { x: from.x + 100, y: from.y + 150 } : { x: from.x + 200, y: from.y + 75 }, compact ? { x: s.x + 100, y: s.y } : { x: s.x, y: s.y + 75 }, { vertical: compact, active, color });
  }
  root.append(links);
  root.append(svg('rect', { x: 40, y: zoneY + 10, width: 185, height: 52, fill: '#101C26' }));
  label(root, 46, zoneY + 29, '03 / ENODE', { fill: '#A3B9CA', 'font-size': 11, 'letter-spacing': 2 });
  label(root, 46, zoneY + 52, '실제 실행 단계', { 'font-size': 16, 'font-weight': 600 });
  label(root, width - 46, zoneY + 29, `${steps.length} STEPS`, { 'text-anchor': 'end', fill: '#819AAD', 'font-size': 10 });
  const legendY = compact ? 526 : 290;
  root.append(svg('rect', { x: mediator.x - 8, y: legendY - 16, width: 286, height: 24, rx: 6, fill: '#101A24' }));
  label(root, mediator.x + 135, legendY, stale ? '관측 갱신 지연 · 마지막 상태' : facts.description, { 'text-anchor': 'middle', fill: stale ? COLORS.asked : '#9AB1C3', 'font-size': 11, class: 'flow-description' });
  flowActor(root, { ...guest, kind: 'guest', name: run.submitter === undefined ? '제출자 미확인' : run.submitter || '제출자 미제공', status: '요청한 사람', description: '이 작업의 시작점', color: '#87CFC4', iso, mode, selected: actor, onSelect: onActorSelect, active: false });
  flowActor(root, { ...mediator, kind: 'mediator', name: 'Mediator', status: facts.label, description: '요청 접수 · 노드 배정 · 결과 확인', color: COLORS[facts.tone], iso, mode, selected: actor, onSelect: onActorSelect, active: facts.activeSteps.length > 0 });
  for (const s of placed) {
    const color = s.state === 'ASKED' ? COLORS.asked : s.state === 'FAILED' ? COLORS.failed : s.state === 'DONE' ? COLORS.idle : s.state === 'CLAIMED' ? COLORS.leased : COLORS.expiring;
    const group = svg('g', { transform: `translate(${s.x} ${s.y})`, opacity: s.state === 'SKIPPED' && !s.chosen ? .5 : 1, class: 'run-step' });
    const node = nodes.find(n => n.node_id === s.node), nodeLabel = node?.label || s.node || '노드 미배정';
    const stepTitle = stepDisplayName(run.run_id, s.id);
    selectable(group, `${s.seq}. ${stepTitle} · ${s.state} · ${nodeLabel}`, `${mode}-step-select-button`, 'step', s.id, () => onSelect(s.id), selected === s.id);
    group.append(svg('title', {}, `${s.seq}. ${stepTitle}\n단계 ID: ${s.id}\n${s.state} · ${s.uses}\n실행 노드: ${nodeLabel}`));
    if (iso) group.append(svg('path', { d: 'M 8 150 H 200 V 8 L 210 18 V 160 H 18 Z', fill: '#0B1723', stroke: '#34495E' }));
    group.append(svg('rect', { width: 200, height: 150, rx: 12, fill: '#192B38', stroke: selected === s.id ? color : '#486074', 'stroke-width': selected === s.id ? 3 : 1.5 }));
    group.append(svg('path', { d: 'M 14 53 H 186', stroke: '#34495E' }));
    label(group, 14, 22, `STEP ${String(s.seq).padStart(2, '0')}`, { fill: '#8DA9BC', 'font-size': 9, 'letter-spacing': 1.5 });
    label(group, 14, 42, wrapLabel(stepTitle, 23, 1)[0], { 'font-size': 13, 'font-weight': 600 });
    label(group, 14, 76, s.state, { fill: color, 'font-size': 12 });
    wrapLabel(nodeLabel, 26, 2).forEach((line, i) => label(group, 14, 100 + i * 15, line, { fill: '#C8D6DF', 'font-size': 11 }));
    label(group, 14, 135, wrapLabel(stepDisplayName(run.run_id, s.uses) || '용도 미제공', 29, 1)[0], { fill: '#92ACBF', 'font-size': 10 });
    if (s.state === 'SKIPPED') label(group, 0, 184, s.chosen ? '선택됨 · 도달하지 못함' : '선택하지 않은 경로', { fill: '#98A4B3', 'font-size': 10 });
    root.append(group);
  }
  if (!steps.length) {
    flowLink(root, { x: mediator.x + 135, y: mediator.y + 168 }, { x: width / 2, y: zoneY }, { vertical: true, context: true });
    label(root, width / 2, zoneY + 124, run.state === 'QUEUED' ? '배정할 enode를 기다리고 있어요' : '관측된 실행 단계가 없습니다', { 'text-anchor': 'middle', fill: '#C3D2DE', 'font-size': 14 });
    label(root, width / 2, zoneY + 151, '작업 정보에서 현재 상태를 확인하세요', { 'text-anchor': 'middle', fill: '#8FA7B9', 'font-size': 11 });
  }
  return { element: root, width, height, compact };
}
