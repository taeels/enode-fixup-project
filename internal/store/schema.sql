-- Mediator 의 상태 저장소 (ADR-015 §3).
--
-- 여기 있는 것   경쟁이 있고 계속 바뀌는 것 — 광고 · 점유 · Run 상태 · 단계 진행
-- 여기 없는 것   경쟁이 없고 봉인되는 것 — Run Record 디렉터리 · blob 본문
--                I4(봉인)를 파일시스템은 강제할 수 있고 행은 못 하기 때문이다.
--
-- 매칭은 SQL 이 하지 않는다 — ADR-014 결정 3 이 매처를 순수 함수로 못 박았고
-- runs 와 dry-run 이 같은 함수를 부른다. DB 는 광고와 점유를 돌려줄 뿐이다.

CREATE TABLE IF NOT EXISTS nodes (
    node_id      text PRIMARY KEY,
    label        text        NOT NULL,
    principal    text        NOT NULL,  -- 식별이지 인증이 아니다 (ADR-015 §1)
    -- [{capability, attrs}] — 속성 어휘가 창발하므로(ADR-012) 정규화하지 않는다.
    -- 델타가 아니라 매번 전부다 그래서 갱신은 통째 교체이고,
    -- capability 를 빼고 보내는 것이 곧 "지금은 못 한다" 가 된다 (ADR-017 결정 3).
    capabilities jsonb       NOT NULL,
    expires_at   timestamptz NOT NULL,  -- 광고는 만료된다 (ADR-012)
    seen_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS runs (
    run_id     text PRIMARY KEY,        -- runctl 이 만든다. 재제출이 멱등이다 (INVARIANTS §4)
    state      text        NOT NULL,
    principal  text        NOT NULL,
    contract   jsonb       NOT NULL,    -- 계약 전문. Record 의 manifest 가 된다.
                                        -- ADR-020 이 스키마를 인라인으로 둔 덕에 여기 다 들어온다.
    assigned   jsonb,                   -- [{as, nodes:[{node,label}]}] — ALLOCATING 을 지난 뒤
    reject     jsonb,                   -- 거절 사유 (422/409). FAILED 의 원인이 남는다.
    verdict    jsonb,                   -- ⑩ 의 대조 결과. Record 의 verdict.json 이 된다.
    created_at timestamptz NOT NULL DEFAULT now(),
    ended_at   timestamptz
);
ALTER TABLE runs ADD COLUMN IF NOT EXISTS verdict jsonb;

-- Work 의 키 (ADR-023 §6.5.2). 계약의 work 에서 유도해 여기 박는다.
-- Run 을 넘어 사는 유일한 식별자다 — 원장이 scope:"work" 로 넓어질 때
-- 「같은 Work 의 이전 Run 들」을 찾는 것이 이 열이고, ADR-005 manifest 의
-- 「Work id」가 오늘 비어 있던 자리이기도 하다.
ALTER TABLE runs ADD COLUMN IF NOT EXISTS work_id text;
CREATE INDEX IF NOT EXISTS runs_work_idx ON runs (work_id, created_at);

-- 계약의 열 중 v2 이후 (ADR-022 §7.6 · P4).
-- v1(제출 전문)은 runs.contract 가 그대로 든다 — 성질 4 는 제출본을 요구한다.
-- 실행 중에 붙는 판만 여기 쌓이고, 봉인 시점에 v1 뒤에 이어 붙는다.
-- 비어 있으면 오늘 그대로 — append 하는 주체가 없으면 판이 하나다.
ALTER TABLE runs ADD COLUMN IF NOT EXISTS contract_versions jsonb NOT NULL DEFAULT '[]'::jsonb;

-- 점유 장부 — I1 이 여기서 스키마로 강제된다
--
-- node_id 가 PRIMARY KEY 인 것이 ADR-019 결정 2 의 직접 표현이다:
-- capability 어휘가 하나로 줄면서 임대 키 (노드, capability) 가 (노드) 로 붕괴했고,
-- 그래서 노드 자체가 배타 자원이 됐다.
--
-- 두 번째 Run 이 같은 노드를 잡으려 하면 애플리케이션 로직이 아니라
-- 기본키 충돌이 막는다. I5(전부 아니면 전무)는 그 충돌에 롤백을 붙여 얻는다.
CREATE TABLE IF NOT EXISTS leases (
    node_id    text PRIMARY KEY,
    run_id     text        NOT NULL REFERENCES runs(run_id) ON DELETE CASCADE,
    not_after  timestamptz NOT NULL,    -- ADR-010 허가 아티팩트의 만료
    nonce      text        NOT NULL,
    granted_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS leases_run_idx ON leases (run_id);

-- 노드 프로세스의 「이번 생」 표식 (ADR-030). 광고가 나른다.
-- 재시작하면 달라진다 — claim 재전달(같은 생)과 재시작 판정(다른 생)을 가른다.
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS instance text;

-- 단계. Mediator 가 시퀀서이므로(ADR-014 결정 1) 순서는 여기 있고 계약에서 온다.
CREATE TABLE IF NOT EXISTS steps (
    run_id   text        NOT NULL REFERENCES runs(run_id) ON DELETE CASCADE,
    seq      int         NOT NULL,      -- 계약의 steps[] 순서. step_id 는 run_id#NN 으로 노출한다.
    name     text        NOT NULL,      -- 계약의 steps[].id
    uses     text        NOT NULL,      -- 역할 이름
    kind     text        NOT NULL,      -- agent | run  (ADR-019 결정 3)
    state    text        NOT NULL,      -- PENDING | CLAIMED | DONE | FAILED | SKIPPED
                                    -- SKIPPED 는 dispatch 가 안 간 경로다 (ADR-022 §7.2).
                                    -- 종료 상태이면서 실패가 아니다. CHECK 를 안 거는 이유는
                                    -- 어휘가 늘 때 마이그레이션을 강요하지 않기 위해서다.
    node_id  text,                      -- 배정된 노드. claim 이 채운다 (S4).
    attempt  int         NOT NULL DEFAULT 0,
    started_at timestamptz,
    ended_at   timestamptz,
    result   jsonb,                     -- exit_code · produced · harness (ADR-020)
    -- 이 단계가 기다리는 단계 이름들 (ADR-023 §4). 게이트의 술어가 여기 선다.
    -- 계약의 steps[].needs 를 정규화한 값이다 — 안 적은 계약은 [직전 단계] 로
    -- 채워져 들어오므로 게이트는 한 형태만 안다. 빈 배열은 "안 기다린다" 다.
    needs    text[]      NOT NULL DEFAULT '{}',
    PRIMARY KEY (run_id, seq)
);

-- 이 열이 생기기 전에 만들어진 행의 백필
-- NULL 로 붙였다가 [직전 단계] 로 채우고 NOT NULL 로 조인다. NULL 이 곧
-- "열이 생기기 전" 의 표지다 — 빈 배열(안 기다린다)과 섞이지 않는다.
ALTER TABLE steps ADD COLUMN IF NOT EXISTS needs text[];
UPDATE steps s SET needs = coalesce(
         (SELECT ARRAY[p.name] FROM steps p
           WHERE p.run_id = s.run_id AND p.seq = s.seq - 1), '{}')
 WHERE s.needs IS NULL;
ALTER TABLE steps ALTER COLUMN needs SET DEFAULT '{}';
ALTER TABLE steps ALTER COLUMN needs SET NOT NULL;

-- 워터마크 (ADR-023 §6.4 자리 2) — 그 단계를 집을 때 원장에 있던 것들.
-- 이것이 성질 4(자기충족)를 지키는 장치다: 봉인된 묶음만 열어서
-- 「무엇을 볼 수 있었나」를 알 수 있어야 하고, 안 깔린 것도 여기 남는다.
-- 항목은 blobs/ 의 파일 이름과 같은 형태라 (회차, 순번, 이름) 으로 유일하다.
ALTER TABLE steps ADD COLUMN IF NOT EXISTS ledger_at text[];

-- 이 단계를 어느 「생」이 집었는가 (ADR-030).
-- 같은 생이 다시 물으면 재전달하고, 다른 생이 나타나면 실패시킨다.
ALTER TABLE steps ADD COLUMN IF NOT EXISTS claimed_instance text;

-- 되묻기의 기한 (ADR-032). ASKED 로 만들 때 timeout.after 에서 계산해 박는다.
-- NULL 이면 무한 대기 — 사람의 시간을 시스템이 짐작하지 않는다.
ALTER TABLE steps ADD COLUMN IF NOT EXISTS ask_deadline timestamptz;

-- 되먹임 봉투의 열쇠 (ADR-050) — 그 단계의 프롬프트에서 앞 단계의 도구
-- 출력을 감싸는 구분자에 붙는다.
--
-- 왜 단계마다 다른가 — 앞 단계도 에이전트다. 구분자가 고정이면 앞 단계가
-- 산출물 안에 종료 표식을 적어 봉투를 빠져나올 수 있다. 열쇠는 집을 때
-- 뽑으므로, 앞 단계가 산출물을 쓰던 시점에 다음 단계의 열쇠는 존재하지 않는다.
--
-- 왜 행에 남기는가 — 재전달(ADR-030)이 같은 프롬프트를 만들어야 하고,
-- 봉인된 Record 를 열었을 때 그 프롬프트가 왜 그 모양이었는지 설명돼야 한다.
-- NULL 이면 열쇠 없이 봉투만 씌운다 — 열쇠는 강화이지 봉투의 조건이 아니다.
ALTER TABLE steps ADD COLUMN IF NOT EXISTS envelope_key text;

-- 이 단계가 갈림길에 골라진 적이 있는가 (ADR-060 §3).
--
-- SKIPPED 하나로는 두 가지 서로 다른 것을 구분할 수 없다:
--
--     안 고른 SKIPPED         경로가 갈렸다 — 조건은 공허하게 참이 옳다
--     고른 뒤에도 SKIPPED     골랐는데 못 닿았다 — 목표 미달이다
--
-- 계약 저자는 경로별로 조건을 나눠 쓸 방법이 없으므로(경로는 실행 시 정해진다)
-- 앞의 것은 참으로 둘 수밖에 없다. 그러나 뒤의 것까지 참으로 두면
-- 목표 판정 단계가 통째로 빠진 Run 이 SUCCEEDED 로 봉인된다 (third-run-1).
--
-- 되돌리지 않는다 — 한 번 골라진 사실은 뒤 회차가 지우지 않는다.
-- loop 이 구간을 되돌릴 때도 남는다: "골랐었다" 는 그 회차의 기록이 아니라
-- 이 Run 에서 그 자리가 목표였다는 사실이다.
ALTER TABLE steps ADD COLUMN IF NOT EXISTS chosen boolean NOT NULL DEFAULT false;

-- 노드 소유자가 건 drain 정책의 복사본 (ADR-063 §6).
--
-- 정본은 노드의 정책 파일이고 이 열은 최근 광고에 실려 온 것이다.
-- 어휘는 셋 — "" (안 걸림) · graceful (새 임대만 막음) · at-boundary (경계에서 닫음).
--
-- CHECK 를 안 건다. steps.state 와 같은 이유다 — 어휘가 늘 때 마이그레이션을
-- 강요하지 않는다. 대신 애플리케이션이 검사한다 (store.DrainPolicy).
-- 그 둘은 다른 것이다: 앞은 스키마의 경직을 피하는 결정이고,
-- 뒤는 어휘 밖 문자열이 들어와 그 노드가 매칭 후보에서 조용히 빠지는 것을 막는다.
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS draining text NOT NULL DEFAULT '';

-- 제출자 표시 라벨. 게스트 로그인 이름이 그대로 실린다.
--
-- 권한이 아니라 자기 신고다 (ADR-015 §1) — principal 과 같은 성격이고,
-- 그래서 거르는 값으로 쓰지 않는다. 실 함대의 Run 은 빈 문자열이다.
ALTER TABLE runs ADD COLUMN IF NOT EXISTS submitter text NOT NULL DEFAULT '';

-- 정렬 열 단독 인덱스.
--
-- 오늘 있는 것은 runs_work_idx (work_id, created_at) 하나뿐이라
-- ?work= 없는 기본 목록과 ?since= 가 그것을 못 탄다. 둘 다 created_at 으로
-- 좁히거나 정렬하는데 선행 열이 work_id 이기 때문이다.
-- limit 상한이 1000 이므로 함대가 자라면 그 정렬 비용이 보인다.
CREATE INDEX IF NOT EXISTS runs_created_idx ON runs (created_at);

-- 대기열의 훑기가 타는 인덱스 (ADR-064).
--
-- 임대가 지워지는 지점마다 WHERE state = 'QUEUED' ORDER BY created_at 을 한 번씩
-- 돈다 — 대기가 0 인 함대에서도 매 종료마다다. state 인덱스가 없으면 그 한 줄이
-- runs 를 통째로 훑고, runs 는 지우지 않고 쌓인다. 부분 인덱스라 대기 행만 들어
-- 대기가 없으면 비어 있고, 승격 UPDATE 가 행을 인덱스에서 뺀다.
CREATE INDEX IF NOT EXISTS runs_queued_idx ON runs (created_at) WHERE state = 'QUEUED';
