// 표시만 바꾸며 서버 값·상태·관측 시계는 보존한다.
const timestampFormat = new Intl.DateTimeFormat('ko-KR', {
  year: 'numeric', month: 'long', day: 'numeric', hour: 'numeric', minute: '2-digit', second: '2-digit',
});
export function formatTimestamp(value, timeZone) {
  if (!value || !Number.isFinite(Date.parse(value))) return '시각 미제공';
  const formatter = timeZone ? new Intl.DateTimeFormat('ko-KR', { ...timestampFormat.resolvedOptions(), timeZone }) : timestampFormat;
  return formatter.format(new Date(value));
}
export function formatDuration(seconds) {
  const n = Math.max(0, Math.ceil(seconds));
  if (n < 60) return `${n}초`;
  if (n < 3600) return `${Math.floor(n / 60)}분${n % 60 ? ` ${n % 60}초` : ''}`;
  if (n < 86400) return `${Math.floor(n / 3600)}시간${Math.floor(n % 3600 / 60) ? ` ${Math.floor(n % 3600 / 60)}분` : ''}`;
  return `${Math.floor(n / 86400)}일${Math.floor(n % 86400 / 3600) ? ` ${Math.floor(n % 86400 / 3600)}시간` : ''}`;
}
export function remainingTime(value, now) {
  if (now === null || !Number.isFinite(now) || !Number.isFinite(Date.parse(value))) return '남은 시간 미확인';
  const remaining = (Date.parse(value) - now) / 1000;
  return remaining > 0 ? `${formatDuration(remaining)} 남음` : '만료 시각 지남';
}
export function elapsedTime(value, now) {
  if (now === null || !Number.isFinite(now) || !Number.isFinite(Date.parse(value))) return '경과 시간 미확인';
  const elapsed = Math.floor((now - Date.parse(value)) / 1000);
  if (elapsed < 0) return '관측 기준 이후 시각';
  return elapsed < 1 ? '방금 전' : `${formatDuration(elapsed)} 전`;
}
export function leaseIdentity(node, runs = []) {
  if (!node.lease) return null;
  const run = runs.find(r => r.run_id === node.lease.run_id);
  return { runId: node.lease.run_id, submitter: run ? run.submitter || '제출자 미제공' : '제출자 미확인' };
}
const capabilityNames = {
  'agent.reason': 'AI 에이전트', 'shell.exec': '명령 실행', 'audio.play': '오디오 재생',
};
export const capabilityName = value => capabilityNames[value] || value;
const attributeNames = {
  harness: '실행 도구', host_arch: '프로세서', arch: '프로세서', lang: '언어', os: '운영체제',
  repo: '프로젝트', service: '서비스', tts: '음성 합성 도구', voice: '목소리',
  fmt_m4a: 'M4A 지원', fmt_mp3: 'MP3 지원', fmt_wav: 'WAV 지원',
  ws: '작업 폴더', greet: '인사 스크립트', sandbox: '실행 격리 (광고값)',
  board: '보드', device: '장치', device_type: '장치 유형', kind: '종류', model: '모델',
};
export const attributeName = key => attributeNames[key] || key;
export function attributeValue(key, value) {
  if (key.startsWith('fmt_') && ['yes', 'no'].includes(value)) return value === 'yes' ? '지원' : '미지원';
  if (key === 'os') return ({ darwin: 'macOS', windows: 'Windows', linux: 'Linux' })[value] || value;
  if (key === 'lang') return ({ ko: '한국어', en: '영어', ja: '일본어' })[value] || value;
  if (key === 'service' && value === 'tts') return '텍스트를 음성으로 변환';
  return value || '값 없음';
}
export const technicalAttribute = key => ['ws', 'greet'].includes(key);
export const drainDescription = value => ({ '': '새 작업 수락 중', graceful: '현재 작업 완료 후 회수 (graceful)', 'at-boundary': '현재 단계 완료 후 회수 (at-boundary)' })[value] || value;

// SVG의 고정 폭 영역 안에서 한글 등 전각 문자를 두 칸으로 계산한다.
export function wrapLabel(value, columns = 30, maxLines = 2) {
  const lines = ['']; let used = 0;
  const units = char => char.codePointAt(0) <= 127 ? 1 : 2;
  for (const char of value.replace(/\s+/g, ' ').trim()) {
    if (used + units(char) > columns) {
      if (lines.length === maxLines) {
        const last = [...lines.pop()];
        while (used > columns - 2) used -= units(last.pop());
        lines.push(last.join('') + '…'); return lines;
      }
      lines.push(''); used = 0;
    }
    lines[lines.length - 1] += char; used += units(char);
  }
  return lines;
}
