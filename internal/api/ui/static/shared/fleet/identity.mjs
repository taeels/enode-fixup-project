import { capabilityName } from './format.mjs';

// 작업/호스트의 정체성 색은 상태나 배열 순서가 바뀌어도 유지한다.
export function identityTheme(key) {
  if (!key) return { accent: '#8D9BAA', surface: '#1B252E', edge: '#3A4856' };
  let hash = 2166136261;
  for (const char of key) hash = Math.imul(hash ^ char.codePointAt(0), 16777619) >>> 0;
  const hue = hash % 360;
  return { accent: `hsl(${hue} 65% 74%)`, surface: `hsl(${hue} 26% 16%)`, edge: `hsl(${hue} 32% 34%)` };
}
export const runTheme = id => identityTheme(id ? `run:${id}` : '');
const values = (node, key) => [...new Set((node?.capabilities || []).map(c => c.attrs?.[key]).filter(v => typeof v === 'string' && v.trim()))];
const single = (node, key) => { const all = values(node, key); return all.length === 1 ? all[0] : ''; };
const prettyOS = os => ({ windows: 'Windows PC', darwin: 'macOS', linux: 'Linux' })[os] || os;
const prettyCPU = cpu => ({ amd64: 'x64', arm64: 'ARM64', arm: 'ARM' })[cpu] || cpu;

export function nodeIdentity(node, { label = '', id = '' } = {}) {
  label = node?.label || label; id = node?.node_id || id;
  // identity.go의 표시 규약이다. 권한/매칭/물리 장비 종류에는 사용하지 않는다.
  const parts = /^([^@\s]+)@([^:]+)(?::(.+))?$/.exec(label);
  const owner = single(node, 'owner') || parts?.[1] || '소유자 미확인';
  const host = single(node, 'hostname') || parts?.[2] || label || (id ? `노드 ${id}` : '실행 장비 배정 대기');
  const hostKey = single(node, 'hostname') || parts?.[2] || id;
  const os = single(node, 'os'), cpu = single(node, 'host_arch');
  const type = single(node, 'device_type') || single(node, 'kind');
  const sandbox = single(node, 'sandbox');
  const vm = ['vm', 'virtual-machine'].includes(type) || ['lima-vm', 'vm'].includes(sandbox);
  const physicalBoard = ['board', 'raspberry-pi', 'raspberry pi', 'rpi'].includes(type);
  const kind = vm ? 'vm' : physicalBoard ? 'board' : ['workstation', 'desktop', 'pc', 'laptop'].includes(type) || os === 'windows' || os === 'darwin' ? 'workstation' : 'generic';
  const platform = vm ? `${prettyOS(os) || '가상 장비'} VM` : physicalBoard ? '보드 호스트' : prettyOS(os);
  const device = [platform, prettyCPU(cpu)].filter(Boolean).join(' · ') || '장비 정보 미관측';
  const roles = [];
  if (values(node, 'tts').length) roles.push('음성 합성');
  if (values(node, 'device').includes('speaker') || values(node, 'audio_playback').includes('true') || node?.capabilities?.some(c => c.capability === 'audio.play')) roles.push('오디오 재생');
  if (values(node, 'device').includes('led')) roles.push('LED 제어');
  if (node?.capabilities?.some(c => c.capability === 'agent.reason')) roles.push(single(node, 'provider') === 'bedrock' ? 'Bedrock Claude' : single(node, 'harness') === 'claude' ? 'Claude' : 'AI 에이전트');
  if (node?.capabilities?.some(c => c.capability === 'shell.exec')) roles.push('명령 실행');
  const boards = values(node, 'board');
  const attachments = [...boards.map(board => `연결 보드 · ${board}`), ...values(node, 'device').filter(v => ['led', 'speaker'].includes(v)).map(v => v === 'led' ? '연결 장치 · LED' : '연결 장치 · 스피커')];
  return { owner, host, hostKey, hostKnown: !!(single(node, 'hostname') || parts), device, kind, label, config: parts?.[3] || '', environment: single(node, 'instance'),
    role: roles.join(' · ') || (node?.capabilities || []).map(c => capabilityName(c.capability)).join(' · ') || '기능 정보 미관측',
    attachments, theme: identityTheme(hostKey ? `host:${hostKey}` : ''), observed: !!node };
}

export function stepRole(step, requires = []) {
  const requirement = requires?.find(r => r.as === step.uses), attrs = requirement?.attrs || {};
  if (attrs.gallery === 'comments-v1') return 'Bedrock Claude · 댓글 작업';
  if (attrs.device === 'led') return 'LED 제어';
  if (attrs.device === 'speaker' || attrs.audio_playback === 'true' || requirement?.capability === 'audio.play') return '오디오 재생';
  if (attrs.tts || attrs.service === 'tts') return '음성 합성';
  if (attrs.arch) return '실행 파일 빌드';
  if (attrs.board) return '연결 보드 작업';
  return requirement ? capabilityName(requirement.capability) : step.uses || '역할 미제공';
}

export function executionIdentity(step, nodes = [], run = {}) {
  const node = step.node ? nodes.find(n => n.node_id === step.node) : null;
  const assigned = step.node ? (run.assigned || []).flatMap(a => a.nodes).find(n => n.node === step.node) : null;
  return { ...nodeIdentity(node, { label: assigned?.label, id: step.node }), node, assigned: !!step.node, role: stepRole(step, run.requires) };
}
