import { COLORS, graphLayout, nodeFacts } from './model.mjs';
import { leaseIdentity } from './format.mjs';

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
const shorten = (text, length = 24) => text.length > length ? `${text.slice(0, length - 1)}…` : text;
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
  const width = iso ? Math.max(880, (columns + rows) * 130 + 140) : Math.max(760, columns * 260 + 60);
  const height = iso ? Math.max(620, (columns + rows) * 76 + 240) : Math.max(460, rows * 170 + 80);
  const root = svg('svg', { width, height, viewBox: `0 0 ${width} ${height}`, 'aria-label': '함대 관측', role: 'group' });
  if (iso) {
    const floor = svg('g', { stroke: '#262D36', 'stroke-width': 1, opacity: .6 });
    const cx = width / 2, top = 75, span = Math.max(columns, rows) + 1;
    for (let i = 0; i <= span; i++) {
      floor.append(svg('path', { d: `M ${cx - i * 120} ${top + i * 66} l ${span * 120} ${span * 66}`, fill: 'none' }));
      floor.append(svg('path', { d: `M ${cx + i * 120} ${top + i * 66} l ${-span * 120} ${span * 66}`, fill: 'none' }));
    }
    root.append(floor);
  }
  nodes.forEach((node, index) => {
    const col = index % columns, row = Math.floor(index / columns);
    const x = iso ? width / 2 + (col - row) * 125 : 30 + col * 260;
    const y = iso ? 170 + (col + row) * 78 : 40 + row * 170;
    const facts = nodeFacts(node, details, asks, now), color = COLORS[facts.tone];
    const owner = leaseIdentity(node, runs), status = `${facts.label}${owner ? ` · ${owner.submitter}` : ''}`;
    const group = svg('g', { transform: `translate(${x} ${y})`, opacity: facts.expiring ? .65 : 1 });
    selectable(group, `${node.label || node.node_id} · ${status}`, `${mode}-node-select-button`, 'node', node.node_id, () => onSelect(node.node_id), selected === node.node_id);
    group.append(svg('title', {}, `${node.label || node.node_id}\n${status}${owner ? `\n작업: ${owner.runId}` : ''}`));
    if (iso) {
      group.append(svg('path', { d: 'M -100 30 L 0 -20 L 100 30 L 0 80 Z', fill: '#1B2027', stroke: color, 'stroke-width': selected === node.node_id ? 3 : 1.5 }));
      group.append(svg('path', { d: 'M -100 30 L 0 80 L 100 30 L 100 42 L 0 92 L -100 42 Z', fill: '#10151B' }));
      machine(group, machineKind(node), color);
      label(group, 0, -67, shorten(node.label || node.node_id), { 'text-anchor': 'middle', 'font-weight': 600 });
      label(group, 0, -48, shorten(status, 28), { 'text-anchor': 'middle', fill: color, 'font-size': 11 });
    } else {
      group.append(svg('rect', { width: 235, height: 143, rx: 10, fill: '#1B2027', stroke: selected === node.node_id ? color : '#39434F', 'stroke-width': 2 }));
      group.append(svg('circle', { cx: 17, cy: 25, r: 4, fill: color }));
      label(group, 30, 30, shorten(node.label || node.node_id), { 'font-weight': 600 });
      label(group, 15, 55, facts.label, { fill: color });
      label(group, 15, 80, node.capabilities.map(c => c.capability).join(' · ').slice(0, 28), { fill: '#98A4B3', 'font-size': 11 });
      label(group, 15, 106, `sandbox: ${[...new Set(node.capabilities.map(c => c.attrs.sandbox ?? '미제공'))].join(', ')}`, { fill: '#98A4B3', 'font-size': 11 });
      label(group, 15, 128, owner ? shorten(`작업 제출자 · ${owner.submitter}`, 28) : facts.expiring ? '광고 만료 임박 · 상세 확인' : '현재 임대 없음', { fill: '#98A4B3', 'font-size': 11 });
    }
    root.append(group);
  });
  return { element: root, width, height };
}
export function runScene({ steps, nodes = [], iso, mode, selected, onSelect }) {
  const placed = graphLayout(steps, iso), byID = new Map(placed.map(s => [s.id, s]));
  const width = Math.max(780, ...placed.map(s => s.x + 230)), height = Math.max(460, ...placed.map(s => s.y + 230));
  const root = svg('svg', { width, height, viewBox: `0 0 ${width} ${height}`, 'aria-label': '작업 의존 그래프', role: 'group' });
  for (const s of placed) for (const id of s.needs) {
    const from = byID.get(id), x = from.x + 175, y = from.y + 58, endX = s.x, endY = s.y + 58;
    root.append(svg('path', { d: `M ${x} ${y} C ${x + 35} ${y}, ${endX - 35} ${endY}, ${endX - 8} ${endY}`, fill: 'none', stroke: '#647386', 'stroke-width': 2 }));
    root.append(svg('path', { d: `M ${endX - 9} ${endY - 4} l 7 4 l -7 4`, fill: 'none', stroke: '#647386' }));
  }
  for (const s of placed) {
    const color = s.state === 'ASKED' ? COLORS.asked : s.state === 'FAILED' ? COLORS.failed : s.state === 'DONE' ? COLORS.idle : s.state === 'CLAIMED' ? COLORS.leased : COLORS.expiring;
    const group = svg('g', { transform: `translate(${s.x} ${s.y})`, opacity: s.state === 'SKIPPED' && !s.chosen ? .5 : 1 });
    const node = nodes.find(n => n.node_id === s.node), nodeLabel = node?.label || s.node || '노드 미배정';
    selectable(group, `${s.seq}. ${s.id} · ${s.state} · ${nodeLabel}`, `${mode}-step-select-button`, 'step', s.id, () => onSelect(s.id), selected === s.id);
    group.append(svg('title', {}, `${s.seq}. ${s.id}\n${s.state} · ${s.uses}\n실행 노드: ${nodeLabel}`));
    if (iso) {
      group.append(svg('path', { d: 'M 8 116 L 175 116 L 175 8 L 185 18 L 185 126 L 18 126 Z', fill: '#0B1723', stroke: '#34495E' }));
    }
    group.append(svg('rect', { width: 175, height: 116, rx: 8, fill: '#19232D', stroke: selected === s.id ? color : '#4B6074', 'stroke-width': selected === s.id ? 3 : 1.5 }));
    group.append(svg('path', { d: 'M 12 36 H 163', stroke: '#34495E' }));
    label(group, 12, 23, `${s.seq}  ${shorten(s.id, 17)}`, { 'font-size': 12, 'font-weight': 600 });
    label(group, 12, 56, s.state, { fill: color, 'font-size': 12 });
    label(group, 12, 76, shorten(s.uses || '용도 미제공', 23), { fill: '#B2C0CD', 'font-size': 10 });
    label(group, 12, 98, shorten(nodeLabel, 23), { fill: '#98A4B3', 'font-size': 10 });
    if (s.state === 'SKIPPED') label(group, 0, 148, s.chosen ? '선택됨 · 도달하지 못함' : '선택하지 않은 경로', { fill: '#98A4B3', 'font-size': 10 });
    root.append(group);
  }
  return { element: root, width, height };
}
