# 컴포넌트 — 정의와 책임

입력은 `requirements.md` 4절(FR-1 ~ FR-8) · `user-stories.md` 3절(NC-1 ~ NC-6) ·
`plans/application-design-plan.md` 6절의 답 넷(전부 A)이다.

**일곱이다.** `requirements.md` 7.3 이 여섯을 적었고 이 단계가 하나를 더 찾았다 —
`internal/runctl`. 제어판이 Mediator 를 부르는 유일한 길이다.

---

## 1. `internal/transcript` — 신규. 사건의 모양을 지는 하나

**책임**: 하네스가 낸 바이트를 사건 열로 읽고, 걸러진 껍데기의 모양을 짓는다.
**짓는 쪽과 읽는 쪽이 한 패키지에 산다** (Q4 = A) — 두 벌로 두면 조용히 갈린다.

```text
   아는 것    stream-json 줄의 모양.  껍데기 줄의 모양.  enode.elided 줄.
             첫 줄이 잘린 입력(링이 감겼다).  못 읽은 줄은 raw 다
   모르는 것   HTTP · 파일 · 링 · Run · 단계 · 시계 · 화면.
             **시계를 모르는 것이 Q3 = A 다** — 경과는 화면이 뺀다
   임포트     표준 라이브러리만.  enode · api · store · panel 금지
```

**들어오는 입력이 넷이다.** 앞 판이 둘로 적었고 U1 의 Functional Design 이
코드를 읽어 둘을 더 찾았다 — **명령 단계는 JSON 을 한 줄도 안 낸다.**

```text
   원문 스트림    진행 파일 · 링.  stream-json 사건 전문
   선별본        봉인된 logs/NN-*.log.  init 전문 · result 전문 · 껍데기 줄 ·
                 enode.elided 한 줄 · enode.capped 한 줄 · stderr 꼬리
   명령 단계     평문이다.  claim.go 의 링과 로그에 오늘도 그대로 흐른다
   stderr 꼬리    평문이다.  선별본 끝에 붙는다
```

**뒤의 둘이 파싱 실패가 아니라 애초에 JSON 이 아니다.** 팩의 `raw` 정의
(「파싱 실패 줄」)가 이 둘을 안 덮으므로 U1 의 규칙이 `text` 와 `raw` 를 갈랐다.

**실패 등급**: 없다. 파서는 못 읽은 줄을 `raw` 로 넘기고 아무것도 안 버린다
(`constraints.md` §4). 오류로 단계를 죽이는 경로가 이 패키지에 0 이다.

**옮겨 오는 것은 셋이 아니라 함수 일곱과 타입 둘이다** (Q4 = A · 오늘
`internal/enode/runner.go` 에 있다). 아래 셋이 이름이 난 것이고, 그 셋은
`usageTokens` · `eventString` · `eventBool` · `eventInt` 와 타입 `logShell` ·
`elidedMark` 없이 안 선다. `splitLines` 는 `Parse` 와 `selectLogs` 가 둘 다
필요하다. 전문은 U1 의 `domain-entities.md` 1절.

```text
   parseEventLine   줄 하나를 아는 키만 읽는다.  SECURITY-13 의 규율이 여기 있다
   eventShell       본문을 걷은 껍데기를 짓는다.  selectLogs 가 쓴다
   elidedMarker     「N 개 · B 바이트를 걷었다」한 줄을 짓는다
```

---

## 2. `internal/enode` — 사건을 만드는 쪽

**는 책임 셋.**

```text
   Decode 가 배출한다 (FR-1)   줄 단위로 읽어 emit 한다.  시그니처는 안 바뀐다.
                              HarnessResult 는 마지막 result 줄에서 그대로 나온다.
                              봉투 파서(ParseClaude · lastJSONObject)는 안 바꾼다
   링 tee 를 되살린다 (FR-2)    runner.go:186 의 자리.  claim.go:794 가 이미 넘기는
                              Job.Transcript 를 받는다.  선별을 링에는 안 건다
   업로더 (FR-5)               새 겉면.  같은 tee 한 겹에서 갈라진 바이트를 모아
                              청크로 올린다.  자기 고루틴 · 자기 버퍼 · 자기 오프셋
   selectLogs 가 파서를 쓴다    셋을 transcript 로 옮기고 임포트한다 (Q4 = A)
```

**실패 등급.**

```text
   치명    Decode 의 실패는 오늘과 같다 — HarnessResult 가 크래시로 나온다
   보조    링 쓰기 실패.  오늘과 같다 (ring == nil 이면 tee 가 nil)
   보조    업로더의 PUT 실패.  **실행도 링도 안 막는다** (FR-5).
           다음 청크에 이어 보낸다.  상한에 닿으면 더 안 올리고 노드 로그에 남긴다
```

---

## 3. `internal/record` — 진행 파일의 주인

**는 책임 넷.**

```text
   진행 파일을 쓴다        기록 디렉터리 **밖**의 형제 트리다 (Q2 = A).
                          <Root>/progress/run-<safe(id)>/NN-<name>.log
   시도가 바뀌면 비운다     Q1 = A.  총 길이가 0 으로 돌아간다
   총 길이를 낸다          쓴 뒤의 총 길이와 지금 시도와 상한 도달 여부
   봉인 때 트리를 지운다    Run 하나의 진행 트리를 통째로.  Seal 이 부른다
```

**왜 기록 디렉터리 밖인가** — `seal(dir)` 이 `Walk` 로 모든 파일을 `0444`,
모든 디렉터리를 `0555` 로 만든다 (`record.go:156`). 오늘 보관 정책이 없어
(`record.go:196` 의 주석) **한 번 굳으면 아무도 못 지운다.** 안에 두고 지우기를
놓치면 그 Run 의 원문이 영구가 된다. 밖에 두면 `Tar` 가 구조상 못 보고
`seal` 이 못 닿는다.

**이름이 안 부딪친다** — Run 디렉터리는 `run-<safe(id)>` 다 (`record.go:33`).
형제 `progress/` 와 겹칠 수 있는 `run_id` 가 없다.

**남는 대가 하나** — Run 트리를 지울 때 형제 트리가 남을 수 있다. **그 값이
N2 다** (NFR Requirements). 여기서 안 닫는다.

**`AppendLog` 는 자리를 지킨다** — 봉인되는 `logs/` 를 계속 쓴다. 바뀌는 것은
반환값의 뜻뿐이다 (이번 호출 바이트 -> 총 길이 · D4).

---

## 4. `internal/api` — 표면 둘

```text
   GET log (FR-6)    신규 라우트 하나.  새 파일이다.  api.go 는 등록 줄 하나만 는다.
                     봉인 전이면 진행 파일, 뒤면 logs/.  출처를 헤더가 말한다
   PUT log 의 갈림     기존 라우트에 쿼리 하나.  새 라우트를 안 만든다 (D2).
                     CB0 의 셈 17 -> 18 이 그 전제다
```

**실패 등급**: 둘 다 요청 하나의 실패로 끝난다. 실행을 안 막는다.

**입력 검증** (SECURITY-05): `seq` 는 양의 정수 · `from` 은 음이 아닌 정수 ·
`as` 는 `raw` · `events` 둘뿐 · `attempt` 는 음이 아닌 정수. 그 밖은 `400`.

---

## 5. `internal/panel` — 노드 소유자의 화면

```text
   도는 것    로컬 링을 1초로 읽는다.  오늘 그대로 (transcript.go:20).
             달라지는 것은 링의 **내용이 NDJSON 이 된 것**이고, 카드가 파서를 쓴다
   지난 것    출처를 GET 라우트로 바꾼다.  tar 를 통째로 푸는 우회를 걷는다.
             **archive/tar 임포트가 이 패키지에서 사라진다**
   헤더 다섯   page.go:15 에 /ui/ 와 같은 보안 헤더를 건다 (requirements.md 5.5)
   본문은 텍스트로만   page.go 가 이미 textContent 와 esc() 를 쓴다 (SECURITY-05)
```

**`verdict` 는 지금처럼 Run 목록에서 온다** — `handleRuns` 가 그대로 낸다.

---

## 6. `internal/api/ui` — 진행자의 화면

```text
   Run 상세의 단계 카드   도는 단계는 2초 폴링으로 자라고 끝난 단계는 한 번 읽는다.
                        목록 폴링(client.mjs:67 의 5초)과 **별개 타이머**다
   표시 규칙             제어판과 같다.  같은 파서의 사건을 그린다 (as=events)
   가리지 않는다          그 카드가 그래프와 목록을 안 가린다
   헤더                 /ui/ 의 보안 헤더 다섯이 이미 걸려 있다 (ui.go:64-71)
```

**화면은 이 회차의 새 `.pen` 으로 진행자가 그린다** — S2 의 확장이다.

---

## 7. `internal/runctl` — 이 단계가 찾은 일곱째

**`requirements.md` 7.3 의 여섯에 없다.** 제어판이 Mediator 를 부르는 유일한
길이 `runctl.Client` 이고 (`panel.go` 의 패키지 주석 — 「Mediator 조회(runctl.Client)」),
FR-4 가 지난 것의 출처를 GET 라우트로 바꾸라고 했다.

```text
   는 것      클라이언트 메서드 하나.  GET .../steps/{seq}/log 를 부른다
   안 는 것    새 의존 0.  기존 do() 와 raw() 를 그대로 탄다
   게이트      enodectl.exe 의 net/http · crypto/tls 심볼 상한에 안 걸린다 —
             cmd/enodectl 이 internal/runctl 을 임포트하지 않는다 (실측)
```

이것을 안 적으면 유닛이 그 자리에서 `net/http` 를 제어판에 직접 들여온다.

---

## 8. 컴포넌트 일곱과 FR · NC 의 사상

| 컴포넌트 | FR | NC |
|---|---|---|
| `internal/transcript` | FR-3 | NC-2 · NC-4 의 재료 |
| `internal/enode` | FR-1 · FR-2 · FR-5 | — |
| `internal/record` | FR-5 | NC-4 의 값 |
| `internal/api` | FR-5 · FR-6 | NC-4 · NC-5 |
| `internal/panel` | FR-4 | NC-1 · NC-2 · NC-3 |
| `internal/api/ui` | FR-7 | NC-6 |
| `internal/runctl` | FR-4 | — |

**FR-8 은 이월이다** (`runctl mcp` 가 `main` 에 없다). 컴포넌트가 0 이다.
