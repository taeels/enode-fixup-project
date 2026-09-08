# obs — 읽기 경로의 흐름

AI-DLC Construction · obs 유닛(W0 · CP1)의 Functional Design 산출물 하나다.
표가 아니라 **흐름**이다 — 요청 하나가 들어와 응답이 나갈 때까지 무엇이 어떤
순서로 도는가. 쓰는 개념의 정의는 `domain-entities.md`, 규칙과 검증은
`business-rules.md`.

---

## 1. 전체 (한눈)

```text
   브라우저 · runctl · MCP · 제어판
        |
        |  Authorization: Bearer <token>      (데모 모드의 읽기 셋은 무인증 · §7)
        v
   +--------------------------------------------------+
   |  internal/api                                    |
   |                                                  |
   |   GET  /v1/nodes      nodes.go   (신규)           |
   |   GET  /v1/runs       runs.go    (신규)           |
   |   GET  /v1/runs/{id}  api.go     (넓힘)           |
   |   POST /v1/nodes      api.go     (정책 복사·되돌림) |
   +--------------------------------------------------+
        |            |             |              |
        | Nodes()    | Runs(f)     | GetRun       | UpsertAdvert
        |            |             | LiveContract |
        |            |             | Steps        |
        v            v             v              v
   +--------------------------------------------------+
   |  internal/store   읽기 셋 + 쓰기 하나              |
   +--------------------------------------------------+
        |
        v
      PostgreSQL   nodes · leases · runs · steps
```

**어느 흐름에도 매처가 없다.** `internal/match` 를 부르지 않는다 — 관측 경로에서
매처를 돌리면 `ADR-065` 가 가른 둘(관측과 배정)이 다시 붙고, 관측 시점과 배정
시점이 달라 그 판정은 내는 순간 낡는다.

---

## 2. GET /v1/nodes

```text
   1  auth         Bearer 토큰.  기존 s.auth (데모 모드는 §7)
   2  질의 인자     넷 다 안 받는다.  하나라도 오면 400 (business-rules 2.4)
   3  Nodes(ctx)   질의 하나 —
                     WITH at AS (SELECT now() AS observed_at)
                     SELECT at.observed_at, n.node_id, ..., l.run_id, l.not_after
                       FROM at
                       LEFT JOIN nodes  n ON n.expires_at > at.observed_at
                       LEFT JOIN leases l ON l.node_id = n.node_id
                      ORDER BY n.node_id
   4  observed_at  모든 행에 같은 값이 실린다.  노드가 0 이어도 행이 하나 나오고
                   (n.node_id 가 NULL 인 그 행) 시각은 거기 있다.  그 행은 건너뛴다
   5  write 200    {observed_at, nodes:[NodeView]}
```

**질의를 하나로 둔다.** 노드 목록과 임대 목록을 따로 읽어 애플리케이션에서
붙이면 두 읽기 사이에 임대가 생기거나 풀려서 「관측된 한 시점」이 아니게 된다.
`observed_at` 이 붙은 스냅샷이라는 말이 그 순간 거짓이 된다.

**시계는 DB 것이다.** `observed_at` 을 프로세스 시각으로 내면 `lease.not_after`
(DB 가 계산한 값)와 다른 시계가 되고, 화면의 카운트다운이 그 차이만큼 어긋난다.

**그런데 시각을 행에 얹으면 빈 함대에서 값이 없다.** 노드가 하나도 없으면 행이
0 이고, 이 유닛은 그때도 `200` 과 `observed_at` 을 내기로 했다.

**그래서 시각을 왼쪽에 둔다.** 1행짜리 `at` CTE 를 `FROM` 에 놓고 노드를 거기에
`LEFT JOIN` 하면 **행 수와 무관하게 시각이 서면서 질의가 여전히 하나**다. 노드가
0 이면 `n.node_id` 가 NULL 인 행 하나가 나오고, 스캔이 그 행을 건너뛴다.
`Store.Nodes` 가 `([]NodeView, time.Time, error)` 를 내는 이유다.

`SELECT now()` 를 자기 문장으로 한 번 더 부르는 길은 **버린다.** 읽기 셋에
트랜잭션이 없으므로(§8) 두 문장은 서로 다른 순간이고, 그러면 `observed_at` 이 행
스냅숏 밖의 시각이 된다 — 「관측된 한 시점」이 거짓이 되고 `domain-entities.md`
§2.1 의 `not_after <= observed_at` 판정이 그 틈만큼 뒤집힌다. 그 규칙은 부등호다.

**만료 필터도 그 시각으로 건다.** `n.expires_at > at.observed_at` 이다 —
`now()` 를 다시 부르면 필터의 기준과 응답이 말하는 시각이 갈린다. `LiveAdverts` 가
쓰는 조건(`expires_at > now()`)과 뜻이 같다: 살아 있음의 신탁이 아니라 「죽었지만
아직 만료 안 된 노드가 잠깐 남는다」는 사실을 그대로 잇는다(`ADR-065`).

**임대는 `not_after` 로 안 거른다.** 행이 있다는 것이 곧 점유이고, `BusyNodes` 를
포함해 저장소 전체가 그 규칙이다. 지나간 `not_after` 가 남아 있는 것은 사고가
아니라 `ADR-047` 이다 — 해석은 `domain-entities.md` §2.1.

**draining 인 노드도 만료 전이면 나온다.** 「있는데 안 빌려준다」가 「죽었다」와
갈리는 자리가 이 응답이다.

---

## 3. GET /v1/runs

```text
   1  auth
   2  질의 문자열을 RunFilter 로 읽는다   state · since · work · limit
      값이 나쁘면 여기서 400 · 모르는 인자는 무시 (business-rules 2절)
   3  Runs(ctx, filter)
        WITH at AS (SELECT now() AS observed_at)
        SELECT at.observed_at, r.*
          FROM at
          LEFT JOIN LATERAL (
                SELECT ... coalesce(work_id,''), coalesce(assigned,'[]'::jsonb),
                       submitter
                  FROM runs
                 WHERE (state 지정되면 state = $)        좁히기만 한다
                   AND (since 지정되면 created_at >= $)  포함 하한
                   AND (work  지정되면 work_id  = $)
                 ORDER BY created_at DESC
                 LIMIT $
          ) r ON true
   4  observed_at  §2 와 같은 규칙 — 행이 0 이어도 값이 선다
   5  write 200    {observed_at, runs:[RunRow]}
```

`LIMIT` 이 안쪽에 있어야 한다 — 밖에 걸면 시각 행까지 세어 한 줄이 모자란다.
`LATERAL` 인 이유는 그 안쪽이 바깥 행(`at`)을 안 보기 때문이 아니라 **0행일 때도
왼쪽 한 줄이 남게 하려는 것**이고, 그것이 §2 와 같은 모양이다.

**최신이 앞이다.** 이 자리는 「지금 무엇이 도는가」를 답한다. 지난 것을 뒤지는
자리는 Record 다.

**페이지네이션이 없다.** `limit` 뿐이다 — 커서를 주면 그 커서가 정렬 키와
필터에 묶이고, 그 순간 이 표면이 질의 표면이 된다(`ADR-065` §4 와 같은 조건).

**빈 결과는 `200` 과 빈 배열이다.** `null` 도 `404` 도 아니다 — 화면이 「없음」과
「못 읽음」을 갈라야 한다.

**NULL 을 SQL 에서 접는다.** `work_id` 와 `assigned` 는 `ALTER` 로 붙거나 nullable
이라 스캔이 죽거나 `null` 이 샌다. `coalesce` 로 접는 자리가 여기다 — `assigned` 는
빈 배열이어야 CP2 의 화면 판정이 선다(`domain-entities.md` §3).

**`coalesce` 가 못 접는 것이 하나 있다 — jsonb 의 `null`.** SQL NULL 이 아니라
값이라 그대로 통과한다. 오늘 그것을 넣는 자리는 없지만 `CreateQueuedRun`(W1)이
제로값을 쓰면 생긴다. 읽기 쪽에서 막을 수 없으므로 **쓰는 쪽의 계약**으로 적었다
(`domain-entities.md` §3.1). 정렬 열(`created_at`)에 단독 인덱스가 없다는 것도
같은 자리에 적어 둔다 — 오늘 `runs_work_idx (work_id, created_at)` 하나뿐이고,
`?work=` 없는 목록은 그것을 못 탄다. `limit` 상한이 1000 이므로 함대가 커지면
정렬 비용이 보인다. **이 유닛은 인덱스를 안 붙인다** — `mediator-api.md` 가 이
표면을 「검색 계층이 아니다. 인덱스도 질의 언어도 없다」로 그었다. 붙일 필요가
보이면 그것은 표면의 성격이 바뀐 것이고 그때 정본을 먼저 연다.

---

## 4. GET /v1/runs/{id} — 넓히는 것 둘

```text
   1  GetRun(id)            오늘 그대로.  없으면 404
   2  Steps(id)             SELECT 에 chosen 을 더한다 (열은 이미 있다)
   3  LiveContract(id)      coalesce(contract_versions -> -1, contract)
        -> requires[] 를 RequireView 로 옮긴다 (attrs 로 감싼다)
   4  write 200             runView + requires
```

**`requires` 는 `getRun` 에서만 채운다.** `runView` 를 짓는 `view()` 를 부르는
자리가 넷이고(재제출 200 둘 · 신규 201 · getRun 200), dry-run 은 `runView` 리터럴을
따로 짓는다. `view()` 안에서 채우면 **제출 응답 셋이 함께 넓어진다** — 계약을 푸는
비용이 제출 경로에 붙고, `ADR-069` §3 이 「목록에는 안 싣는다」로 좁혀 둔 것이 다른
문으로 새어 나간다. 그래서 `view()` 는 그대로 두고 `getRun` 이 채운다.

키는 `json:"requires,omitempty"` 다. 「키는 언제나 있다」는 새 라우트 둘의 규칙이고
(`business-rules.md` §3), 여기는 **기존 표면의 글자를 안 바꾸는 것**이 먼저다 —
`omitempty` 가 아니면 제출 응답 셋에 `"requires": null` 이 생긴다. `getRun` 에서는
계약이 언제나 `requires` 를 갖고(비면 `Validate` 가 접수에서 막는다) 그래서 키가
언제나 선다. **상태로 안 가른다**는 `ADR-069` §4 는 지켜진다 — 가른 것은 라우트다.

**`chosen` 은 반대다 — 제출 응답도 함께 바뀐다.** `StepView` 에 언제나 싣기로
했으므로 `201` 본문의 단계마다 `"chosen": false` 가 는다. 그것이 옳다:
`202`(QUEUED)가 `201` 과 같은 모양이어야 하고, 둘이 같은 `view()` 를 타므로
함께 움직이는 쪽이 그 규칙을 지킨다.

**계약을 통째로 안 싣는다.** `requires` 만 옮긴다 — 계약 전문은 낸 쪽이 갖고
봉인본에 남는다(`ADR-025` §6). `requires` 가 예외인 이유는 그것이 **배정의
입력**이라 성격이 다르기 때문이다(`ADR-069` §4).

**계약을 못 읽어도 Run 상태는 준다.** 오늘 `Steps` 실패가 조회를 안 막는 것과
같은 규칙이다. `requires` 가 빠진 채로 나간다.

**빠졌다는 것을 응답에 적는다 — 로그에만 남기지 않는다.** `omitempty` 와 이 갈래를
함께 두면 `200` 에서 「기다리는 것이 없다」와 「계약을 못 읽었다」가 한 글자가 되고,
그것이 CP2 가 재는 바로 그 값이다(「QUEUED 행을 누르면 무슨 능력을 기다리는지」).
화면은 조용히 빈 칸을 그리고 원인은 서버 로그에만 있다. `runView` 에 `warnings` 가
이미 있으므로 **거기에 한 줄을 싣는다** — 새 필드도 새 라우트도 아니다. 로그는
그대로 남긴다.

빈 `requires` 자체는 안 나온다 — 계약이 요구를 안 가지면 `Validate` 가 접수에서
막는다. 그래서 `getRun` 에서 이 키가 없다는 것은 **언제나 읽기 실패**이고,
`warnings` 는 그것을 읽는 쪽이 볼 수 있게 만드는 한 줄이다.

**판정을 안 낸다.** 「ws-b 는 harness 가 안 맞는다」를 서버가 내지 않는다.
요구를 그대로 내고 화면이 `GET /v1/nodes` 와 대조한다(`ADR-069` §5).

---

## 5. POST /v1/nodes — 정책의 왕복 (이 유닛의 유일한 쓰기)

답 1=C 가 이 흐름을 obs 로 들였다. **왕복이 셋이다** — 받고, 적고, 되돌린다.

```text
   1  광고 본문을 읽는다        contract.Advert 에 Policy{Drain} 이 늘었다.  핸들러는
                               값을 안 본다 — 그대로 UpsertAdvert 에 넘긴다
   2  UpsertAdvert             어휘를 검사해 벗어나면 "" 로 접고 로그에 남긴 뒤,
                               CTE 로 이전 draining 을 받으면서 새 값을 쓴다
                               (domain-entities §7.3 · §8)
   3  응답에 drain 을 싣는다     advertResponse 에 저장된 값.  통보이지 판정이 아니다.
                               언제나 싣는다 — omitempty 가 아니다
   4  그 뒤는 오늘 그대로        재시작 판정 -> 임대 갱신 -> 응답
```

**검사는 핸들러가 아니라 store 안이다.** 이 열에 쓰는 자리가 하나뿐이라 그 자리에
붙는 것이 맞고, `internal/api` 의 커버리지 여유가 5.6 문장뿐이라 검사와 로그를
그 예산에 얹을 자리가 없다(`business-rules.md` §7.6). 근거는 `domain-entities.md`
§7.3 에 있다.

**이름이 자리마다 다르다** — 본문은 `policy.drain`, 응답은 `drain`, `GET /v1/nodes`
는 `draining`. `ADR-063` §6 과 `mediator-api.md` 가 셋 다 그렇게 적어 두었다.

**어휘 검사에서 `400` 을 내지 않는다.** 광고는 하트비트를 겸하므로(`ADR-016`)
정책 값 하나로 광고 전체를 거절하면 노드가 함대에서 사라진다 — 증상이 원인보다
훨씬 크다. 접고 로그에 남긴다.

**이전 값을 돌려주는 이유는 obs 가 안 쓴다.** `decisions.md` 1절이 정한
`WakeQueued` 호출 지점 여섯 중 하나가 「drain 해제를 받은 광고 처리」인데,
덮어쓰기만 하면 해제를 본 사람이 아무도 없다. **W0 이 버릴 값을 세워 두는 것**이고,
그래야 drain(W2)이 `store.go` 접점을 다시 안 연다.

**정책의 정본은 노드의 파일이다.** 이 열은 복사본이고, 중앙은 그것을 되돌려
보여줄 뿐이다. 중앙 라우트로 drain 을 걸 수 있게 하지 않는다 — 토큰만 있으면
남의 노드를 뺄 수 있게 된다(`decisions.md` 1절).

**drain 유닛(W2)이 딛는 자리가 여기까지 미리 선다.** 남는 것은 `internal/enode` 의
정책 파일 읽기와 광고 적재 · 응답을 받아 Worker 에 넘기기 · 두 모드 · at-boundary
취소, 그리고 해제 시 `WakeQueued` 호출이다.

---

## 6. runctl.Client

```text
   Nodes(ctx)      GET /v1/nodes           -> 응답 본문 원문 그대로
   Runs(ctx, q)    GET /v1/runs?<질의>      -> 응답 본문 원문 그대로
```

기존 `do` 를 그대로 탄다 — `Authorization` 과 `X-Enode-Principal` 이 붙고,
2xx 가 아니면 `Fail{Code, Reason}` 이 된다. **파싱하지 않는다**: mcp 의
`fleet.list` 가 `GET /v1/nodes` 와 글자까지 같아야 하고(CP5), 파싱해서 다시
마셜하면 키 순서와 생략 규칙이 그 사이에서 갈린다.

**`RunsQuery` 는 빈 값과 0 을 질의에서 뺀다.** 제로값을 그대로 펴면 `?limit=0` 이
되고 서버가 `400` 으로 거절한다 — 인자 없이 부르는 것이 `runs.list` 의 기본
사용법이므로 그것을 안 지키면 CP5 의 첫 호출이 실패한다.

부르는 쪽 셋이 이미 정해져 있다.

```text
   mcp (W1)         fleet.list · runs.list 의 passthrough
   panel (W3)       자기 node_id 행의 lease 를 뽑는다
   transcript (W4)  GET /v1/runs 를 자기 node_id 로 거른다 (assigned 로 거른다)
```

뒤 둘은 원문을 자기 자리에서 자기가 아는 만큼만 푼다. obs 가 파싱 타입을 미리
내지 않는 이유다 — 세 유닛이 원하는 모양이 서로 다르다.

**예외는 `Client.Status` 가 쓰는 파싱 타입이다.** 그쪽은 원문이 아니라 구조체이고
panel 이 S3 를 그리려면 `Step.StartedAt` 이 있어야 하는데 오늘 없다. 담당이 갈리는
파일이므로 W0 에서 세운다 — `domain-entities.md` §5 · §8.

**panel 에 넘기는 주의 한 줄** — `Client` 를 세울 때 `runctl.Principal()` 을
부르면 안 된다. 그 함수는 `git config --get user.email` 이 없으면 그 자리에서
죽고, 제어판이 도는 기계가 정확히 그런 기계일 수 있다(git 없는 노트북).
`Client.Principal` 은 평범한 필드이므로 `enode.Derive(config).Principal` 을 넣는다.

---

## 7. 데모 모드의 읽기 (답 4=A)

데모 모드에는 S0(토큰 입력)이 없다 — 「토큰 입력 대신 Guest 로그인」이다. 그런데
데모 대시보드는 3D 함대뷰 · run 목록 · 작업 그래프를 그리려면 이 유닛의 라우트를
불러야 한다. **읽기 셋을 데모 인스턴스에서만 무인증으로 등록한다.**

```text
   데모 아님 (기본)        전부 s.auth.  오늘 그대로
   데모 인스턴스           GET /v1/nodes · GET /v1/runs · GET /v1/runs/{id} 만 무인증
                          쓰기(POST /v1/runs · cancel · answer ...)는 그대로 잠긴다
                          GET /v1/asks 도 잠긴다 (domain-entities §2.1 의 확정이 반쪽)
```

**가둠 셋이 그대로 산다**(`decisions.md` §8.4) — 일회용 데모 Mediator · 서버측
allow-list(demo-back) · **브라우저에 실 토큰 없음**. 토큰을 페이지에 박는 길은
그 셋째를 깨고, 그 토큰으로 임의 계약을 낼 수 있어 allow-list 가 장식이 된다.
demo-back 에 읽기 프록시를 세우는 길은 관측 표면을 두 벌로 만든다 —
`ADR-025` §3 · `ADR-069` §3 의 「표면 개수가 비용이다」와 정면이다.

### 7.1 이 답은 정본을 벗어난다 — 그 사실을 적는다

`ADR-065` 2절이 **「인증은 다른 Mediator 표면과 같은 Bearer 토큰을 쓴다. 별도 운영
권한은 MVP 신뢰 경계가 갈릴 때 붙일 자리로 남긴다」**로 못 박았고,
`enode-features.md` 3.1.1 의 보안 요구도 「화면이 S0 에서 토큰을 받아
`Authorization` 헤더로 두 라우트를 부른다」다. **답 4=A 는 그 줄을 벗어난다.**

벗어나는 근거는 같은 팩 안에 있다 — `enode-features.md` 의 데모 전환이 「토큰 입력
대신 Guest 로그인」이라 S0 자체가 없다. **정본 둘이 어긋나는 자리이고
`canon.md` 머리말의 서열대로면 `enode-design` 이 이긴다.** 사용자가 A 를 유지하기로
정했으므로 이 유닛은 그 값으로 가되, **어긋남을 통과로 적지 않는다.**

```text
   벗어난 정본        ADR-065 2절 인증 줄 · enode-features 3.1.1 보안 요구
   벗어난 범위        데모 인스턴스의 읽기 셋 셋뿐.  실 함대는 무변경
   되돌리는 값        Handler() 의 조건부 등록과 config 스위치 하나
   진행자가 정할 것    ADR-065 를 개정할지, 데모를 정본 밖 예외로 둘지
```

### 7.2 무인증으로 나가는 값을 센다

「`principal` 을 안 낸다」로 닫으면 **안 나가는 것 하나만 센 것**이다. 나가는 것을
적는다.

```text
   GET /v1/nodes       label · instance · capabilities · seen_at · expires_at ·
                       lease{run_id, not_after} · draining
   GET /v1/runs        run_id · state · verdict · work_id · created_at ·
                       ended_at · assigned[] · submitter
   GET /v1/runs/{id}   위 + steps[] · reject · verdict · warnings · requires
```

**`label` 이 신원을 진다.** `ADR-065` 가 `principal` 을 안 내기로 하면서 「사람이
읽는 식별은 이미 `label` 에 있다」고 적었고, `mediator-api.md` 의 예시가 그 실체다 —
`<이메일 로컬파트>@<호스트명>:<워크스페이스>`. 즉 **게스트는 랜덤 가명인데 노드
소유자는 실명이 공개 페이지에 나간다.** `constraints.md` 가 데모의 게스트 이름을
「랜덤 표시 라벨이지 신원이 아니다」로 못 박은 것과 방향이 반대다.

**이 유닛은 응답을 안 바꾼다** — `label` 을 가리면 `ADR-065` 의 응답 모양을 모드에
따라 가르는 것이 되고 mcp 의 글자 일치(CP5)와 부딪친다. **대신 데모에 세울 함대의
`label` 을 데모용으로 짓는 것**이 값싼 길이고, 그것은 노드 설정의 일이라 이 유닛
밖이다. **진행자에게 남기는 표시다** — 데모 함대를 누구 기계로 세우느냐가
이 노출의 크기를 정한다.

### 7.3 스위치

갈림의 자리는 **`Handler()` 의 등록 줄**이고 값은 설정에서 온다.

```text
   이름     ENODE_DEMO_MODE
   형       불리언 (1 · true · yes 를 참으로 읽는다.  internal/config 의 기존 모양)
   기본값   거짓.  실 함대는 오늘 그대로다
   읽는 쪽  obs (읽기 셋 등록) · demo-back (데모 쓰기 라우트 등록 · W2)
```

`internal/config` 가 이미 `DATABASE_URL` · `ENODE_MEDIATOR_TOKEN` 을 환경변수로
받으므로 같은 모양이다. `decisions.md` 3절의 「`internal/config` 를 **보안 이유로**
고치지 말 것」은 TLS 를 실으려는 것을 막은 문장이고, 이것은 `decisions.md` 8절이
범위로 들인 데모 모드의 스위치다 — 다른 이유다.

**`internal/config/config.go` 는 행렬 밖이지만 단독이 아니다.** demo-back(runixs)이
같은 스위치를 읽어야 하므로 **접점으로 봐야 한다** — 진행자가 직렬 대상에 넣는다.
`internal/contract/advert.go`(단독 obs)와 다른 처리다.

### 7.4 「쓰기는 잠긴다」는 이 유닛의 규칙이지 인스턴스의 규칙이 아니다

demo-back(W2)은 데모 제출 라우트를 **무인증으로** 내야 한다 — 브라우저에 토큰이
없는데 CP9 가 그 화면에서 제출을 요구한다. 가둠 셋의 둘째(서버측 allow-list)가
그 라우트를 지키는 값이고, 그것이 `decisions.md` §8.4 의 설계다.

**그러므로 §7 의 「쓰기는 그대로 잠긴다」는 obs 가 여는 라우트에 대한 문장이다.**
demo-back 이 자기 라우트를 무인증으로 여는 것은 이 규칙을 깨는 것이 아니다 —
그쪽은 allow-list 가 지킨다. 이 문장을 안 적으면 W2 담당이 obs 의 선언을 인스턴스
전역으로 읽고 자기 설계를 되연다.

---

## 8. 이 유닛이 안 태우는 것

```text
   internal/match     관측 경로에서 매처를 안 돈다 (ADR-065)
   트랜잭션            읽기 셋은 필요 없다.  UpsertAdvert 는 CTE 한 문장이다.
                      postNodes 에 tx 를 두르는 것은 queue(W1)의 일이다 —
                      WakeQueued 여섯 지점 중 둘이 그 핸들러 안이다
   Record 저장소       봉인은 이 유닛의 일이 아니다.  GET record 는 오늘 그대로
   describeWant       봉인용 표기다.  QUEUED 는 정의상 종료 전이라 안 돈다
```
