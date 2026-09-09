# obs — 관측 표면이 쓰는 것

AI-DLC Construction · obs 유닛(W0 · CP1)의 Functional Design 산출물 하나다.
**기술 비의존**이 원칙이나 이 유닛은 기존 표에서 이미 사는 값을 내보내는 일이라,
개념마다 **어느 열에서 나오는가**와 **그 열이 NULL 일 수 있는가**를 함께 적는다.
그것이 이 유닛의 실체다.

정본 — `ADR-065`(nodes 응답) · `ADR-069`(requires) · `ADR-060`(chosen) ·
`ADR-063`(소유자 정책) · `decisions.md` 2절(runs 응답·필터) · §8.6(submitter) ·
`enode-design/protocol/mediator-api.md`. **어긋나면 `enode-design` 이 이긴다**
(`requirements/canon.md` 머리말). 계획과 답 다섯은
`aidlc-docs/taeels/construction/plans/obs-functional-design-plan.md`.

---

## 1. 새 저장소는 없다

이 유닛이 만드는 개념은 **없다.** 전부 이미 저장돼 있는 것을 밖에서 읽는
모양으로 옮긴 것이다. 새로 생기는 것은 열 둘과 광고 필드 하나뿐이고, 셋 다
**이미 있는 사실의 자리**다 — 소유자 정책의 복사본과 제출자 표시 라벨.

```text
   개념             어디서 오나                              누가 쓰나
   ──────────────   ──────────────────────────────────────   ─────────────────
   NodeView         nodes 표 + leases 표 (LEFT JOIN)          GET /v1/nodes
   LeaseView        leases 표 (run_id · not_after)            NodeView 안
   RunRow           runs 표                                   GET /v1/runs
   RunFilter        질의 문자열                                GET /v1/runs (서버)
   RunsQuery        같은 넷                                    runctl.Client (클라이언트)
   StepView.chosen  steps.chosen (열은 이미 있다)              GET /v1/runs/{id}
   RequireView      지금 유효한 계약의 requires[]              GET /v1/runs/{id} 만
```

---

## 2. NodeView — 관측된 노드 하나

`ADR-065` 2절의 응답 모양 그대로다. 값을 더하지도 빼지도 않는다.

```text
   필드           출처                          NULL 취급        비고
   ────────────   ───────────────────────────   ─────────────   ──────────────────────
   node_id        nodes.node_id                 PRIMARY KEY      신원.  재시작해도 같다
   label          nodes.label                   NOT NULL         사람이 읽는 식별
   instance       nodes.instance                coalesce -> ""   ALTER 로 붙어 NOT NULL 이
                                                                아니다.  옛 enode 는 NULL
   capabilities   nodes.capabilities (jsonb)    NOT NULL         [{capability, attrs}] 그대로
   seen_at        nodes.seen_at                 NOT NULL         마지막 광고 시각
   expires_at     nodes.expires_at              NOT NULL         이 광고가 언제 죽나
   lease          leases (LEFT JOIN)            없으면 null      빈 객체를 안 낸다
   draining       nodes.draining (신설)          NOT NULL ''      "" · graceful · at-boundary
```

**principal 은 안 낸다** (`ADR-065` 2절 · `constraints.md` §4 가 제외 목록에 명시).
사람이 읽는 식별은 `label` 에 이미 있고, `principal` 은 식별이지 인증이 아니라
(`ADR-015` §1) 응답에 실으면 권한처럼 보인다. **제어판(panel)이 자기 신원을
보이는 것은 이 라우트가 아니라 `enode.Derive(config)` 의 로컬 유도다** —
`cmd/enodectl` 이 이미 그 방식을 쓴다. 부딪히지 않는다.

`lease` 는 값이 있거나 `null` 이다. 빈 객체를 내지 않는다 — 「임대 없음」과
「임대가 있는데 값이 비었다」가 갈려야 한다.

```text
   LeaseView
     run_id      leases.run_id      어느 Run 이 쥐었나
     not_after   leases.not_after   그 시점에 저장돼 있던 만료 시각
```

`leases.nonce` 는 안 낸다 — 허가 아티팩트의 비밀이고 관측이 쓸 값이 아니다.
`leases.capability` 도 안 낸다 — `ADR-019` 이후 항상 `agent.reason` 이라 줄마다
같은 글자가 반복될 뿐이다.

### 2.1 `not_after` 가 과거일 수 있다 — 카운트다운이 아니다

`not_after` 를 그대로 카운트다운으로 그리면 **0 아래로 내려가는 노드가 생긴다.**
`Reap` 이 만료 임대를 회수할 때 **`ASKED` 단계가 있는 Run 을 제외**하기 때문이다
(`internal/store/reap.go` 의 `NOT EXISTS (... st.state = 'ASKED')` · `ADR-047`).
사람을 기다리는 동안 자원을 놓지 않는 것이 그 결정이고, 그래서 그 Run 의 임대는
`not_after` 를 지나서도 무기한 남는다.

```text
   not_after > observed_at    곧 풀린다.  카운트다운이 뜻을 가진다
   not_after <= observed_at   이미 지났는데 남아 있다 — 「사람을 기다리는 중」이다
                              (GET /v1/asks 의 그 run_id 와 대조하면 확정된다)
```

**응답은 안 바꾼다** — 행이 있다는 것이 곧 점유라는 규칙은 `BusyNodes` 를 포함해
저장소 전체가 같고(`not_after` 로 거르지 않는다), `ADR-065` 도 「응답 시점에
저장돼 있던 사실」이라고 적었다. 바뀌는 것은 **읽는 쪽의 해석**이고, 그 대조 재료가
이미 응답 둘에 다 있다 — `lease.not_after` 와 `observed_at` 이다.

화면이 이 갈림을 안 그리면 `design/README.md` §2 의 카드 상태 ⑥(사람을 기다림)이
**음수 카운트다운**으로 나온다. ④(광고 만료 임박)와 헷갈리지 않는다 — 그쪽은
`expires_at` 기준이고 이쪽은 `not_after` 다. 다른 값이고 다른 상태다.

**데모 인스턴스에서는 이 확정이 반쪽이다** — `GET /v1/asks` 는 무인증 셋에 없다
(`business-logic-model.md` §7). 데모 화면은 `not_after <= observed_at` 까지만 보고
「사람을 기다리는 중일 수 있다」로 그린다. 확정하려면 asks 를 무인증 셋에 더해야
하고, 그것은 ADR-069 의 되묻기 표면을 여는 일이라 이 유닛의 결정이 아니다.

### 2.2 「지금 무슨 단계」는 두 홉이다 (답 5=A)

팩 3.1.1 은 S1 카드에 「지금 무슨 단계」를 그리라고 적었는데 **이 응답에는
단계가 없다.** 단계 이름과 회차는 `GET /v1/runs/{id}` 의 `steps[]` 에만 있다.

```text
   1  GET /v1/nodes          카드 경계 · 임대 유무 · not_after
   2  임대된 노드만           lease.run_id 로 GET /v1/runs/{id} -> CLAIMED 단계
```

**응답 모양을 안 넓힌다.** `ADR-065` §2.1 이 이 표면의 책임을 「광고와 임대의
관측 사실」로 그었고 단계는 그 둘이 아니다. `decisions.md` 2절의 「S3 현재 작업의
출처」가 제어판에 대해 **이미 같은 두 홉을 정해 두었다** — 관측 기준을 하나로
두는 것이 그 근거다. 임대된 노드만 도므로 홉의 수는 함대 크기가 아니라 **동시에
도는 Run 의 수**다.

**진행자에게 남기는 표시** — 팩 3.1.1 은 카드에 단계를 요구하고
`design/README.md` §2 는 「어느 Run · 언제 풀리나」까지만 적었다. 정본 둘이
어긋나 있고, 어느 쪽이 중앙 카드의 정본인지는 이 유닛이 정하지 않는다.

---

## 3. RunRow — 목록의 한 줄

`decisions.md` 2절이 정한 모양이고 `mediator-api.md` 의 `GET /v1/runs` 절이 같은
것을 이미 받아 적었다. **`steps` 는 안 싣는다** — 하나를 들여다보는 것은 상세다.

```text
   필드         출처                NULL 취급         비고
   ──────────   ─────────────────   ───────────────   ────────────────────────────
   run_id       runs.run_id         PRIMARY KEY
   state        runs.state          NOT NULL          어휘를 이 유닛이 고정하지 않는다
   verdict      runs.verdict        없으면 null       종료 전이면 null
   work_id      runs.work_id        coalesce -> ""    ALTER 로 붙어 NOT NULL 이 아니다.
                                                     이 질의가 저장소에서 이 열을
                                                     처음 SELECT 하는 자리다
   created_at   runs.created_at     NOT NULL          정렬 키
   ended_at     runs.ended_at       없으면 null       종료 전이면 null
   assigned     runs.assigned       NULL -> []        빈 배열이다.  null 이 아니다
   submitter    runs.submitter (신설) NOT NULL ''     게스트 로그인 이름.  없으면 ""
```

**`assigned` 는 값이 없으면 빈 배열이다.** `mediator-api.md` 의 `GET /v1/runs`
예시가 QUEUED 행에 `"assigned": []` 로 적어 두었고, `runs.assigned` 는 nullable
jsonb 이라 `CreateRejectedRun` 이 만든 행은 오늘도 NULL 이다. `null` 이나 키 생략을
내면 화면이 세 모양을 다 받아내야 하고, 그것은 `StepView.Needs` 가 빈 배열을
언제나 내보내는 이유(「안 내보내면 읽는 쪽이 기본값 규칙을 추측한다」)와 정면으로
어긋난다. **CP2 의 「QUEUED 행이 어느 카드에도 안 얹혀 있다」를 화면이 이 값으로
판정한다.**

### 3.1 queue 에 넘기는 계약 한 줄 — `assigned` 는 jsonb null 로 들어오면 안 된다

`coalesce(assigned, '[]'::jsonb)` 는 **SQL NULL 만 접는다.** jsonb 의 `null` 은
SQL NULL 이 아니라 값이므로(`'null'::jsonb IS NULL` 은 거짓) 그대로 통과해 와이어에
`"assigned": null` 로 나간다. 읽기 쪽에서 막을 방법이 없다.

```text
   CreateRun            label() 이 make([]Assigned, 0, n) 을 낸다 — non-nil 이라 [] 다
   CreateRejectedRun    열을 INSERT 목록에서 아예 뺀다 — SQL NULL 이라 접힌다
   CreateQueuedRun      아직 없는 심볼이다.  store.Run{} 제로값을 그대로 쓰면
                        nil 슬라이스가 json.Marshal 에서 null 이 되고 그것이 앉는다
```

**`assigned` 는 열을 생략하거나 `[]` 로 넣는다.** 이것을 W0 이 적는 이유는 증상이
나는 자리가 멀기 때문이다 — 값은 queue(W1)의 INSERT 가 넣고, 깨지는 것은 CP2 의
화면 판정(W1·ui)이며, 읽는 질의(obs)는 아무 잘못이 없다.

`verdict` 를 목록에 싣는 이유는 하나다 — drain 으로 닫힌 `FAILED`(사유
`drain:<node_id>`)와 진짜 실패를 **목록에서** 가르는 열이 그것뿐이다.

### 3.2 `submitter` 는 목록에만 산다 (답 3=A)

`GET /v1/runs/{id}` 에는 **안 더한다.** 팩이 요구한 자리가 목록과 RUN 카드이고
(`decisions.md` §8.6), 데모 모드는 작업 그래프로 전환해도 run 목록을 안 가리므로
(CP9) 화면이 이름을 계속 쥔다.

**키는 언제나 있고 값이 빈다.** 실 함대 Run 은 `"submitter": ""` 로 나간다.
목록 행의 기존 관행이 그것이다 — `verdict` 와 `ended_at` 이 종료 전에 `null` 로
적혀서 나간다.

**ui 에 넘기는 계약 한 줄** — 상세에는 이름이 없다. RUN 카드가 상세를 그리면
이름은 **목록에서 이어 붙인다.** 딥링크로 상세에 바로 들어오는 경로(새로고침 ·
링크 공유 · `limit` 밖으로 밀려난 오래된 Run)는 목록을 먼저 읽어야 이름이 산다.
이것을 뒤집으려면 `GetRun` 의 SELECT 와 `runView` 에 두 줄이면 되지만, 팩이 요구한
자리가 아니므로 이 유닛은 안 연다.

**진행자에게 남기는 표시 — `submitter` 는 설계 정본에 없다.** `enode-design` 전체에
그 낱말이 없고(`grep` 실측 0건), `GET /v1/runs` 의 응답 모양은 `ADR-064` §4 와
`mediator-api.md` 가 일곱 열로 적어 두었다. `decisions.md` 머리말이 그 모양을
**「팩이 고를 자리가 아니다 — 벗어날 수 없다」**로 못 박았는데 같은 팩의 §8.2 Q3 이
뒤에 열 하나를 더했고, `canon.md` §4 의 「이 팩이 정본에 더하는 것」 둘에도
`submitter` 가 없다. **이 유닛이 그 여덟째 열을 낳으므로 표시를 남기는 자리도 여기다** —
`mediator-api.md` 와 `ADR-064` §4 를 누가 언제 고쳐 올릴지는 진행자가 정한다.
구현이 병합되는 순간 정본과 구현이 한 열만큼 갈린다.

---

## 4. RunFilter · RunsQuery — 좁히기만 한다

같은 값 넷이 서버와 클라이언트 양쪽에 산다. 이름을 갈라 둔다.

```text
   필드    형             RunFilter (internal/store)   RunsQuery (internal/runctl)
   ─────   ────────────   ──────────────────────────   ──────────────────────────
   State   string         runs.state 등호               ?state=      "" 면 뺀다
   Since   time.Time      runs.created_at >=            ?since=      IsZero 면 뺀다
   Work    string         runs.work_id 등호             ?work=       "" 면 뺀다
   Limit   int            LIMIT                         ?limit=      0 이면 뺀다
```

**형을 여기서 못 박는다.** `Since` 가 `time.Time` 이므로 「빈 값」의 판정은
`IsZero()` 다 — `string` 이면 클라이언트가 형식 검사를 지게 되고, 그것은 아래
「클라이언트는 검증하지 않는다」와 부딪친다. 서버는 질의 문자열을 RFC 3339 로
파싱해 `RunFilter.Since` 를 채운다. `Limit` 의 기본 100 과 상한 1000 은 **api 가**
적용해 `RunFilter` 에 이미 정해진 수를 넣는다 — store 는 받은 수를 그대로 건다.

**`since` 는 포함 하한이다** — `created_at >= since`. `mediator-api.md` 가
「`created_at` 하한」이라고만 적어 경계가 열려 있었다. 포함으로 못 박는다:
같은 초에 만들어진 Run 이 폴링 사이에 사라지지 않는 쪽이다.

**`RunsQuery` 는 빈 값과 0 을 질의에서 뺀다.** 이것이 규칙이 아니라 계약인 이유 —
Go 의 `RunsQuery{}` 제로값을 그대로 펴면 `?limit=0` 이 되고, 서버는 그것을
`400` 으로 거절한다(`business-rules.md` §2.1). **인자 없이 부르는 것이 `runs.list`
의 기본 사용법**이므로 그 순간 MCP 의 첫 호출이 실패한다.

**`?work=` 는 응답의 `work_id` 로 되돌아가지 못하는 행이 있다.** 응답은
`coalesce(work_id,'')` 로 접어 내는데 필터는 접기 전 열에 등호를 건다. `work_id` 가
NULL 인 옛 행은 응답에 `""` 로 보이지만 어떤 `?work=` 값으로도 안 잡힌다. 접은
값에 등호를 걸면(`coalesce(work_id,'') = $`) `runs_work_idx (work_id, created_at)`
를 못 타므로 **접기 전 열에 건다.** `""` 로 거르는 것은 「work 가 없는 것 전부」라
뜻이 없는 질의이고, 그 하나를 위해 유일한 인덱스를 버릴 이유가 없다.

**클라이언트는 검증하지 않는다.** 상한도 형식도 서버가 진다 — 두 곳에서 검사하면
어긋났을 때 화면이 이유 없는 400 을 받는다.

**`principal` 필터는 없다** — `ADR-065` 와 같은 이유다. 거르는 값으로 내면 그
필드가 권한처럼 보인다. **node 필터도 없다** — `enode-features.md` 3.1.3 이
「`assigned[].nodes[].node` 로 화면에서 거른다 · Mediator 쪽 필터를 신설하지
않는다」로 이미 정했고, `runs.assigned` 가 노드가 붙는 모든 자리에서 자라므로
(`claim.go` 의 「원천은 runs.assigned 다 — steps 가 아니다」) 그 거르기가 성립한다.

**뒤로 걸어갈 수단이 없다.** 정렬이 `created_at` 내림차순이고 페이지네이션이
없으므로(`ADR-065` §4 · `decisions.md` 2절), `limit` 상한 밖의 오래된 Run 은
어떤 인자로도 못 꺼낸다. transcript(W4)의 「지난 작업」은 **함대 전체 최근 1000 건
안에서만** 참이다. 그것을 넘겨야 하면 페이지네이션을 여는 결정이 먼저다.

---

## 5. StepView 에 더하는 한 필드 — chosen

`steps.chosen` 열은 이미 있고 살아 있다 — `internal/store/dispatch.go` 가
`SET chosen = true` 를 쓰고 `verdict.go` 가 이미 읽는다. 뷰가 그것을 안 싣고
있었다. 더하는 것은 한 필드다.

```text
   chosen   steps.chosen   이 단계가 갈림길에서 골라진 적이 있는가
```

**언제나 싣는다.** `false` 를 생략하면 `SKIPPED` 하나가 두 가지를 다시 뜻하게
된다 — 「경로가 갈려 안 갔다」와 「골랐는데 못 닿았다」. 그 둘을 가르려고 만든
열이므로 뷰에서 다시 붙이면 열이 무의미해진다.

**`POST /v1/runs` 응답도 함께 바뀐다.** `runView` 를 짓는 `view()` 를 제출 경로가
같이 쓰기 때문이다(호출자 넷). 그것이 옳다 — `202`(QUEUED)는 `201` 과 같은 모양
이어야 하므로(`ADR-064` · `mediator-api.md`), 둘이 함께 움직이는 쪽이 그 규칙을
지킨다.

**`runctl.Step` 에도 더한다.** 그 구조체는 `internal/runctl/client.go` 의 별도
타입이고 `runctl status` 와 mcp 의 `run.get` 이 그것으로 디코드한다. 안 더하면
`chosen` 이 CLI 와 MCP 표면에서만 조용히 사라진다.

**`started_at` 도 같은 자리에서 더한다 — panel(W3)이 그것 없이 못 선다.**
서버의 `store.StepView` 는 `started_at` 과 `ended_at` 을 이미 내는데
`runctl.Step` 은 `Seq · ID · State · Uses · Node · Needs · Attempt` 뿐이다.
`decisions.md` 2절이 S3 「현재 작업」의 출처를 두 홉(`lease.run_id` 로 상세를 부른다)
으로 정했고 `design/README.md` §3 은 그 자리에 **단계 이름 · 회차 · 시작 시각**을
요구한다. 앞의 둘은 `ID` 와 `Attempt` 로 이미 서지만 시작 시각은 안 선다.

담당이 다르므로(panel 은 nacl1119) 「자기 필요만큼 넓힌다」가 여기서는 안 통한다 —
그 파일은 행렬이 「단독 obs」로 세워 진행자가 직렬 대상으로 보지 않는다. **한 필드를
W0 이 세우는 것이 W3 에 다른 담당이 남의 파일을 여는 것보다 싸다.**

---

## 6. RequireView — 무엇을 기다리는가

`ADR-069` 가 여는 여섯째 물음(「왜 아직 안 갔나」)의 재료다. **새 라우트를 만들지
않고** `GET /v1/runs/{id}` 를 넓힌다.

```text
   필드         생략        뜻
   ──────────   ─────────   ────────────────────────────────────────────────────
   as           언제나 있다  별칭.  steps[].uses 와 이어야 어느 단계가 무엇을
                            기다리는지 붙는다
   capability   언제나 있다  능력 이름 (ADR-019 이후 사실상 agent.reason)
   count        0 이면 생략  몇 대를 원하나.  없으면 1 로 읽는다
   attrs        언제나 있다  속성 사상 {키: 값}.  비면 {} 를 낸다
```

**생략은 `count` 하나뿐이다.** `attrs` 가 비면 `{}` 를 낸다 — 키를 지우면 읽는 쪽이
기본값 규칙을 추측하고, 그것이 `assigned` 와 `chosen` 을 언제나 싣기로 한 이유와
같은 논거다. `count` 만 다른 이유는 계약 입력이 이미 그 모양이기 때문이다:
`contract.Require` 가 `count` 를 `omitempty` 로 두고 「없으면 1」이 그 자리의
기본값 규칙이라 정본이 이미 정해 두었다.

**`attrs` 로 감싼다** (답 2=A). `ADR-069` §4 의 예시가 `as` · `capability` ·
`attrs` 셋으로 그 모양을 보인다. **`count` 는 그 예시에 없다** — 계약의 필드이고
(`run-contract.md`) 관측이 그것을 그대로 옮기는 것이다. 예시가 셋이라고 넷째를
빼면 「몇 대를 기다리나」가 화면에서 사라진다.

```jsonc
"requires": [
  { "as": "b", "capability": "agent.reason",
    "attrs": { "harness": "claude", "os": "linux/amd64" } }
]
```

**이 모양은 관측 전용이다.** 계약 **입력**의 `requires[]` 는 속성을 형제 키로
편다(`contract.Require` 의 Marshal/Unmarshal). 두 모양이 같은 개념을 지고,
변환기는 코드 어디에도 없다. 관측 응답을 그대로 계약 본문에 복사하면
`requires[].attrs must be a string` 이라는, **원인을 안 가리키는 400** 이 난다.
화면은 읽기만 하므로 안 밟지만, mcp 의 `run.submit` 은 사용자의 Claude 가 지은
계약을 그대로 나르므로 밟을 수 있다 — **mcp 가 이 문장을 `run.submit` 도구
설명에 싣는다.**

출처는 **지금 유효한 계약**이다 — `coalesce(contract_versions -> -1, contract)`.
계약이 여러 판이면 마지막 판이다(`ADR-069` §4). 제출 전문만 보면 재계획이 지은
요구가 빠진다. 판이 쌓여도 `requires` 가 살아남는 것은 확인했다 —
`ContractVersion` 이 `contract.Contract` 를 익명으로 품어 평평하게 마셜되고,
판을 만드는 세 자리가 전부 이전 계약을 값으로 복사한다.

**상태로 응답 모양을 안 가른다** — `QUEUED` 일 때만 싣지 않는다(`ADR-069` §4).
**라우트로는 가른다** — `business-logic-model.md` §4 가 그 자리다. `view()` 를
타는 제출 응답 셋에는 안 싣고 `getRun` 에서만 채운다.

---

## 7. 새로 생기는 자리 셋

전부 additive 다. 기존 행의 뜻을 바꾸지 않는다.

### 7.1 nodes.draining (답 1=C)

```text
   형      text NOT NULL DEFAULT ''
   어휘    "" (안 걸림) · graceful (새 임대만 막음) · at-boundary (경계에서 닫음)
   정본    노드의 정책 파일.  이 열은 최근 광고에 실려 온 것의 복사본이다
   쓰는 쪽  UpsertAdvert (obs 가 넣는다 · §7.3 의 어휘 검사를 거친다)
   읽는 쪽  GET /v1/nodes (obs) · store.DrainingNodes (queue · W1)
```

**읽는 쪽이 둘이다.** queue 의 `DrainingNodes(ctx, tx) (map[string]bool, error)`
가 이 열을 불리언으로 접어 매칭의 `busy` 에 합친다. **W1 이 착수할 때 이 열이
이미 서 있다는 것이 답 C 의 값이다** — 유닛 정의는 이 열을 drain(W2)의 「이미
있음」으로 적었고, 그대로 갔으면 W1 이 없는 열을 읽었다.

DB CHECK 제약을 걸지 않는다 — `steps.state` 가 CHECK 를 안 거는 것과 같은 이유로
어휘가 늘 때 마이그레이션을 강요하지 않는다. **대신 애플리케이션이 검사한다**
(§7.3). 그 둘은 다른 것이고, `decisions.md` 3절이 요구한 것은 뒤쪽이다.

### 7.2 runs.submitter

```text
   형      text NOT NULL DEFAULT ''
   뜻      제출자 표시 라벨.  게스트 로그인 이름(랜덤 2단어)이 그대로 실린다
   정본    decisions.md §8.6 — 새 작업 모달에 이름 칸을 따로 두지 않는다
   읽는 쪽  GET /v1/runs 목록 (obs)
```

**권한이 아니다.** `principal` 과 같은 성격의 자기 신고이고(`ADR-015` §1) 거르는
값으로 쓰지 않는다 — `RunFilter` 에 `submitter` 가 없는 이유다.

**쓰는 입구를 obs 가 연다.** 열만 만들고 입구를 안 정하면 queue(W1)와
demo-back(W2)이 같은 `submit()` 을 두 번 고친다. 그래서 셋을 W0 에 세운다.

```text
   store.Run 에 Submitter string       CreateRun 의 INSERT 열에 든다
   internal/api 에 요청 컨텍스트 키       principal(r) 과 같은 모양의 submitter(r)
   submit() 의 store.Run 리터럴 세 자리   Submitter: submitter(r).  실 함대는 언제나 ""
   CreateRejectedRun 의 INSERT 열       거절 경로도 같은 열에 쓴다
```

**`submit()` 의 리터럴은 하나가 아니라 셋이다** — 매처 거절 · 폭 상한 초과 · 정상.
앞의 둘은 `CreateRejectedRun` 을 타고 그 INSERT 는 열 여섯(`run_id · state ·
principal · contract · reject · ended_at`)이라 `submitter` 가 없다. 정상 경로에만
더하면 **거절된 Run 의 제출자 이름이 영원히 빈다.** 데모에서 가장 흔한 실패가
「rpi 가 꺼져 있어 422」이고 그것이 CP9 가 재는 바로 그 행이다.

이러면 demo-back 은 `postDemo` 에서 게스트 이름을 컨텍스트에 넣기만 하고
`submit()` 본문을 **한 줄도 안 고친다** — queue 의 diff 와 겹치지 않는다.
그것이 이 넷을 W0 에 두는 이유다.

**스텝 주입은 demo-back 몫이다**(`unit-of-work.md` §8). obs 는 열과 입구와 읽기까지다.

`CreateQueuedRun(ctx, tx, contract, submitter string)` 은 **아직 없는 심볼**이다 —
queue(W1)의 겉면이고, 그 인자가 이 입구에서 값을 받는다.

### 7.3 contract.Advert 의 정책 — 정본은 `policy.drain` 이다

답 1=C 를 이행하려면 `UpsertAdvert` 가 복사할 값이 광고에 있어야 하는데
`contract.Advert` 에 정책 필드가 없다. obs 가 연다. **모양은 정본이 이미 정했다.**

```text
   POST /v1/nodes  광고 본문    "policy": { "drain": "" }
   POST /v1/nodes  응답        "drain": ""          중앙이 받아 적었다는 통보
   GET  /v1/nodes  노드 항목    "draining": ""       관측 사실
```

세 자리의 이름이 다른 것은 우연이 아니라 `ADR-063` §6 이 그렇게 못 박은 것이고
`mediator-api.md` 가 셋 다 적어 두었다.

```go
type Policy struct {
    Drain string `json:"drain"`
}
// Advert 에
Policy Policy `json:"policy,omitzero"`
```

**「값 하나이므로 구조체로 안 감싼다」는 앞 판의 논거를 접는다.** 감싸는 모양을
obs 가 짐작하는 것이 아니라 `ADR-063` §6 이 이미 정해 두었다. 지금 필드 수는
여전히 하나다. (`ADR-063` §2.1 은 「자리를 남겨두는 것들 — 지금 만들지 않는 이유를
함께 적는다」의 목록이지 `policy` 안의 자리를 잡아 둔 절이 아니다. 근거는 §6 이다.)

**응답의 `drain` 도 obs 가 낸다.** `advertResponse` 는 `internal/api/api.go` 에
있고 그 파일의 행렬에 drain 유닛이 없다 — obs 가 안 내면 drain 이 행렬 밖 접점을
W2 에 다시 연다. 받은 값을 그 자리에서 되돌리는 것은 새 의미가 0 이다.

**응답의 `drain` 은 언제나 나간다 — `omitempty` 가 아니다.** `Advert.Policy` 의
`omitzero` 는 **요청 본문**의 규칙이고(안 걸린 노드가 `policy` 를 안 보낸다),
응답은 반대다. drain(W2)이 이 값을 받아 Worker 에 넘기므로, 생략되면 「해제됐다」와
「이 필드를 모르는 중앙이다」가 한 글자가 된다. 통보는 값이 비어도 통보다.

**어휘를 검사하는 것은 store 다.** `UpsertAdvert` 가 쓰기 전에 셋(`""`·`graceful`·
`at-boundary`)을 보고, 벗어나면 **`""` 로 접어 저장하고 로그에 남긴다.**
핸들러가 아니라 store 인 이유가 둘이다 — 이 열에 쓰는 자리가 하나뿐이라 그 자리에
붙이면 앞으로 생길 다른 쓰기도 함께 걸리고, `internal/api` 의 커버리지 여유가
**5.6 문장**뿐이라(`business-rules.md` §7.6) 검사와 로그 문장을 그 예산에 얹을
자리가 없다. `internal/store` 의 여유는 31.2 문장이고 `Store.Log` 는 이미 걸려 있다.
`400` 을 내지 않는 이유 —
광고는 하트비트를 겸하므로(`ADR-016`) 정책 값 하나로 광고 전체를 거절하면 노드가
함대에서 사라진다. 검사가 필요한 이유 — queue 의 `DrainingNodes` 가 이 열을
`map[string]bool` 로 접으므로 어휘 밖 문자열이 들어오면 **그 노드가 매칭 후보에서
조용히 빠진다.** 증상은 「능력은 있는데 계속 QUEUED」이고 원인은 눈으로만 보인다.

**옛 enode 는 언제나 `""` 를 쓴다.** 광고가 매번 전부이므로(`ADR-012` ·
`ADR-017` 결정 3) 정책도 통째 교체가 옳고, 빼고 보내는 것이 곧 「지금은 안 걸려
있다」다.

**그 대가를 정본이 이름으로 거절했다 — 진행자에게 남기는 표시다.** 필드를 모르는
판으로 되돌아간 enode 는 걸린 drain 을 지우고, 그 노드는 다음 광고 주기에 다시
후보가 된다. `ADR-063` §4 가 정확히 그것을 막았다 — 「`drain` 은 임대가 다 빠졌다고
저절로 풀리지 않는다. 파일에서 지워야 다시 후보가 된다. **저절로 풀리면 소유자가
되찾은 보드를 다음 광고 주기에 도로 빌려주는 셈이 된다**」. `decisions.md` 1절의
「자동 복귀는 「돌려받았다」를 무효로 만든다」도 같다.

대안은 「`policy` 키 부재를 변경 없음으로 읽는다」인데 그것은 `ADR-012` 의 「광고는
매번 전부」를 이 필드 하나에서 깨는 것이고, 그러면 **소유자가 파일에서 지워도
중앙이 안 푼다** — 반대 방향의 같은 고장이다. 어느 쪽이든 정본 하나를 접어야 한다.

**obs 는 「매번 전부」쪽으로 간다** (`ADR-012` 가 프로토콜 불변식이고 §4 는 정책의
수명 규칙이다). 다만 그것이 **저울질 없는 수락이 아니라는 것**을 적어 둔다 —
노출 창은 옛 판이 도는 동안이고, 닫는 값은 drain(W2)이 enode 를 올린 뒤 사라진다.
`ADR-063` §4 를 그대로 지키려면 광고에 「이 판은 policy 를 안다」는 표시가 필요하고
그것은 `ADR-068` 이 거절한 종류의 프로토콜 필드다. **진행자가 이 저울을 본다.**

**`LiveAdverts` 는 이 필드를 안 채운다.** `contract.Advert` 는 광고 수신과 DB 읽기
둘 다에 쓰이는데, 매처에 들어가는 `Advert` 의 정책 필드는 draining 노드에서도
언제나 제로값이다. **그것이 옳다** — 매처는 draining 을 `busy` 로 받는다(queue).
비대칭이므로 필드 주석에 남긴다. 주석은 하네스로 안 나가므로 한국어다
(`CONVENTIONS.md` 2.2).

**`internal/contract/advert.go` 는 어느 유닛의 파일 행렬에도 없다.** 행렬 밖
파일을 만진 diff 로 가져간다 (`CONVENTIONS.md` 3.5). 계획 6.1 에 그 줄을 적었다.

---

## 8. 겉면 (Go 시그니처)

유닛 정의 §1 의 겉면에 이 문서가 정한 형을 채운 것이다. **둘은 그 겉면과 다르다** —
`Nodes` · `Runs` 가 반환을 하나 늘렸고(아래 「시각을 값으로 함께 돌려준다」),
`UpsertAdvert` 가 반환을 하나 늘렸다. 셋 다 이 문서의 결정이고 유닛 정의를 넘는다.

```go
// internal/store
func (s *Store) Nodes(ctx context.Context) ([]NodeView, time.Time, error)
func (s *Store) Runs(ctx context.Context, f RunFilter) ([]RunRow, time.Time, error)
func (s *Store) UpsertAdvert(ctx context.Context, a contract.Advert,
        principal string, ttl time.Duration) (prevDrain string, err error)

// internal/runctl — 원문 그대로 돌려준다 (mcp 의 글자 일치 · CP5)
func (c *Client) Nodes(ctx context.Context) (json.RawMessage, error)
func (c *Client) Runs(ctx context.Context, q RunsQuery) (json.RawMessage, error)
```

**새로 짓지 않는 것 하나를 적어 둔다** — `Store.LiveContract` 는 이미 있다
(`internal/store/observe.go`, 오늘 세 자리가 부른다). `getRun` 이 `requires` 를
꺼낼 때 그것을 그대로 탄다. 새 메서드가 아니므로 이 유닛의 커버리지 예산에
그 본문이 들지 않는다 — 느는 것은 반환을 `RequireView` 로 옮기는 사상뿐이다.

**시각을 값으로 함께 돌려준다.** `observed_at` 을 행에 얹으면 **결과가 0 행일 때
값이 없다** — 빈 함대와 아무것도 안 맞는 필터가 정확히 그 경우이고, 이 유닛은
그때도 `observed_at` 을 내기로 했다. 그래서 DB 시계를 행과 따로 받는다
(`business-logic-model.md` §2).

**`UpsertAdvert` 가 이전 `draining` 을 돌려준다.** `decisions.md` 1절이 정한
`WakeQueued` 호출 지점 여섯 중 하나가 **「drain 해제를 받은 광고 처리」**인데,
덮어쓰기만 하면 해제를 본 사람이 아무도 없다. 이 반환이 그 트리거다. obs 는
그 값을 안 쓴다 — **버릴 값을 W0 이 세워 두는 것**이고, 그래야 drain(W2)이
store 접점을 다시 안 연다. 한 문장 안에서 낸다.

```sql
WITH old AS (SELECT draining FROM nodes WHERE node_id = $1)
INSERT INTO nodes (...) VALUES (...)
ON CONFLICT (node_id) DO UPDATE SET ...
RETURNING coalesce((SELECT draining FROM old), '')
```

**`coalesce` 는 서브쿼리 밖이다.** 처음 광고하는 노드는 `old` 가 0행이라 스칼라
서브쿼리가 SQL NULL 을 내고, `prevDrain string` 으로 스캔하면 그 자리에서 죽는다.
`nodes.node_id` 가 PRIMARY KEY 이므로 0행인 경우는 정확히 **「이 노드의 첫 광고」**다 —
안 접으면 어떤 노드도 함대에 처음 들어오지 못하고 `POST /v1/nodes` 가 `503` 을 낸다.
시험도 함께 죽는다: `newServer` 가 매번 `Truncate` 하므로 모든 광고가 첫 삽입이다.

`(SELECT coalesce(draining,'') FROM old)` 로는 안 접힌다 — 0행이면 여전히 NULL 이다.
새 노드의 이전 값은 `""` 이고, 그것은 「걸린 적 없음」과 같은 글자여야 한다.
drain(W2)이 그 둘을 가를 이유가 없다.

`RETURNING OLD.*` 은 PostgreSQL 18 부터다. 이 저장소는 17 이므로 CTE 가 유일한 길이다.
CTE 는 문장 시작 스냅숏을 보므로 갱신 전 값이 나온다. 질의는 여전히 하나다.

`Nodes` 는 필터를 안 받는다 — `ADR-065` 가 여유 질의를 안 만들기로 했고, 인자가
없는 것이 그 결정의 코드상 표현이다.

`runctl.Client` 의 둘이 파싱한 타입이 아니라 `json.RawMessage` 인 이유는 mcp 의
`fleet.list` 가 `GET /v1/nodes` 와 **글자까지 같아야** 하기 때문이다(CP5).
파싱해서 다시 마셜하면 키 순서와 생략 규칙이 그 사이에서 갈린다.

**파싱하는 타입 둘도 함께 넓힌다** — `runctl.Run` 에 `Requires`, `runctl.Step` 에
`Chosen` 과 `StartedAt`. 앞의 둘을 안 넓히면 `runctl status` 와 mcp 의
`run.get`(정본이 `Client.Status` 로 못 박았다)에서 이 유닛이 연 값이 조용히
사라진다. `StartedAt` 은 이 유닛이 연 값이 아니라 **panel(W3)이 필요로 하는데
클라이언트에만 없던 값**이다 — §5 가 그 이유를 적는다.

**`internal/runctl/client.go` 를 mcp 가 넓힐 수 있다.** 행렬은 이 파일을
「단독 obs」로 세웠지만, `asks.list` 의 글자 일치(CP7)는 원문 반환이 하나 더
필요하다 — 기존 `runctl.AskItem` 이 서버의 `AskView` 와 필드가 어긋나
(`asked_at` 이 없고 `omitempty` 넷이 빠졌다) 재마셜에서 값이 실제로 사라진다.
**담당이 같으므로(taeels) mcp 가 W1 에 자기 필요만큼 넓힌다** — obs 가 안 쓸
메서드를 미리 짓지 않는다. 행렬의 「단독 obs」 옆에 그 사실을 적는다.

**panel(W3)은 다르다 — 담당이 갈린다.** 그래서 `Step.StartedAt` 은 미루지 않고
W0 에서 세운다(§5). 「단독 obs」인 파일을 다른 담당이 W3 에 여는 것이 이 한 필드보다
비싸고, 진행자가 그 파일을 직렬 대상으로 보고 있지 않아 부딪히면 늦게 보인다.
