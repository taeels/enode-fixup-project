import { retryDelay, timeLimit, describeFailure, FREEZE_AFTER } from './client.mjs';

// 트랜스크립트 카드의 폴러. ObservationClient 밖의 타이머 하나다.
//
// 왜 밖인가 — CB4 가 재는 것이 「목록 폴링을 멈춰도 카드는 자란다」이고, 그
// 문장이 재는 것은 타이머가 둘이라는 구조다. ObservationClient 에는 자원마다의
// 간격이라는 개념이 없어서 (refresh 가 자원 전부를 한 번에 돈다) 카드를 그
// 안의 자원 종류로 더하면 간격 분기를 클라이언트 안에 만들게 되고, 그러면
// CB4 가 구현 세부를 재게 된다 (business-rules R46).
//
// 규율은 두 벌로 안 짓는다 — 429 · 제한시간 · 연속 실패 얼리기를 client.mjs
// 에서 가져다 쓴다 (R50). GET log 는 데모 모드에서 무인증 + 한도이므로
// (api.go 의 read) 카드도 같은 한도를 진다.
export const CARD_INTERVAL_MS = 2000;

// 방향이 한쪽이다 — 화면이 track 으로 말하고 폴러는 시각만 진다.
//
// 폴러가 상세 자원을 직접 읽으면 두 타이머가 서로를 보게 되고, CB4 가
// 「목록 폴링을 멈춘다」로 재려던 자리가 폴러 안으로 숨는다 (R47).
export class TranscriptPoller {
  // render 는 카드 렌더러 모듈이다 (internal/transcriptui/card.mjs). 임포트가
  // 아니라 주입인 이유 — 그 파일은 static 트리 밖에 살아 브라우저가 보는
  // URL 과 디스크 경로가 다르다. 정적 임포트를 적으면 브라우저에서는 서고
  // node --test 에서는 그 경로가 없어 터진다.
  constructor({ render, token = '', fetcher = (...args) => fetch(...args), now = () => performance.now(), wallNow = () => Date.now(), timers = globalThis, onChange = () => {} } = {}) {
    this.render = render; this.token = token; this.fetcher = fetcher; this.now = now; this.wallNow = wallNow; this.timers = timers; this.onChange = onChange;
    this.run = null; this.steps = []; this.cards = new Map(); this.retryAt = 0; this.stopped = false;
  }

  // track 은 화면이 그릴 때마다 부른다. steps 는 {seq, name, state} 의 열이다.
  //
  // name 이 steps[].id 인 것이 카드가 URL 을 만드는 법이다 (R42) — 상세의
  // 단계 객체에는 name 이 없고 id 가 곧 로그의 이름이다. 고정 문자열 "step"
  // 을 쓰면 서버가 200 에 0 바이트를 내고 카드가 빈 채로 선다.
  track(run, steps = []) {
    if (run !== this.run) { this.run = run; this.cards.clear(); }
    this.steps = steps.map(s => ({ seq: s.seq, name: s.name, state: s.state }));
    for (const seq of [...this.cards.keys()]) if (!this.steps.some(s => s.seq === seq)) this.cards.delete(seq);
  }

  card(seq) {
    if (!this.cards.has(seq)) {
      // changedAt 은 총 길이가 마지막으로 움직인 시각이다 (NC-6). 기간이
      // 아니라 시각인 것이 값이다 — 화면이 빼면 폴링이 죽어도 시계가 흐른다.
      this.cards.set(seq, { seq, events: [], total: null, source: '', attempt: null, capped: false, raw: '', rawOpen: false, open: new Set(), changedAt: null, receivedAt: null, error: '', failures: 0, frozen: false, done: false, missing: false });
    }
    return this.cards.get(seq);
  }

  setRaw(seq, on) { this.card(seq).rawOpen = on; if (!on) this.card(seq).raw = ''; }
  toggleOpen(seq, id, on) { const c = this.card(seq); if (on) c.open.add(id); else c.open.delete(id); }

  // due 는 이번 바퀴에 부를 단계를 고른다.
  //
  // 도는 단계(CLAIMED)만 2초로 돌고 끝난 단계는 한 번 읽고 멈춘다 (R49).
  // 한 Run 에서 동시에 도는 단계는 보통 하나이므로 2초 요청의 수가 단계
  // 수가 아니라 1 근처다.
  due() {
    return this.steps.filter(s => {
      const c = this.cards.get(s.seq);
      // frozen 은 낡음 표시이지 중단이 아니다 — 연속 실패가 폴링을 영영
      // 멈추면 잠깐 끊긴 중앙이 카드를 죽인다. ObservationClient 도 얼린
      // 뒤 계속 부른다.
      if (c?.done || c?.missing) return false;
      if (s.state === 'CLAIMED') return true;
      return !c || c.receivedAt === null; // 끝난 단계의 첫 한 번
    });
  }

  start() { this.interval = this.timers.setInterval(() => this.tick(), CARD_INTERVAL_MS); return this.tick(); }
  stop() {
    this.stopped = true; this.timers.clearInterval(this.interval);
    for (const c of this.cards.values()) c.controller?.abort();
    this.cards.clear(); this.run = null; this.steps = []; this.token = '';
  }

  // Run 상세가 안 열려 있으면 요청이 0 이다 (R51).
  async tick() {
    if (this.stopped || !this.run || this.now() < this.retryAt) return;
    await Promise.allSettled(this.due().map(s => this.request(s)));
  }

  async request(step) {
    const run = this.run, c = this.card(step.seq);
    if (c.pending) return;
    const controller = new AbortController();
    c.pending = true; c.controller = controller;
    const current = () => !this.stopped && this.run === run && this.cards.get(step.seq) === c && c.controller === controller;
    const limit = timeLimit(controller, this.timers);
    try {
      const work = async () => {
        // from= 을 안 쓴다 (R43). 창을 잘라 받으면 경계를 가로지르는 줄이
        // 앞 창에서는 partial, 뒤 창에서는 head 라 양쪽에서 다 빠지고,
        // 사라진 것을 화면이 알 길이 없다 — 잘림을 값으로 내는 이 팩의
        // 성질을 정확히 어기는 자리다.
        const body = await this.fetch(run, step, 'events', controller, limit);
        if (body === null) return null;
        // parseEvents 는 검증한 몸통을 그대로 낸다 (head · partial · elided 가
        // 거기 있다). 카드가 드는 것은 사건 배열 하나이므로 여기서 꺼낸다.
        const result = this.render.parseEvents(await body.json());
        return { headers: body.headers, events: result.events };
      };
      const got = await Promise.race([work(), limit.promise]);
      if (!current() || !got) return;
      this.apply(c, got.headers, got.events);
      if (c.rawOpen) {
        const raw = await this.fetch(run, step, 'raw', controller, limit);
        if (current() && raw) c.raw = await raw.text();
      }
      // 멈추는 신호는 둘뿐이다 — 상세가 끝을 말하거나 서버가 봉인된 파일을
      // 낸다 (R48). 총 길이가 안 움직이는 것을 끝으로 읽지 않는다: 오래
      // 생각하는 구간과 끝난 구간이 같은 모양이 되고, 목록 폴링을 막은
      // 상태에서 그 규칙이 폴러를 멈추면 CB4 가 재려던 동작이 사라진다.
      if (c.source === 'sealed' || step.state === 'DONE' || step.state === 'FAILED') c.done = true;
      Object.assign(c, { error: '', failures: 0, frozen: false, receivedAt: this.now() });
    } catch (error) {
      if (!current() || (controller.signal.aborted && !limit.timedOut)) return;
      // 마지막 값을 안 지운다 (R59). 빈 화면으로 떨어뜨리면 「단계가 끝났다」
      // 로 읽히는데 실제로는 중앙에 못 닿은 것이다.
      c.failures++; c.error = describeFailure(error, limit.timedOut);
      if (c.failures >= FREEZE_AFTER) c.frozen = true;
    } finally {
      limit.clear();
      if (current()) { c.pending = false; c.controller = null; this.onChange(this.cards); }
    }
  }

  async fetch(run, step, as, controller, limit) {
    const url = `/v1/runs/${encodeURIComponent(run)}/steps/${step.seq}/log?name=${encodeURIComponent(step.name)}&as=${as}`;
    const response = await this.fetcher(url, { method: 'GET', headers: this.token ? { Authorization: `Bearer ${this.token}` } : {}, credentials: 'omit', cache: 'no-store', redirect: 'error', signal: controller.signal });
    if (!limit || limit.timedOut) return null;
    if (response.status === 404) { this.card(step.seq).missing = true; return null; }
    if (response.status === 429) { this.retryAt = Math.max(this.retryAt, this.now() + retryDelay(response.headers.get('Retry-After'), this.wallNow())); throw new Error('조회 한도 초과 · 잠시 후 다시 조회'); }
    if (!response.ok) throw new Error(`조회 실패 (${response.status})`);
    return response;
  }

  // apply 는 헤더 넷과 사건 열을 카드에 앉힌다.
  //
  // 시도가 갈리면 든 것을 버린다 (R44 · R45). 진행 파일은 시도마다 다른
  // 파일이고 서버가 앞 시도를 걷으므로, 섞으면 한 카드가 두 시도의 바이트를
  // 이어 붙인 것처럼 보인다.
  apply(c, headers, events) {
    const attempt = Number(headers.get('X-Enode-Log-Attempt') ?? 0);
    const total = Number(headers.get('X-Enode-Log-Bytes') ?? 0);
    if (c.attempt !== null && attempt !== c.attempt) {
      Object.assign(c, { events: [], total: null, raw: '', open: new Set(), changedAt: null });
    }
    // 시계는 첫 바이트가 온 뒤에 흐른다. null 에서 0 으로 간 것은 움직인
    // 것이 아니다 - 그것을 갱신으로 세면 아직 시작도 안 한 단계가
    // 「마지막 갱신 67초 전」이라고 말한다 (CB4 에서 실제로 그랬다).
    if (total > 0 && c.total !== total) c.changedAt = this.wallNow();
    Object.assign(c, { attempt, total, events, source: headers.get('X-Enode-Log-Source') || '', capped: headers.get('X-Enode-Log-Capped') === '1' });
  }
}
