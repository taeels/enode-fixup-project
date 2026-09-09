import { COLORS, graphLayout, nodeFacts, runFlowFacts } from './model.mjs';
import { leaseIdentity, wrapLabel, formatRunId } from './format.mjs';
import { nodeIdentity, executionIdentity, runTheme } from './identity.mjs';

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
// 연결 보드는 호스트 종류가 아니다. 모델과 동일한 호스트 사실을 사용한다.
export function machineKind(node) {
  return nodeIdentity(node).kind;
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
  } else if (kind === 'vm') {
    parent.append(svg('rect', { x: -42, y: -30, width: 84, height: 62, rx: 8, fill: '#111B25', stroke: color, 'stroke-dasharray': '4 4' }));
    parent.append(svg('rect', { x: -30, y: -19, width: 60, height: 40, rx: 5, fill: '#243440', stroke: color }));
    label(parent, 0, 7, 'VM', { fill: color, 'text-anchor': 'middle', 'font-size': 17, 'font-weight': 700 });
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
export function fleetScene({ nodes, runs = [], details, asks, now, iso, compact = false, mode, selected, onSelect }) {
  const columns = compact ? 1 : Math.max(2, Math.ceil(Math.sqrt(nodes.length))), rows = Math.ceil(nodes.length / columns);
  const gapX = 310, gapY = 190;
  const width = compact ? 360 : iso ? Math.max(990, (columns + rows - 2) * gapX + 370) : Math.max(760, columns * 360 + 40);
  const height = compact ? Math.max(460, nodes.length * (iso ? 370 : 300) + 65) : iso ? Math.max(740, (columns + rows - 2) * gapY + 460) : Math.max(640, rows * 300 + 60);
  const originX = (width - (columns - rows) * gapX) / 2;
  const root = svg('svg', { width, height, viewBox: `0 0 ${width} ${height}`, class: 'fleet-identity', 'aria-label': '함대 관측', role: 'group' });
  if (iso && !compact) {
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
    const x = compact ? iso ? 180 : 30 : iso ? originX + (col - row) * gapX : 30 + col * 360;
    const y = compact ? iso ? 140 + index * 370 : 30 + index * 300 : iso ? 180 + (col + row) * gapY : 40 + row * 300;
    const facts = nodeFacts(node, details, asks, now), color = COLORS[facts.tone];
    const device = nodeIdentity(node), theme = device.theme;
    const owner = leaseIdentity(node, runs), status = `${facts.label}${owner ? ` · ${owner.submitter}` : ''}`;
    const group = svg('g', { transform: `translate(${x} ${y})`, opacity: facts.expiring ? .65 : 1, 'data-host-key': device.hostKey });
    selectable(group, `${device.owner} · ${device.host} · ${device.device} · ${device.role} · ${status}`, `${mode}-node-select-button`, 'node', node.node_id, () => onSelect(node.node_id), selected === node.node_id);
    group.append(svg('title', {}, `${device.label || node.node_id}\n${device.device}\n${device.role}\n${device.attachments.join('\n')}\n${status}${owner ? `\n작업: ${owner.runId}` : ''}`));
    if (iso) {
      group.append(svg('path', { d: 'M -100 30 L 0 -20 L 100 30 L 0 80 Z', fill: theme.surface, stroke: theme.accent, 'stroke-width': selected === node.node_id ? 3 : 1.5, class: 'host-surface' }));
      group.append(svg('path', { d: 'M -100 30 L 0 80 L 100 30 L 100 42 L 0 92 L -100 42 Z', fill: '#10151B' }));
      machine(group, device.kind, theme.accent);
      label(group, 0, -104, wrapLabel(`소유자 · ${device.owner}`, 36, 1)[0], { 'text-anchor': 'middle', fill: theme.accent, 'font-size': 14, 'font-weight': 600, class: 'node-owner' });
      wrapLabel(device.host, 32, 2).forEach((line, i) => label(group, 0, -81 + i * 18, line, { 'text-anchor': 'middle', 'font-size': 14, 'font-weight': 600, class: 'node-name' }));
      group.append(svg('rect', { x: -145, y: 100, width: 290, height: 126, rx: 12, fill: theme.surface, stroke: theme.edge }));
      label(group, -129, 124, wrapLabel(device.device, 38, 1)[0], { fill: theme.accent, 'font-size': 12, class: 'node-device' });
      label(group, -129, 147, wrapLabel(`기능 · ${device.role}`, 38, 1)[0], { 'font-size': 12, class: 'node-role' });
      label(group, -129, 170, wrapLabel(device.attachments.join(' · ') || (device.config ? `enode · ${device.config}` : 'enode 실행 장비'), 42, 1)[0], { fill: '#A9B8C7', 'font-size': 11 });
      group.append(svg('circle', { cx: -126, cy: 198, r: 4, fill: color }));
      label(group, -114, 202, wrapLabel(status, 36, 1)[0], { fill: color, 'font-size': 11 });
    } else {
      group.append(svg('rect', { width: 300, height: 250, rx: 12, fill: theme.surface, stroke: selected === node.node_id ? theme.accent : theme.edge, 'stroke-width': 2, class: 'host-surface' }));
      group.append(svg('path', { d: 'M 18 1 H 282', stroke: theme.accent, 'stroke-width': 3 }));
      label(group, 18, 29, wrapLabel(`소유자 · ${device.owner}`, 38, 1)[0], { fill: theme.accent, 'font-size': 13, 'font-weight': 600, class: 'node-owner' });
      wrapLabel(device.host, 32, 2).forEach((line, i) => label(group, 18, 55 + i * 19, line, { 'font-size': 15, 'font-weight': 600, class: 'node-name' }));
      label(group, 18, 101, wrapLabel(device.device, 36, 1)[0], { fill: theme.accent, 'font-size': 12, class: 'node-device' });
      wrapLabel(`기능 · ${device.role}`, 37, 2).forEach((line, i) => label(group, 18, 127 + i * 18, line, { 'font-size': 12, class: 'node-role' }));
      label(group, 18, 170, wrapLabel(device.attachments.join(' · ') || (device.config ? `enode · ${device.config}` : 'enode 실행 장비'), 40, 1)[0], { fill: '#A9B8C7', 'font-size': 11 });
      group.append(svg('path', { d: 'M 18 189 H 282', stroke: theme.edge }));
      group.append(svg('circle', { cx: 22, cy: 212, r: 4, fill: color }));
      label(group, 34, 216, facts.label, { fill: color, 'font-size': 12 });
      label(group, 18, 236, owner ? wrapLabel(`요청자 · ${owner.submitter}`, 40, 1)[0] : '현재 임대 없음', { fill: '#A9B8C7', 'font-size': 11 });
    }
    root.append(group);
  });
  return { element: root, width, height, compact };
}
function flowLink(parent, from, to, { vertical = false, context = false, active = false, color = '#53687B' } = {}) {
  const end = vertical ? { x: to.x, y: to.y - 8 } : { x: to.x - 8, y: to.y };
  const bend = vertical ? Math.max(28, (to.y - from.y) / 2) : Math.max(28, (to.x - from.x) / 2);
  const controls = vertical ? `${from.x} ${from.y + bend}, ${end.x} ${end.y - bend}` : `${from.x + bend} ${from.y}, ${end.x - bend} ${end.y}`;
  parent.append(svg('path', { d: `M ${from.x} ${from.y} C ${controls}, ${end.x} ${end.y}`, fill: 'none', stroke: color, 'stroke-width': active ? 2.5 : 1.8, class: context ? 'flow-link flow-context-link' : 'flow-link dependency-edge', 'data-active': active }));
  parent.append(svg('path', { d: vertical ? `M ${to.x - 4} ${to.y - 10} l 4 7 l 4 -7` : `M ${to.x - 10} ${to.y - 4} l 7 4 l -7 4`, fill: 'none', stroke: color, 'stroke-width': 1.8 }));
}

function flowActor(parent, { x, y, kind, name, status, description, color, statusColor = color, iso, mode, selected, onSelect, active }) {
  const group = svg('g', { transform: `translate(${x} ${y})`, class: `flow-actor${active ? ' is-active' : ''}` });
  const title = kind === 'guest' ? mode === 'demo' ? 'Guest' : 'Submitter' : 'Mediator';
  selectable(group, `${title} · ${name} · ${status}`, `${mode}-flow-actor-button`, 'actor', kind, () => onSelect(kind), selected === kind);
  group.append(svg('title', {}, `${title}\n${name}\n${status}\n${description}`));
  if (iso) group.append(svg('path', { d: 'M 8 168 H 270 V 8 L 282 20 V 180 H 20 Z', fill: '#101C26', stroke: '#3B5063' }));
  group.append(svg('rect', { width: 270, height: 168, rx: 16, fill: kind === 'guest' ? '#172B2B' : '#2A261C', stroke: selected === kind ? color : kind === 'guest' ? '#3B6560' : '#6A583C', 'stroke-width': selected === kind ? 3 : 1.5 }));
  label(group, 20, 27, `${kind === 'guest' ? '01' : '02'} / ${title.toUpperCase()}`, { fill: color, 'font-size': 11, 'letter-spacing': 2 });
  group.append(svg('circle', { cx: 45, cy: 74, r: 25, fill: '#0D1A23', stroke: color, 'stroke-opacity': .5, class: 'flow-halo' }));
  if (kind === 'guest') {
    group.append(svg('circle', { cx: 45, cy: 67, r: 7, fill: '#9BD8D0' }));
    group.append(svg('path', { d: 'M 31 87 Q 31 76 45 76 Q 59 76 59 87', fill: '#9BD8D0' }));
  } else {
    group.append(svg('path', { d: 'M 45 56 L 60 65 L 60 83 L 45 92 L 30 83 L 30 65 Z M 30 65 L 45 74 L 60 65 M 45 74 V 92 M 45 56 V 64', fill: '#173449', stroke: color, 'stroke-width': 1.8 }));
  }
  wrapLabel(name, 20, 2).forEach((line, i) => label(group, 84, 69 + i * 19, line, { 'font-size': 15, 'font-weight': 600 }));
  group.append(svg('circle', { cx: 24, cy: 116, r: 3, fill: statusColor }));
  label(group, 35, 120, status, { fill: statusColor, 'font-size': 12 });
  label(group, 20, 146, wrapLabel(description, 39, 1)[0], { fill: '#B2C0CD', 'font-size': 11 });
  parent.append(group);
}

export function runScene({ steps = [], nodes = [], run = {}, stale = false, compact = false, iso, mode, selected, onSelect, actor, onActorSelect }) {
  const facts = runFlowFacts(run.state, steps, stale);
  const theme = runTheme(run.run_id), cardWidth = 300, cardHeight = 270;
  const placed = graphLayout(steps).map(s => {
    const level = (s.x - 65) / 240, lane = (s.y - 70) / 180;
    return { ...s, device: executionIdentity(s, nodes, run), x: compact ? 30 + lane * 360 : 50 + level * 390 + (iso ? lane * 24 : 0), y: compact ? 674 + level * 350 : 425 + lane * 320 + (iso ? level * 28 : 0) };
  });
  const byID = new Map(placed.map(s => [s.id, s]));
  const width = Math.max(compact ? 360 : 840, ...placed.map(s => s.x + cardWidth + (compact ? 30 : 50)));
  const height = Math.max(compact ? 994 : 760, ...placed.map(s => s.y + cardHeight + 60));
  const guest = { x: compact ? (width - 270) / 2 : 55, y: compact ? 55 : 76 };
  const mediator = { x: compact ? guest.x : 485, y: compact ? 310 : 76 };
  const zoneY = compact ? 570 : 326;
  const root = svg('svg', { width, height, viewBox: `0 0 ${width} ${height}`, class: 'run-flow', 'aria-label': '요청자, Mediator, enode 실행 흐름과 단계 의존 그래프', role: 'group', 'data-paused': stale });
  root.append(svg('desc', {}, '카드마다 실행 호스트와 소유자, 역할을 표시합니다. 같은 호스트는 같은 장비색입니다. 점선은 요청과 배정, 실선은 실제 단계의 의존 관계입니다.'));
  label(root, 30, 28, wrapLabel(`RUN / ${formatRunId(run.run_id)}`, compact ? 41 : 100, 1)[0], { fill: theme.accent, 'font-size': 12, class: 'run-identity' });
  root.append(svg('path', { d: `M 30 40 H ${width - 30}`, stroke: theme.accent, 'stroke-opacity': .45 }));
  if (compact) {
    flowLink(root, { x: guest.x + 135, y: guest.y + 168 }, { x: mediator.x + 135, y: mediator.y }, { vertical: true, context: true, color: '#6BAEA6' });
    label(root, guest.x + 153, 269, '요청 접수', { fill: '#93BEBB', 'font-size': 11 });
  } else {
    flowLink(root, { x: guest.x + 270, y: guest.y + 84 }, { x: mediator.x, y: mediator.y + 84 }, { context: true, color: '#6BAEA6' });
    label(root, 405, 142, '요청 접수', { 'text-anchor': 'middle', fill: '#93BEBB', 'font-size': 11 });
  }
  const links = svg('g', { class: 'run-connections' });
  for (const s of placed.filter(s => !s.needs.length)) {
    flowLink(links, { x: mediator.x + 135, y: mediator.y + 168 }, { x: s.x + cardWidth / 2, y: s.y }, { vertical: true, context: true, active: facts.activeSteps.includes(s.id), color: facts.activeSteps.includes(s.id) ? theme.accent : '#526E84' });
  }
  for (const s of placed) for (const id of s.needs) {
    const from = byID.get(id), active = facts.activeSteps.includes(s.id);
    const color = active ? theme.accent : s.state === 'FAILED' ? COLORS.failed : '#758899';
    flowLink(links, compact ? { x: from.x + cardWidth / 2, y: from.y + cardHeight } : { x: from.x + cardWidth, y: from.y + cardHeight / 2 }, compact ? { x: s.x + cardWidth / 2, y: s.y } : { x: s.x, y: s.y + cardHeight / 2 }, { vertical: compact, active, color });
    const relation = from.device.assigned && s.device.assigned && from.device.hostKnown && s.device.hostKnown ? from.device.hostKey === s.device.hostKey ? '같은 호스트' : '다른 호스트' : '다음 단계';
    if (s.needs.length === 1 && placed.filter(next => next.needs.includes(from.id)).length === 1) {
      label(links, compact ? s.x + cardWidth / 2 + 12 : (from.x + cardWidth + s.x) / 2, compact ? (from.y + cardHeight + s.y) / 2 : from.y + cardHeight / 2 - 14, relation, { fill: '#BCCCDC', 'text-anchor': compact ? 'start' : 'middle', 'font-size': 10, class: 'host-transfer' });
    }
  }
  root.append(links);
  const hostCount = new Set(placed.filter(s => s.device.assigned && s.device.hostKnown).map(s => s.device.hostKey)).size;
  root.append(svg('rect', { x: 24, y: zoneY + 10, width: width - 48, height: 66, rx: 8, fill: '#101A24' }));
  label(root, 30, zoneY + 29, '03 / EXECUTION HOSTS', { fill: '#A3B9CA', 'font-size': 11, 'letter-spacing': 1.2 });
  label(root, 30, zoneY + 55, '장비별 실행 단계', { 'font-size': 17, 'font-weight': 600 });
  label(root, width - 30, zoneY + 55, `${hostCount ? `${hostCount}개 호스트 확인` : '호스트 미확인'} · ${steps.length}단계`, { 'text-anchor': 'end', fill: theme.accent, 'font-size': 10 });
  const legendY = compact ? 526 : 290;
  label(root, mediator.x + 135, legendY, stale ? '관측 갱신 지연 · 마지막 상태' : facts.description, { 'text-anchor': 'middle', fill: stale ? COLORS.asked : '#9AB1C3', stroke: '#101A24', 'stroke-width': 4, 'stroke-linejoin': 'round', 'paint-order': 'stroke fill', 'font-size': 11, class: 'flow-description' });
  flowActor(root, { ...guest, kind: 'guest', name: run.submitter === undefined ? '제출자 미확인' : run.submitter || '제출자 미제공', status: '브라우저에서 요청', description: '실행할 일만 요청합니다', color: '#87CFC4', iso, mode, selected: actor, onSelect: onActorSelect, active: false });
  flowActor(root, { ...mediator, kind: 'mediator', name: 'Mediator', status: facts.label, description: '서버 · 장비 배정과 결과 수집', color: '#D7B67A', statusColor: COLORS[facts.tone], iso, mode, selected: actor, onSelect: onActorSelect, active: facts.activeSteps.length > 0 });
  for (const s of placed) {
    const color = s.state === 'ASKED' ? COLORS.asked : s.state === 'FAILED' ? COLORS.failed : s.state === 'DONE' ? COLORS.idle : s.state === 'CLAIMED' ? COLORS.leased : COLORS.expiring;
    const device = s.device, hostTheme = device.theme;
    const group = svg('g', { transform: `translate(${s.x} ${s.y})`, opacity: s.state === 'SKIPPED' && !s.chosen ? .5 : 1, class: 'run-step', 'data-host-key': device.hostKey });
    selectable(group, `${s.seq}. ${s.id} · ${s.state} · ${device.owner} · ${device.host} · ${device.device} · ${device.role}`, `${mode}-step-select-button`, 'step', s.id, () => onSelect(s.id), selected === s.id);
    group.append(svg('title', {}, `${s.seq}. ${s.id}\n${s.state} · ${s.uses}\n소유자: ${device.owner}\n호스트: ${device.host}\n${device.device}\n역할: ${device.role}\n${device.attachments.join('\n')}\n${device.label}`));
    if (iso) group.append(svg('path', { d: 'M 8 270 H 300 V 8 L 310 18 V 280 H 18 Z', fill: '#0B1723', stroke: hostTheme.edge }));
    group.append(svg('rect', { width: cardWidth, height: cardHeight, rx: 14, fill: hostTheme.surface, stroke: selected === s.id ? hostTheme.accent : hostTheme.edge, 'stroke-width': selected === s.id ? 3 : 1.5, class: 'host-surface' }));
    group.append(svg('path', { d: 'M 18 1 H 282', stroke: hostTheme.accent, 'stroke-width': 3 }));
    label(group, 18, 23, `ENODE / STEP ${String(s.seq).padStart(2, '0')}`, { fill: hostTheme.accent, 'font-size': 10, 'letter-spacing': 1 });
    label(group, 282, 23, s.state, { fill: color, 'font-size': 11, 'text-anchor': 'end', class: 'step-status' });
    label(group, 18, 49, wrapLabel(s.id, 31, 1)[0], { 'font-size': 17, 'font-weight': 600 });
    group.append(svg('path', { d: 'M 18 63 H 282', stroke: hostTheme.edge }));
    label(group, 18, 85, wrapLabel(device.assigned ? `소유자 · ${device.owner}` : '실행 장비 미배정', 39, 1)[0], { fill: hostTheme.accent, 'font-size': 12, class: 'node-owner' });
    wrapLabel(device.host, 32, 2).forEach((line, i) => label(group, 18, 108 + i * 20, line, { 'font-size': 15, 'font-weight': 600, class: 'node-name' }));
    label(group, 18, 153, wrapLabel(device.device, 38, 1)[0], { fill: '#B9C9D8', 'font-size': 12, class: 'node-device' });
    wrapLabel(`역할 · ${device.role}`, 32, 2).forEach((line, i) => label(group, 18, 184 + i * 18, line, { 'font-size': 12, class: 'node-role' }));
    const icon = svg('g', { transform: 'translate(264 188) scale(.4)' }); machine(icon, device.kind, hostTheme.accent); group.append(icon);
    label(group, 18, 232, wrapLabel(device.attachments.join(' · ') || (device.environment ? `환경 · ${device.environment}` : device.config ? `enode · ${device.config}` : s.uses || '역할 미제공'), 41, 1)[0], { fill: '#A9B8C7', 'font-size': 11 });
    label(group, 18, 253, device.assigned ? device.observed ? '장비에서 실행 · 결과는 enode로 전달' : '지난 배정 · 현재 장비 정보 미관측' : '실행 가능한 enode를 기다립니다', { fill: '#93A5B6', 'font-size': 10 });
    if (s.state === 'SKIPPED') label(group, 0, 305, s.chosen ? '선택됨 · 도달하지 못함' : '선택하지 않은 경로', { fill: '#98A4B3', 'font-size': 10 });
    root.append(group);
  }
  if (!steps.length) {
    flowLink(root, { x: mediator.x + 135, y: mediator.y + 168 }, { x: width / 2, y: zoneY }, { vertical: true, context: true });
    label(root, width / 2, zoneY + 124, run.state === 'QUEUED' ? '배정할 enode를 기다리고 있어요' : '관측된 실행 단계가 없습니다', { 'text-anchor': 'middle', fill: '#C3D2DE', 'font-size': 14 });
    label(root, width / 2, zoneY + 151, '작업 정보에서 현재 상태를 확인하세요', { 'text-anchor': 'middle', fill: '#8FA7B9', 'font-size': 11 });
  }
  return { element: root, width, height, compact };
}
