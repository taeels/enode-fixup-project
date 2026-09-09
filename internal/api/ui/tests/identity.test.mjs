import test from 'node:test';
import assert from 'node:assert/strict';
import { nodeIdentity, executionIdentity, runTheme, stepRole } from '../static/shared/fleet/identity.mjs';
const node = (label, attrs = {}, node_id = 'node-1') => ({ node_id, label, capabilities: [{ capability: 'agent.reason', attrs }] });

test('host identity separates owner, hostname and config without guessing a device from its label', () => {
  const device = nodeIdentity(node('operator@MacBook-RaspberryPi.local:work', { owner: 'declared-owner', os: 'windows', host_arch: 'amd64', board: 'rpi2b-v1.1', harness: 'claude' }));
  assert.equal(device.owner, 'declared-owner'); assert.equal(device.host, 'MacBook-RaspberryPi.local'); assert.equal(device.config, 'work');
  assert.equal(device.device, 'Windows PC · x64'); assert.equal(device.kind, 'workstation');
  assert.deepEqual(device.attachments, ['연결 보드 · rpi2b-v1.1']);
  const unknown = nodeIdentity(node('MacBook Pro', { harness: 'claude' }));
  assert.equal(unknown.kind, 'generic'); assert.equal(unknown.hostKnown, false); assert.equal(unknown.owner, '소유자 미확인');
});
test('host colors stay the same across users, enode configs and observation states', () => {
  const first = node('alice@workstation.local:one', { os: 'linux' });
  const second = node('bob@workstation.local:two', { os: 'linux' }, 'node-2');
  assert.deepEqual(nodeIdentity(first).theme, nodeIdentity(second).theme);
  first.draining = 'graceful'; first.lease = { run_id: 'a' };
  assert.deepEqual(nodeIdentity(first).theme, nodeIdentity(second).theme);
  assert.notDeepEqual(nodeIdentity(first).theme, nodeIdentity(node('alice@another-host.local')).theme);
  assert.deepEqual(runTheme('run-1'), runTheme('run-1')); assert.notDeepEqual(runTheme('run-1'), runTheme('run-2'));
});
test('VM, connected devices and target architecture remain different facts', () => {
  const vm = nodeIdentity(node('operator@lima-guest:bedrock', { sandbox: 'lima-vm', os: 'linux', host_arch: 'arm64', arch: 'amd64', harness: 'claude', provider: 'bedrock' }));
  assert.equal(vm.device, 'Linux VM · ARM64'); assert.equal(vm.kind, 'vm'); assert.equal(vm.role, 'Bedrock Claude');
  const audio = nodeIdentity(node('operator@audio-host', { os: 'darwin', tts: 'say-macos', device: 'speaker' }));
  assert.equal(audio.kind, 'workstation'); assert.match(audio.role, /음성 합성.*오디오 재생/); assert.deepEqual(audio.attachments, ['연결 장치 · 스피커']);
});
test('historical labels survive missing live nodes without inventing capabilities or assigning pending steps', () => {
  const run = { assigned: [{ as: 'tts', nodes: [{ node: 'past', label: 'author@past-host:voice' }] }], requires: [{ as: 'tts', capability: 'agent.reason', attrs: { tts: 'say-macos' } }] };
  const step = { id: 'make', uses: 'tts', node: 'past' }, before = structuredClone({ run, step });
  const past = executionIdentity(step, [], run);
  assert.equal(past.owner, 'author'); assert.equal(past.host, 'past-host'); assert.equal(past.role, '음성 합성');
  assert.equal(past.device, '장비 정보 미관측'); assert.equal(past.observed, false); assert.equal(past.hostKnown, true);
  const pending = executionIdentity({ id: 'make', uses: 'tts' }, [node('author@live-host')], run);
  assert.equal(pending.assigned, false); assert.equal(pending.hostKnown, false); assert.equal(pending.host, '실행 장비 배정 대기');
  assert.deepEqual({ run, step }, before);
});
test('roles follow requirements rather than step names, and conflicting host attributes are not guessed', () => {
  const requires = [{ as: 'speaker', capability: 'agent.reason', attrs: { audio_playback: 'true' } }, { as: 'gallery', capability: 'agent.reason', attrs: { gallery: 'comments-v1' } }];
  assert.equal(stepRole({ id: 'anything', uses: 'speaker' }, requires), '오디오 재생');
  assert.equal(stepRole({ id: 'play', uses: 'unknown' }, requires), 'unknown');
  assert.equal(stepRole({ id: 'gallery', uses: 'gallery' }, requires), 'Bedrock Claude · 댓글 작업');
  const conflicting = node('author@host', { os: 'windows' }); conflicting.capabilities.push({ capability: 'shell.exec', attrs: { os: 'linux' } });
  assert.equal(nodeIdentity(conflicting).device, '장비 정보 미관측');
});
