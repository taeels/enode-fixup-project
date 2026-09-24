# 유닛 분해 계획 — Units Generation

입력은 `requirements.md` (FR-1 ~ FR-13 · 8절 ⑬) · `stories.md` (스토리 열아홉 · 완료 조건
열) · `application-design/` 다섯 (새 패키지 셋 · 기존 여덟의 새 책임) · `scene-gates.md`
(조각 0 ~ 12) · `plans/execution-plan.md` (1.4 의 제약 둘 · 4절의 직렬 병합 지점).

- **회차 브랜치**: `v4-run-finalize-bake` · HEAD `1b4e606`
- **작성 시각**: 2026-09-24T06:50:29Z
- **이 단계가 명시로 지는 것 둘**: 파일 행렬과 직렬 병합 지점 (`execution-plan.md` 3절) ·
  **범위를 자를 자리** (`requirements.md` 9절)

---

## 1. 이 단계가 딛는 제약 — 이미 정해진 것

**묻지 않는다.** 앞 단계가 닫았다.

```text
   담당        한 손 (taeels).  질문 2 = B.  Construction 문서 루트는 aidlc-docs/taeels/
   브랜치      유닛마다 unit/<유닛>.  v4-run-finalize-bake 에서 딴다.  PR 로 main
   경로        새 셋 (internal/lower · merge · scratch) 과 책임이 느는 여덟 ·
               packaging/macos/examples.  application-design/components.md 가 적는다
   되돌리기     합치기 규칙(internal/merge)은 자기 유닛이다 (execution-plan.md 1.4 제약 첫째)
   재개 시험    가짜 트리의 재개 시험이 실제 lower 에 합치기를 처음 돌리기 전에 초록이다
               (같은 자리 제약 둘째 · 조각 7 의 기계 부분)
   직렬 병합    후보 여덟.  실행 계획의 다섯 (contract.go · claim.go · runc_overlay_linux.go ·
               schema.sql · api.go) 에 설계가 셋을 더했다 (advertise.go · detect.go · status.go)
   NFR 값      N1 ~ N3 은 각각 한 유닛이 진다 (execution-plan.md 3절)
   보류        조각이 유닛 밖의 이유로 빨가면 audit.md 에 적고 보류로 넘어간다.
               보류는 통과가 아니고 병합 지점도 아니다 (scene-gates.md 4절 · CONVENTIONS.md 3.3)
```

## 1.1 규칙이 요구한 범주 여섯 — 무엇을 물었나

`units-generation.md` Step 3 이 여섯 범주를 전부 평가하라고 한다.

```text
   Story Grouping      물었다 (Q1).  묶는 단위가 패키지냐 팩의 세 갈래냐가 갈린다
   Dependencies        물었다 (Q2).  조각이 유닛 게이트가 아니다.  사람 조각 여덟 중 여섯이
                       SunnyVM 이라 이 답이 앞 회차보다 무겁다
   Business Domain     물었다 (Q3).  범위를 자르는 자리다.  FR-11 이 가치의 고정점 밖에 있다
   Technical           물었다 (Q4).  병렬로 도나.  웨이브를 가르는 기준이 하나 는다
   Team Alignment      안 물었다.  질문 2 = B 가 닫았다 (한 손 · taeels)
   Code Organization   해당 없음.  greenfield 전용이고 이 회차는 브라운필드다
```

## 1.2 설계가 답하는 것 — 묻지 않는다

근거가 앞 단계 문서나 코드에 있다. **반대하면 답에 적어 주세요.**

```text
   NFR 값 배정        N1 (삭제자가 한 번에 지우는 양) -> trash
                     N2 (checkpoint 상한과 inode 한도) -> checkpoint
                     N3 (업로드 예산과 요청의 관계) -> finalize
   FR-12 · FR-13     유닛이 없다.  코드가 0 이고 사람 확인과 정본 문서다.  조각 11 · 12 는
                     Build and Test 에서 사용자가 돌고, ADR-073 §11 · ADR-072 의 문서 고침은
                     진행자가 enode-design 에 올린다 (execution-plan.md 9절)
   상태 파일 · 제어판   drain 의 출처와 양을 들이는 유닛이 제 줄을 함께 쓴다 — 여유 부족과
                     trash 양은 trash, 굽기 출처는 lower-state, spool 양과 보존 수는 checkpoint.
                     따로 모아 마지막에 한 유닛으로 그리면 그 사이 main 의 제어판이
                     「drain 없음」을 말한다.  완료 조건 1 이 그 거짓 때문에 생겼다
                     (panel/view.go:97).  대가는 status.go 와 panel/view.go 가 직렬 병합이 된다
   완료 조건 4        Mediator 유닛이 진다.  진행 조회의 요구 줄(RequireView)이 phase 칸과 같은
                     파일이라 떼면 직렬 병합 지점이 하나 는다
   조각 7            기계 부분(가짜 트리 · 1 ~ 30번째 연산 뒤 강제 종료)은 merge-rules 의 코드
                     게이트다.  조각 전체(SunnyVM 에서 SIGKILL 뒤 재개)는 bake 가 진다
   완료 조건 8        조각 6 · 7 · 8 의 스크립트 자리라 그 조각을 지는 유닛(bake)의 Code
                     Generation 계획이 진다
   완료 조건과 조각    완료 조건은 그 유닛의 Code Generation 승인에서 재는 완료 기준이다.
                     병합을 막는 것은 조각이다 (Q2)
   유닛 이름          kebab.  이미 있는 unit/ 브랜치 열여섯과 안 겹친다 (advert · chunk-push ·
                     contract-vocab · … · runtime-environment-profile)
   번호와 약칭        유닛을 번호 약칭으로 안 부른다.  이름으로 부른다.  웨이브도 「웨이브 1」로
                     풀어 쓴다 (CLAUDE.md 의 glossary 규약 · stories.md 2절과 같은 이유)
```

---

## 2. 결정이 필요한 것 — `[Answer]:` 태그

**넷이다** (장면 4 가 둘을 더했다 — 2.1).  각 질문에 권장과 근거를 붙였다. `[Answer]:` 뒤에 기호를 적어 주세요.
「권장대로」라고 답하셔도 됩니다.

---

### Q1. 유닛을 어디서 가르나

기능 열셋 · 조각 열셋 · 패키지 열하나(새 셋 포함)가 서로 안 겹친다. 무엇을 유닛의 경계로
삼느냐가 갈린다. A 의 아홉은 이렇다.

| 유닛 | 자리 | FR | 지는 조각 |
|---|---|---|---|
| `contract-grammar` | `internal/contract` — effect · budget · build · merge · discover 와 400 검증 | 1 · 3 · 5 의 문법 | 없음 (조각 5 의 400 절반을 코드 게이트로 잰다) |
| `step-phase` | `internal/store` · `internal/api` · `internal/record` — exited 수락 · phase 칸 셋 · result 새 칸 · 진행 조회 · QUEUED 후보 셋 | 2 (Mediator) · 완료 조건 4 | 0 (라우트 18 -> 19) |
| `finalize` | `internal/enode` 명령 단계 — 수확 좁히기 · bounded discovery · 진단 · exited 전송 · 예산 · 업로드 client | 1 · 2 (노드) · 3 | 1 · 2 · 3 |
| `trash` | `internal/scratch` (trash · 삭제자) · runRoot 다섯 자리 · trash-helper · 여유 drain · arch 조건문 제거 · 상태 파일과 제어판 · 예시 넷 | 4 | 4 |
| `merge-rules` | `internal/merge` — 합치기 표 · 시작 전 확인 · 가짜 트리 재개 시험 | 7 (규칙) | 7 의 기계 부분 |
| `lower-state` | `internal/lower` · LowerGuard(후보 잠금 · 굽기 drain) · 광고 키 · env check 셋 | 8 (상태 · 잠금 · drain · env check) · 9 (광고) | 없음 |
| `bake` | build · merge 단계 · merge-helper · 기동의 정리와 재개 · metadata 쓰기 · 대기 로그 | 5 (노드) · 6 · 7 (연결) · 8 (나머지) · 9 (쓰기) | 5 · 6 · 7 · 8 |
| `checkpoint` | `internal/scratch` (Store) · capture capability · spool · receipt · `enode checkpoint` | 10 | 9 |
| `result-adapters` — 순연 | Q3 = A 로 뺐다 (2026-09-24T09:06:19Z) | 11 | 10 (해당 없음) |

A) 아홉 (위 표). Q3 에서 FR-11 을 자르면 여덟

B) 다섯 — 팩의 세 갈래에 합치기 규칙과 adapter 를 더한다. `step-result`(문법 · Mediator ·
명령 단계를 한 유닛) · `run-state`(trash 와 checkpoint 를 한 유닛) · `merge-rules` ·
`bake`(lower-state 와 굽기 단계를 한 유닛) · `result-adapters`

C) 열하나 — A 에서 둘을 더 가른다. `contract-grammar` 를 결과 문법(effect · budget ·
discover)과 굽기 문법(build · merge)으로, `lower-state` 에서 env check 를 떼어 따로

D) Other (please describe after [Answer]: tag below)

**권장 A.** 근거 넷이다.

```text
   되돌리기가 갈린다      셋 다 merge-rules 를 떼지만 lower-state 를 bake 에서 떼는 것은 A 뿐이다.
                        lower-state 는 모든 runc-overlay 노드의 준비도를 바꾸고(완료 조건 3 ②)
                        bake 는 lower 를 바꾸는 단계를 들인다.  B 는 되돌리기 어려운 쪽(bake)을
                        물릴 때 env check 까지 같이 문다
   사람 조각이 한 유닛에   A 는 사람 조각(2 · 4 · 6 · 8 · 9 · 10)이 각각 한 유닛에 떨어진다.
   하나씩 떨어진다        B 의 run-state 는 조각 4 와 9 를 함께 지고 둘 다 SunnyVM 이며 9 가 4 를
                        딛는다.  사람 게이트 둘을 줄지어 기다리는 동안, 사내 증상의 뿌리
                        하나(ADR-076 §4.1)를 고치는 trash 가 checkpoint 를 기다린다
   직렬 병합이 준다       문법을 한 유닛에 모으면 contract.go 가, Mediator 를 한 유닛에 모으면
                        api.go 와 schema.sql 이 한 손만 탄다.  실행 계획의 다섯 중 셋이 빠진다.
                        남는 것은 claim.go · runc_overlay_linux.go 와 설계가 더한 셋이다 —
                        internal/enode 를 관심사로 가른 값이다
   C 는 게이트를 안 늘린다  문법을 둘로 가르면 contract.go 가 다시 직렬이 된다.  env check 를 떼도
                        그것을 재는 조각 8 은 bake 가 진다.  PR 과 직렬 지점만 는다
```

A 의 대가 하나 — PR 이 아홉(또는 여덟)이다. **한 손이라 충돌은 0 이고** 직렬 지점은
진행자가 파일 행렬의 순서대로 하나씩 병합한다.

**[Answer]:** A — 「A.」 (2026-09-24T09:18:59Z).  Q3 = A 로 result-adapters 가 빠져 여덟이다

---

### Q2. 장면 조각을 유닛 병합에 어떻게 거나

`CONVENTIONS.md` 3.3 이 「그 유닛의 장면 게이트가 초록인 뒤에 병합한다」고 적는다.
**그런데 조각은 유닛 게이트가 아니다** — 조각 6 하나가 FR-5 · 6 · 7 · 8 · 9 를 함께 재고
그것이 A 에서 유닛 넷(contract-grammar · merge-rules · lower-state · bake)에 흩어진다.
그대로 읽으면 앞선 유닛이 전부 「유닛 밖의 이유로 빨감」이 되어 보류에 걸리고, 보류는
병합 지점이 아니므로 아무것도 안 움직인다. 앞 회차가 같은 자리에서 물었다
(`v3-run-transcript` 계획 Q2).

A) 조각이 재는 기능을 **마지막으로 완성하는 유닛**이 그 조각을 진다. 앞선 유닛은 코드
게이트(빌드 · 시험 · 패키지별 커버리지 80% · 경계 시험 · 크로스 빌드 셋)로 병합한다.
앞 회차의 답을 잇는다

B) 조각이 재는 기능을 가진 유닛 전부가 함께 진다. 마지막 하나가 초록일 때까지 그 유닛들이
다 브랜치에 머문다

C) 유닛마다 완료 조건을 새로 쓰고 조각은 Build and Test 에서만 돈다

D) Other (please describe after [Answer]: tag below)

**권장 A.** 근거 셋이다.

```text
   팩의 제목이 A 다        「진행의 단위는 유닛 완료가 아니라 장면 완주다」.  조각은 장면을 잰다.
                          그것을 세우는 마지막 손이 진다
   B 는 노트북을 기다린다   사람 조각 여덟 중 여섯이 SunnyVM 이다.  B 면 contract-grammar ·
                          merge-rules · lower-state 가 조각 6 · 8 의 사람 초록을 기다린다.
                          그동안 그것을 코드로 딛는 bake 는 main 에 없는 코드 위에 브랜치를
                          쌓아야 한다
   C 는 두 벌이 된다       조각과 따로 완료 어휘가 생긴다.  CONVENTIONS 1.4 의 「도구는 두 벌로
                          두지 않는다」와 같은 결이다
```

A 가 남기는 위험 하나 — 앞선 유닛이 `main` 에 들어간 뒤 조각이 빨개질 수 있다. **그래서 유닛
정의마다 「지는 조각」과 「딛는 조각」을 함께 적는다.** 빨개졌을 때 누구에게 돌아가야
하는지가 그 표다.

**A 면 팩의 착수 규칙을 이렇게 읽는다.** `scene-gates.md` 머리의 「조각 게이트가 초록이 아니면
다음 유닛을 착수하지 않는다」— 유닛은 **자기가 지는 조각의 「먼저 서는 조각」이 초록이거나
보류일 때** 착수한다. 자기가 지는 조각은 빼고 센다 (bake 는 조각 5 를 스스로 진다).

**[Answer]:** A — 「A」 (2026-09-24T09:41:10Z).  앞 회차의 답을 잇는다

---

### Q3. FR-11 의 adapter 둘을 이 회차에 두나

`requirements.md` 9절이 범위를 자를 자리를 이 단계의 승인으로 남겼고, Application Design
이 FR-11 을 그 후보로 적었다 (`application-design-plan.md` 2절 ⑪). bounded discovery 는
이미 FR-1 로 옮겼으므로 여기 남은 것은 adapter 둘이다.

A) **자른다.** 두 adapter 를 순연으로 돌린다. 조각 10 은 해당 없음이다(보류가 아니다).
완료 조건 3 ⑤ 와 US-10 은 FR-11 과 함께 순연 행으로 간다. 계약에 `produce` 칸을 안 더한다

B) 둔다 — 마지막 유닛(`result-adapters`)으로. 아무 유닛도 그것을 딛지 않는다

C) Git changeset adapter 만 둔다 — agent 단계의 걷기를 끈다

D) Yocto producer adapter 만 둔다 — BitBake 의 deploy output 을 named output 으로 올린다

E) Other (please describe after [Answer]: tag below)

**권장 C** (2026-09-24T07:36:56Z 고침 — 처음 판은 A 를 권했다). 근거 셋이다.

```text
   agent 단계도 같은 창을 탄다   agent 단계가 끝나면 Harvest 가 워크스페이스 전체를 걷는다.
                               runc-overlay 는 merged view 를 걷는다 — lower 와 upper 를 합친
                               트리다 (runc_overlay_linux.go:979 -> runtime.go:145 changedSince.
                               .git 과 .repo 만 뺀다).  300만 파일 Yocto 워크스페이스에서 agent 가
                               돌면 사내 증상과 같은 걷기가 임대 창에 남는다.  그것을 끄는 길은
                               Git changeset adapter 하나다 (결정 1-14)
   producer 는 편의다           BitBake 산출물은 오늘처럼 $OUT 과 collect 로 나온다.  없어도
                               아무것도 느려지지 않는다.  그런데 등록표 · 판 · 광고 키 · 계약
                               칸이 붙는 가장 넓은 표면이다
   둘 다 잎이다                 조각 10 을 「먼저 서는 조각」으로 적은 조각이 0 이다.  어느 쪽을
                               골라도 다른 유닛의 순서가 안 바뀐다
```

**처음 판이 틀린 자리.** A 의 근거로 「사내 증상은 명령 단계다 — agent 단계는 나빠지지 않고
나아지지 않는다」를 들었다. 앞 절반은 맞고 뒤 절반이 걷기의 비용을 빠뜨렸다. agent 단계는
나빠지지 않지만, **큰 워크스페이스에서는 명령 단계가 고쳐진 뒤에도 같은 기다림을 혼자
남긴다.** 사용자의 되물음(「왜?」)에 답하며 코드를 다시 대고 찾았다.

**두 번째 정정 (2026-09-24T07:52:00Z) — 위 첫 줄의 끝 문장이 틀렸다.** 「그것을 끄는 길은 Git
changeset adapter 하나다」가 아니다. 사용자가 짚었다 — 「중간 단계는 TTL 을 두고 버리는
걸로 정했잖아」. 2026-09-23 에 단계 결과를 receipt · checkpoint · published result 셋으로
갈랐고(원장 `EN-43c3e7d8` · ADR-076), 가운데 checkpoint 는 upper 를 TTL 로 두었다 버린다.

```text
   걷기의 산물은 셋 중 어디에도 안 든다   workspace.changed 는 published result 가 아니고
                                       (ADR-075 §8 — discovery 는 진단이다) checkpoint 도
                                       아니다 (checkpoint 는 upper 를 rename 으로 둔 것이라
                                       걷지 않는다)
   agent 단계에 걷기를 남긴 까닭이 거짓이다   requirements.md FR-1 과 확인된 사실 6 의 끝이
                                       「훅이 workspace.changed 를 읽는다」를 근거로 들었다.
                                       그 파일을 읽는 코드는 0 이다 (쓰는 곳만 claim.go:1005).
                                       훅은 산출물이 빠졌을 때 스스로 걷는다 (hook.go:281 ~ :283)
                                       — Application Design Q5 가 이미 찾았다
   agent 단계의 결과는 diff 다            agent 가 고친 것은 RecordDiff(git add -N . 과
                                       git diff --binary HEAD)로 나간다.  이것이 adapter 가
                                       갈아 끼울 자리이고, 계약이 steps[].workspace 를 적을
                                       때만 켜진다
```

**그래서 사내 증상과 같은 걷기(Discover)는 adapter 없이 FR-1 에서 agent 단계까지 끌 수
있다** (Q7). adapter 가 남기는 값은 증상이 아니라 장면 4 다 — agent 가 고친 것을 정확한
base 와 함께 build 로 넘기고, RecordDiff 의 전체 트리 비용(합성 300만 파일에서 12.79 s ->
1m31.723s · 원장 `EN-4694ff31`)을 유계로 만든다. 사내 계약에는 steps[].workspace 가 없어
그 비용이 사내 사건에 관여하지 않았다 (`EN-5390ec70`).

C 의 대가 — Q1 표의 `result-adapters` 가 `git-changeset` 이 되고 유닛 수는 아홉 그대로다.
조각 10 은 Git 절반(정확한 base 에 적용되는 changeset · compiler 부산물 없음)만 잰다. Yocto
절반(deploy output 만 commit set 에)은 producer adapter 와 함께 순연 행으로 간다 — 되살릴
조건은 산출물 경로를 빌드 시스템에 물어야 하는 계약이 생길 때다. `produce` 칸과
`producer.<이름>` 광고 키를 안 짓는다. US-10 과 완료 조건 3 ⑤ 는 범위에 남는다.
**A 를 고르면** agent 단계의 걷기가 이 회차 뒤에도 남는다는 것을 순연 행에 적는다.

**사용자 지시 (2026-09-24T07:48:31Z)** — 「장면을 만들어 그럼. 어댑터 인터페이스는 있어?」.
`requirements.md` 6.1 에 장면 4(agent 가 고친 것을 build 가 받는다)를 더했다. 그래서 위의
근거 둘이 바뀐다.

```text
   가치의 고정점     장면 4 가 adapter 둘을 다 지난다.  「어디도 안 지난다」가 더는 안 선다
   잎이 아니다       조각 12 가 조각 10 을 딛는다.  FR-11 을 자르면 FR-13 의 조각도 장면 4 를
                    끝까지 못 잰다
   빈자리 셋         받는 쪽 · base 의 뜻 · adapter 인터페이스 (6.1).  셋째는 Application
                    Design 의 겉면이라 이 답을 닫기 전에 설계에 더할지가 걸린다
```

**[Answer]:** A — 둘 다 순연 (2026-09-24T09:06:19Z).  사용자 「git changeset 은 사내에서 쓸일이 없을듯. … 이런 상황에서 어댑터를 순연해도 사내 동작에 아무 문제 없다면 순연으로 결정한다.」 사내 조건에 대 본 결과는 아래 「Q3 의 답을 사내에 대 봤다」. 문제가 없는 것은 Q7 = A 일 때다

### Q3 의 답을 사내에 대 봤다 (2026-09-24T09:06:19Z)

사용자가 준 사내 조건을 제품이 알 필요가 있는 모양으로만 적는다 — Yocto 기반 리눅스 빌드 ·
repo manifest 하나로 묶은 트리 · 구성이 여럿이고 그중 일부에 변형이 있다 · 구성 가운데
Yocto 가 아닌 것이 있다 · 빌드는 사내 스크립트가 시작한다. 스크립트와 구성의 이름은 제품이
몰라야 하는 사내 정보라 적지 않는다 (2026-09-24T11:57:06Z 사용자 지적).

```text
   Git changeset 없이      사내 계약에는 steps[].workspace 가 없다 (원장 EN-5390ec70).  그래서
                          agent 단계의 RecordDiff 는 오늘도 꺼져 있고 diff 가 안 나간다.  사내가
                          Git changeset 을 쓸 일이 없다는 사용자의 말과 맞는다
   agent 단계의 걷기        남는 문제는 이것 하나다.  Discover 는 계약과 무관하게 켜져 있다
                          (claim.go:879).  Q7 = A 면 FR-1 이 끄고, Q7 = B 면 사내 증상과 같은
                          걷기가 agent 단계에 남는다.  그래서 「문제 없음」은 Q7 = A 에 달렸다
   producer 없이           ADR-075 §7 이 output 을 확정하는 순서가 $OUT -> producer ->
                          저장소가 가진 export 스크립트 -> collect 다.  사내 빌드는 커스텀
                          스크립트가 시작하므로 첫째나 셋째 자리다.  Yocto 가 아닌 구성은
                          Yocto producer 가 덮지도 못한다
   굽기                    굽기의 결과는 파일 목록이 아니라 lower 자체다 (prepare).  builds[] 의
                          항목이 스크립트 호출이고 producer 를 안 지난다.  구성과 변형은
                          builds[] 의 이름으로 편다.  광고 키 repo.built.<이름>
   조각 12 (FR-13)         오늘의 diff 경로(steps[].workspace 와 in.from)로 돈다.  adapter 를
                          안 딛는다.  먼저 서는 조각은 6 하나로 돌아간다
```

**FR-1 과 따로 확인할 것 하나.** 사내 계약이 빌드 산출물을 `workspace.changed` 목록으로
찾고 있었다면, FR-1 이 명령 단계의 걷기를 끄는 날 그 목록이 없어진다. adapter 와 무관하게
FR-1 에서 생기는 변화다 (완료 조건 3 ④). 대체는 `$OUT` · `collect` 와 명시로 켜는 bounded
discovery 다.

**Q3 = A 가 바꾸는 것.**

```text
   유닛            result-adapters 가 빠져 여덟이다
   장면 4          requirements.md 6.1 을 순연 행으로 옮긴다.  되살릴 때 그 장면으로 잰다
   조각 10         해당 없음 (보류가 아니다).  조각 12 는 조각 6 만 딛는다
   계약 · 광고     produce 칸과 producer.<이름> 키를 안 짓는다
   US-10 · 완료 조건 3 ⑤   Q7 = A 면 「agent 단계의 걷기가 이 회차에 꺼진다」로 고쳐 finalize 가
                          진다.  Q7 = B 면 FR-11 과 함께 순연이다
   순연 행          4-16 — 결과 adapter 둘과 장면 4.  되살릴 조건은 agent 가 고친 것을 build 로
                  넘기는 계약이 생길 때, 또는 산출물 경로를 빌드 시스템에 물어야 하는 계약이
                  생길 때.  설계 빚 셋(받는 쪽 · base 의 뜻 · 인터페이스)을 함께 적는다
```

---

### Q4. Construction 을 병렬로 도나

앞 회차는 한 손 직렬로 계획했다가 Construction 착수 때 사용자가 병렬을 지시해 웨이브를
다시 짰다 (`v3-run-transcript` 의존 문서 2.1). 그때 웨이브를 가른 기준은 둘이었다 — 코드
의존과 **같은 파일.**

A) **웨이브로 병렬.** 처음부터 두 기준으로 웨이브를 가르고 한 웨이브 안의 유닛을 각자의
worktree 에서 함께 돈다. 직렬 병합 지점은 진행자가 파일 행렬의 순서대로 하나씩 병합한다

B) 한 줄 직렬. 팩의 착수 순서(`decisions.md` 5절 — 기능 1 ~ 3 과 4, 그다음 5 ~ 9, 그다음
10 · 11)대로 하나씩

C) Other (please describe after [Answer]: tag below)

**권장 A.** 근거 셋이다.

```text
   다시 짜지 않는다      앞 회차에서 한 번 걷힌 전제다.  처음부터 두 기준으로 내면 착수 때 표를
                        다시 대지 않는다.  B 를 고르면 같은 표에서 한 줄 순서만 쓴다
   병렬 기회가 실재한다   contract-grammar · merge-rules · trash 가 코드 의존 0 이다.  겹치는
                        파일은 경계 시험의 표 하나이고 줄이 서로 다르다
   제약 둘째가 일찍 선다   merge-rules 가 첫 웨이브에 들면 재개 시험이 bake 착수보다 한참 먼저
                        초록이 된다 (1절의 재개 시험 줄)
```

A 의 대가 둘 — **단계 승인이 한꺼번에 몰린다.** 유닛마다 Functional Design 부터 Code
Generation 까지 승인이 있고, 한 웨이브가 셋이면 그 요청도 셋이 나란히 온다. 사람 조각도
웨이브 끝에 몰린다.

**[Answer]:** B — 「병렬이어야 할 이유는 없다. 혼자 작업하기 때문에.」 (2026-09-24T06:54:38Z)

---

## 2.1 장면 4 가 낸 질문 둘 (2026-09-24T07:48:31Z)

`requirements.md` 6.1 의 빈자리 셋 중 둘이 결정이다. 셋째(base 의 뜻 — repo 로 묶은 트리에서
어느 HEAD 인가)는 규칙이라 그 유닛의 Functional Design 이 닫는다.

---

### Q5. changeset 을 받는 쪽은 누구인가

A) **노드가 적용한다.** 계약의 `in.diff` 를 런타임이 읽는다 (정본 `run-contract.md:125` 의
`@work.patch_rev` 자리 · 오늘 `contract.go:905` 가 자리만 둔다). 명령 전에 base 를 대조하고
다르면 단계를 시작하지 않는다

B) 계약의 명령이 적용한다. `in.from` 으로 받아 `git apply` 한다. 노드는 descriptor 만 싣고
base 대조는 계약 저자의 몫이다

C) Other (please describe after [Answer]: tag below)

**권장 A.** 근거 둘이다.

```text
   장면 4 의 6 이 A 로만 선다    B 는 ADR-075 §6 이 base identity 로 막으려는 것 — 같은 path 의
                              다른 revision 에 조용히 붙는 것 — 을 저자의 규율에 맡긴다
   자리가 이미 있다              정본과 계약 문법이 in.diff 를 적어 두었다.  새 어휘가 아니라
                              읽지 않던 칸을 읽는 것이다
```

A 의 대가 — 런타임 동작이 하나 는다. adapter 유닛이 만드는 쪽과 받는 쪽을 함께 진다.

**[Answer]:** 해당 없음 — Q3 = A 로 받는 쪽을 짓지 않는다. 장면 4 와 함께 순연 행으로 간다

---

### Q6. adapter 인터페이스를 어디서 정하나

**adapter 는 이름이 아니라 자리다** (2026-09-24T08:22:21Z 사용자 되물음에 더함). ADR-075 §5 의
표가 effect 마다 정상 commit set 을 적는다 — edit 는 source changeset, build/test 는 producer
가 확정한 output, prepare 는 manifest. adapter 는 그 칸을 채우는 자리이고 Git 과 Yocto 는 첫
구현이다 (§6 이 Git 밖의 filesystem changeset 을, §7 이 Bazel · CMake · Meson 을 뒤에 둔다).
그래서 A 의 인터페이스는 하나가 아니라 둘이다 — 약속이 갈린다.

```text
   source changeset   정확한 base 를 싣는다.  받는 쪽이 base 를 대조해 적용한다 (Q5)
   producer           빌드 시스템에 묻는 typed query 다.  결과 경로가 워크스페이스 밖으로
                      안 나간다.  namespace 가 살아 있는 동안 돈다
   둘 다              Finalize 안에서 세션이 닫히기 전에 돈다.  워크스페이스 전체를 안 걷는다
```

A) **Application Design 을 고친다.** `component-methods.md` 에 인터페이스 둘(source
changeset · producer)과 등록표의 모양을 더하고, 받는 쪽(Q5)의 겉면을 적는다. 고친 것을
이 단계의 승인에 함께 올린다

B) adapter 유닛의 Functional Design 이 정한다

C) Other (please describe after [Answer]: tag below)

**권장 A.** 근거 둘이다.

```text
   겉면은 설계의 몫이다     메서드 시그니처와 입출력 타입은 Application Design 이 진다.
                          Functional Design 은 그 위의 규칙이다
   finalize 가 먼저 짓는다   finalize 유닛이 FinalizeSpec 을 먼저 세운다.  지금 설계대로
                          칸(Changeset bool · Produce)으로 박히면 adapter 유닛이 claim.go 와
                          세션을 다시 고친다.  인터페이스가 먼저 서면 finalize 가 끼울 자리를
                          두고 adapter 유닛은 구현만 더한다
```

**[Answer]:** 해당 없음 — Q3 = A. 인터페이스 둘과 등록표는 순연 행의 설계 빚으로 적는다. finalize 유닛은 설계의 칸 둘(Changeset · Produce)을 짓지 않는다

---

### Q7. agent 단계의 전수 걷기를 FR-1 에서 함께 끄나 (2026-09-24T07:52:00Z)

`requirements.md` FR-1 과 결정 1-14 는 agent 단계의 `RecordDiff` 와 `Discover` 를 Git
changeset adapter 까지 그대로 두었다. 근거는 「훅이 `workspace.changed` 를 읽는다」였고
그것이 거짓이다 (Q3 의 두 번째 정정).

A) **Discover 는 지금 끄고 RecordDiff 는 adapter 까지 둔다.** FR-1 의 범위를 agent 단계의
Discover 까지 넓힌다. 결정 1-14 를 「agent 단계의 RecordDiff 는 adapter 까지 그대로」로 좁힌다.
bounded discovery(FR-1)가 대체다

B) 둘 다 adapter 까지 그대로 (오늘의 FR-1)

C) Other (please describe after [Answer]: tag below)

**권장 A.** 근거 둘이다.

```text
   사내 증상의 기제가 그 걷기다   사내의 안 풀리는 임대는 changedSince 의 WalkDir 하나로 났다
                                (EN-5390ec70).  agent 단계에도 같은 줄이 켜져 있다 (claim.go:879)
   셋으로 가른 원칙과 맞는다      걷기의 산물은 결과도 checkpoint 도 아니다.  남길 까닭이 없다
```

A 의 대가 — `requirements.md` FR-1 · 확인된 사실 6 · 10절의 1-14 를 고친다(승인된 문서).
finalize 유닛이 agent 단계의 Harvest 도 만진다.

**[Answer]:** A — 「그래 a」 (2026-09-24T09:50:00Z).  requirements.md FR-1 · 10절 1-14 와 stories.md US-10 · 완료 조건 3 ⑤ 를 고쳤다

---

## 2.2 되물음에서 찾은 것 둘 (2026-09-24T11:52:52Z)

사용자가 굽기 계약 예시를 두고 「이걸 계약 제출자가 알 수 있는 방법은?」이라고 물었다.
답을 찾다가 둘이 나왔다. 질문이 아니라 유닛 정의에 넣을 일이다.

**하나 — 계약 작성 도구가 새 문법을 절반만 저절로 보인다.** 도구는 ADR-066(계약을 쓰는 쪽을
돕는 도구는 Mediator 없이 돈다)의 것이다.

```text
   저절로 보인다      runctl schema steps   Go 구조체에서 칸 이름과 형을 뽑는다 (뜻은 안 보인다)
                     runctl lint           Validate 의 새 400 규칙이 그대로 걸린다
                     runctl capabilities   굽기 뒤 광고 키(workspace.writes · ir · repo.built.<이름>)
                     runctl dry-run        앉을 노드가 있는지
   손봐야 보인다      runctl example        예시가 internal/contract/examples 에 파일로 있다.
                                          굽기 예시가 없다
                     runctl lint 의 경고    판정 조건 제안(suggestCondition, cmd/runctl/shape.go)이
                                          명령 · agent 단계만 안다
                     contract.Grammar      계획을 짓는 agent 에게 주입하는 문법 문장이다.
                                          effect 와 예산이 없다
```

그래서 contract-grammar 유닛이 굽기 예시와 두 문장을 함께 맡는다. **예시는 사내 명령을
담지 않는다** — 제품은 sync 와 builds[].command 를 해석하지 않는 문자열로만 알고, 그 내용은
계약을 쓰는 쪽의 것이다 (FR-5 · 팩 보안 표의 「계약의 명령」 줄). 예시는 공개 도구로 쓴다 —
SunnyVM 시험이 쓴 poky 가 후보다. application-design 이
「안 만지는 것」에 둔 cmd/runctl 중 shape.go 한 파일이 그 유닛의 파일로 들어온다.

**둘 — 굽기 Run 의 성공을 무엇으로 판정하는지가 없다.** I3(Run 의 성공은 계약에 선언된
기계적 조건으로만 정한다) 아래에서 계약이 쓸 수 있는 조건은 produced · exit_code · changed ·
fleet_has 다. build · merge 는 명령 단계가 아니라 exit_code 를 걸 수 없다(contract.go:1612).
success_when 이 비면 모든 단계가 실패해도 Run 이 성공한다(runctl lint 의 경고). 팩 · ADR-077 ·
이번 Inception 산출물 어디에도 이 줄이 없다. 예시 시험(TestExamples_EveryStepHasASuccessCondition)
이 굽기 예시를 더하는 순간 이것을 묻는다.

```text
   후보   ①  build · merge 단계에 「단계가 성공으로 끝났다」는 조건을 연다
          ②  fleet_has 로 repo.built.<이름> · ir 광고를 본다 — 광고 주기만큼 늦다
          ③  build manifest 를 $OUT 산출물로 내고 produced 로 본다
   자리   계약 문법의 겉면이라 contract-grammar 유닛의 Functional Design 이 닫는다.
          정본 되돌림(ADR-077 · run-contract.md)에 한 줄을 더한다
```

## 2.3 결정 — 굽기 계약이 구울 IR 을 적는다 (2026-09-24T12:09:37Z)

사용자 「ir tag는 굽기 작업 제출자가 쓰게 하자」 · 갈래 질문에 「"나" 로 하고」. Application
Design 질문 7(「ir_tag · 기본 사내 형식」)을 고친다.

```text
   계약        build 단계가 구울 IR 태그를 정확한 값으로 적는다.  필수다.  제품에 기본값과
              형식 규칙이 없다 — 사내 형식은 제품이 몰라야 하는 사내 규약이다
   노드        그 값을 sync 와 builds 명령에 환경 변수로 넘긴다.  명령은 해석하지 않는다
   확인        sync 뒤 .repo/manifests 의 HEAD 에 그 태그가 정확히 붙었는지 본다.
              다르면 build 단계를 실패로 끝내고 합치지 않는다
   없어지는 것  태그가 여럿일 때 가려내는 규칙 · 「태그 없음 → ir null」 갈래
   근거        사실이 계약 한 곳에서 온다 — ADR-077 §5 가 계약 칸을 반대한 근거
              (같은 사실이 sync 명령과 칸 두 곳에서 온다)가 환경 변수로 풀린다.
              잘못된 리비전이 공용 lower 에 합쳐지는 것을 합치기 전에 막는다
```

**고칠 자리** — 빌드 구성 표기 논의(사용자가 이어서 요청)가 같은 자리를 건드릴 수 있어,
그 논의가 닫힌 뒤 한 번에 고친다. requirements.md FR-9 · 6절 조각 6 줄 · 10절에 결정 행 ·
11절 정본 되돌림(ADR-077 §5 · §11) · stories.md US-19 · 완료 조건 10 · application-design
다섯의 ⑩ 줄 · ir_tag · IR 유도 · ir_reason. 칸 이름과 환경 변수 이름 · 어긋났을 때의 원인 코드는
contract-grammar 와 bake 의 Functional Design 몫이다.

## 3. 산출물 계획 (체크박스)

규칙이 요구하는 셋에 **실행 계획이 거는 하나**(파일 행렬)를 더해 넷이다. 자리는
`aidlc-docs/v4-run-finalize-bake/inception/application-design/`. **2절의 답이 들어온 뒤
생성한다.**

- [x] `unit-of-work.md` — 유닛 정의와 책임
  - [x] 유닛마다 — 무엇을 하나 · 어느 경로를 만지나 · 무엇을 안 하나
  - [x] 유닛마다 **지는 조각과 딛는 조각** (Q2 의 답이 이 칸을 만든다)
  - [x] 유닛마다 지는 완료 조건과 스토리
  - [x] N1 · N2 · N3 을 지는 유닛을 하나씩 박는다 (1.2)
  - [x] 유닛마다 **Functional Design · NFR Requirements · NFR Design 을 도나 스킵하나** —
        규칙을 만드는 유닛만 Functional Design, 성능 · 보안 표면을 만드는 유닛만 NFR
        (`execution-plan.md` 3절의 적용 범위)
  - [x] 범위에서 뺀 것 — Q3 의 답 · FR-12 · FR-13 과 그 자리
- [x] `unit-of-work-dependency.md` — 의존과 순서
  - [x] 의존 행렬 — 코드 의존과 조각 의존을 갈라 적는다
  - [x] 한 줄 착수 순서 (Q4 = B — 웨이브 표는 안 낸다)
  - [x] Q2 의 착수 규칙으로 유닛마다 착수 조건 (먼저 서야 할 조각)
  - [x] **실행 계획 1.4 의 제약 둘이 지켜졌는지 센다** — merge-rules 가 따로인가 · 재개 시험이
        bake 의 조각 6 보다 먼저 초록인가
- [x] `unit-of-work-file-matrix.md` — 파일 행렬 (실행 계획이 필수로 걸었다)
  - [x] 파일마다 어느 유닛이 만지나 (코드와 시험)
  - [x] **유닛 둘 이상이 만지는 파일**을 표시하고 병합 순서를 적는다 (`CONVENTIONS.md` 3.1).
        후보 여덟에서 몇이 남고 무엇이 새로 나왔나 (경계 시험 · 제어판 · cmd/enode 가 후보다)
  - [x] `internal/match` · `cmd/mediator` 를 만지는 유닛이 0 임을 센다 (품질 게이트 3 · 4)
- [x] `unit-of-work-story-map.md` — 사상표
  - [x] 스토리 열아홉 -> 유닛.  **배정 안 된 것 0 을 센다**
  - [x] 완료 조건 열(3 의 다섯 줄 포함) -> 유닛.  **배정 안 된 것 0 을 센다**
  - [x] FR-1 ~ FR-13 -> 유닛.  유닛이 없는 FR 은 이유와 함께 센다
  - [x] 조각 0 ~ 12 -> 지는 유닛.  **지는 유닛이 없는 조각은 이유와 함께 센다**
- [x] 검증 — 경계와 의존
  - [x] 임포트 금지 넷 · 봉인 셋(`component-dependency.md` 2절)을 어느 유닛도 안 깬다
  - [x] 유닛마다 혼자 빌드되고 혼자 시험이 돈다 — 코드 의존이 행렬에 다 있다
  - [x] `internal/enode` 에 문장을 더하는 유닛마다 커버리지 하한을 어떻게 지키는지 한 줄씩
  - [x] Linux 전용 코드를 만드는 유닛마다 `_other.go` 짝
  - [x] 표기 — `emphasis-check.py` · U+2605 0 · 새 약칭 0

---

## 4. 답이 들어온 뒤의 순서

```text
   1   답 넷을 읽고 모순 · 모호를 본다 (Step 7).  있으면 이 파일에 되묻는 질문을 더한다
   2   승인을 받는다 (Step 9)
   3   3절의 체크박스를 순서대로 돌리고 그 자리에서 [x] 로 바꾼다
   4   산출물 넷을 낸다.  배정 안 된 스토리 · 완료 조건 · 조각이 0 임을 마지막에 센다
   5   확장 준수 요약을 붙인다 (셋 다 꺼짐)
   6   승인을 받고 커밋한다.  Inception 이 거기서 닫힌다 — 회차 브랜치를 PR 로 main 에 올리고,
       다음은 Construction 이며 문서 루트가 aidlc-docs/taeels/ 로 바뀐다
```

---

## 5. 답 (2026-09-24T09:50:00Z)

채팅에서 하나씩 논의했다. 질문이 넷에서 일곱으로 늘었다 — 장면 4 가 둘(Q5 · Q6)을,
Harvest 를 셋으로 가른 결정(원장 `EN-43c3e7d8`)이 하나(Q7)를 낳았다.

```text
   Q1 = A   유닛 여덟 (Q3 = A 로 result-adapters 가 빠졌다)
   Q2 = A   조각은 그 기능을 마지막으로 완성하는 유닛이 진다.  앞선 유닛은 코드 게이트로 병합
   Q3 = A   결과 adapter 둘과 장면 4 를 순연 — 사내 조건에 대 보고 정했다.  조건은 Q7 = A
   Q4 = B   한 줄 직렬 — 「혼자 작업하기 때문에」
   Q5       해당 없음 (Q3 = A)
   Q6       해당 없음 (Q3 = A)
   Q7 = A   agent 단계의 Discover 를 FR-1 에서 끈다.  RecordDiff 는 adapter 까지 그대로
```

### 모순 · 모호 분석 (Step 7)

**추가 질문 0.** 짝을 대 봤다.

```text
   Q3 x Q7   Q3 의 「사내 문제 없음」은 Q7 = A 를 조건으로 섰다.  Q7 = A 로 그 조건이 찼다
   Q7 x 조각 12   Q7 이 RecordDiff 를 남겼으므로 조각 12(FR-13)가 오늘의 diff 경로
             (steps[].workspace 와 in.from)로 돈다.  둘 다 껐으면 조각 12 가 설 길이 없었다
   Q1 x Q7   걷기를 끄는 자리가 claim.go 의 agent 경로(:876 ~ :882)다.  finalize 가 이미
             그 파일을 만지므로 유닛이 안 는다.  완료 조건 3 ⑤ 와 US-10 이 finalize 로 온다
   Q1 x Q2   여덟 중 사람 조각을 지는 유닛은 넷(finalize · trash · bake · checkpoint)이다.
             나머지 넷은 코드 게이트로 병합한다.  조각 11 · 12 는 유닛 없이 Build and Test
   Q2 x Q4   직렬이라 착수 규칙(자기가 지는 조각이 딛는 앞 조각이 초록이거나 보류)이 그대로
             순서의 하한이 된다.  웨이브 표는 안 낸다
   Q3 x Q1   유닛이 여덟이 되며 사람 조각이 하나(10) 준다.  조각 12 가 조각 6 만 딛는다
```

### 답이 고친 앞 단계 문서

```text
   requirements.md   FR-1 의 agent 줄 (Q7) · FR-11 머리에 순연 (Q3) · 6.1 장면 4 를 더하고
                     순연으로 표시 · 10절 1-14 를 좁히고 4-16 을 더했다
   stories.md        US-10 · 완료 조건 3 ⑤ (Q7) · 4.3 의 FR-11 줄
```

### 답 없이 넘어간 것 하나

사내가 단계에서 무엇을 가져가야 하나 (09:11:56Z 되물음). 이미지를 가져가야 하면 `$OUT` 의
blob 상한(기본 10 MiB)에 걸린다. adapter 와 무관하고 설계를 안 바꾸는 확인이라 이 단계를
막지 않는다. 큰 artifact 는 ADR-018 의 재개 조건 자리다.
