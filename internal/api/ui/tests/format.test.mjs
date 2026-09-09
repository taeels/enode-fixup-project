import test from 'node:test';
import assert from 'node:assert/strict';
import { formatTimestamp, formatDuration, remainingTime, elapsedTime, leaseIdentity, attributeName, attributeValue, capabilityName, wrapLabel } from '../static/shared/fleet/format.mjs';

test('lease submitter is joined only through the current lease, never historical assignments', () => {
  const runs = [{ run_id: 'past', submitter: 'old guest', assigned: [{ nodes: [{ node: 'node-1' }] }] }, { run_id: 'current', submitter: 'current guest' }];
  assert.equal(leaseIdentity({ node_id: 'node-1', lease: null }, runs), null);
  assert.deepEqual(leaseIdentity({ lease: { run_id: 'current' } }, runs), { runId: 'current', submitter: 'current guest' });
  assert.equal(leaseIdentity({ lease: { run_id: 'missing' } }, runs).submitter, '제출자 미확인');
  assert.equal(leaseIdentity({ lease: { run_id: 'missing' } }).submitter, '제출자 미확인');
  assert.equal(leaseIdentity({ lease: { run_id: 'empty' } }, [{ run_id: 'empty', submitter: '' }]).submitter, '제출자 미제공');
});

test('localized timestamps preserve the instant across offsets and day boundaries', () => {
  const source = '2026-09-09T00:47:54.207806+09:00';
  assert.equal(formatTimestamp(source, 'Asia/Seoul'), formatTimestamp('2026-09-08T15:47:54.207806Z', 'Asia/Seoul'));
  assert.match(formatTimestamp(source, 'Asia/Seoul'), /9월 9일/);
  assert.match(formatTimestamp(source, 'UTC'), /9월 8일/);
  assert.doesNotMatch(formatTimestamp(source, 'Asia/Seoul'), /207806|T00|\+09:00/);
  for (const value of [null, undefined, '', 'invalid']) assert.equal(formatTimestamp(value), '시각 미제공');
});

test('countdowns use supplied observation time and distinguish expired from unknown', () => {
  const value = '2026-09-09T04:48:24Z', now = Date.parse(value);
  assert.equal(remainingTime(value, now - 24000), '24초 남음');
  assert.equal(remainingTime(value, now - 61000), '1분 1초 남음');
  assert.equal(remainingTime(value, now), '만료 시각 지남');
  assert.equal(remainingTime(value, now + 1000), '만료 시각 지남');
  assert.equal(remainingTime(value, null), '남은 시간 미확인');
  assert.equal(remainingTime('invalid', now), '남은 시간 미확인');
  assert.equal(elapsedTime(value, now), '방금 전');
  assert.equal(elapsedTime(value, now + 60000), '1분 전');
  assert.equal(elapsedTime(value, now - 1000), '관측 기준 이후 시각');
});

test('durations carry into minutes, hours and days', () => {
  assert.equal(formatDuration(59.5), '1분');
  assert.equal(formatDuration(3600), '1시간');
  assert.equal(formatDuration(3660), '1시간 1분');
  assert.equal(formatDuration(90000), '1일 1시간');
});

test('only known attribute meanings are translated and sandbox evidence stays verbatim', () => {
  assert.equal(attributeValue('os', 'darwin'), 'macOS');
  assert.equal(attributeValue('lang', 'ko'), '한국어');
  assert.equal(attributeValue('fmt_mp3', 'yes'), '지원');
  assert.equal(attributeValue('sandbox', 'os'), 'os');
  assert.equal(attributeValue('sandbox', 'yes'), 'yes');
  assert.equal(attributeValue('custom', '<script>text</script>'), '<script>text</script>');
  assert.equal(attributeName('custom'), 'custom');
  assert.equal(capabilityName('custom.capability'), 'custom.capability');
});

test('long host labels wrap within two lines while wide characters consume more room', () => {
  assert.deepEqual(wrapLabel('worker@MacBook-Pro.local'), ['worker@MacBook-Pro.local']);
  const host = 'operator@long-development-workstation.local';
  assert.equal(wrapLabel(host).length, 2);
  assert.equal(wrapLabel(host).join(''), host);
  assert.deepEqual(wrapLabel('가나다라마바사아자차카타파하가나다라마바사'), ['가나다라마바사아자차카타파하가', '나다라마바사']);
  assert.deepEqual(wrapLabel('x'.repeat(100)), ['x'.repeat(30), 'x'.repeat(28) + '…']);
  assert.deepEqual(wrapLabel('  worker\n name\t '), ['worker name']);
});
