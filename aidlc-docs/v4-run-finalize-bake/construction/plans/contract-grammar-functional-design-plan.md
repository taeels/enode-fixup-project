# `contract-grammar` — Functional Design 계획

**유닛** `contract-grammar` (계약 문법) · **브랜치** `unit/contract-grammar` · **담당** taeels ·
**회차** `v4-run-finalize-bake` (굽기) · **순서** 여덟 중 첫째 · **선행** 없음 ·
**맡는 조각** 없음 — 코드 검사가 초록이면 병합한다

정본 입력은 `aidlc-docs/v4-run-finalize-bake/inception/` 의 유닛 정의(`unit-of-work.md` 1절) ·
요구(`requirements.md` FR-1 · FR-3 · FR-5 · FR-9) · 설계(`application-design/components.md` 3.2 ·
`component-methods.md` 5절) · 유닛 분해 계획 2.2 · 2.3 이다. 이 계획은 그 위에서 **계약 문법의
규칙과 거절 문구**만 짓는다. 코드는 다음 단계다.

---

## 0. 이 단계가 닫는 것과 안 닫는 것

```text
   닫는다     굽기 Run 의 성공 판정 — 계약이 무엇으로 「구웠다」를 확인하나 (계획 2.2)
             effect(작업 성격)와 budget(예산)을 어느 단계 종류가 받나 · 기본값 · 400 경계
             굽기 계약의 모양 — build · merge 두 단계가 받는 칸과 거절하는 칸
             구울 IR 칸의 이름 · 허용 문자 · 노드가 넘기는 환경 변수 이름 (계획 2.3)
             명시로 켜는 훑기 discover 의 문법
             계획(기계가 지은 계약 조각)이 새 칸을 쓸 수 있나 · Grammar 가 무엇을 가르치나
             runctl example bake 의 내용 · lint 의 조건 제안
             새 거절 문구의 목록 (영어 · 5.7 표기)

   안 닫는다   노드가 effect 를 보고 무엇을 켜고 끄나          finalize 유닛
             예산을 노드가 어떻게 지키나 · 원인 코드         finalize 유닛
             훑기의 상한 값과 결과를 싣는 자리              finalize 유닛
             sync 와 builds 를 노드가 어떻게 돌리나          bake 유닛
             IR 대조를 어디서 어떻게 하나 · 어긋난 원인 코드   bake 유닛
             build manifest 와 merge 결과의 내용            bake 유닛
             merge.wait 을 노드가 어떻게 세나               bake 유닛
             produce 칸 (결과 adapter)                    순연 (Units Generation Q3 = A)
```

---

## 1. 실측 — 코드가 지금 어떻게 생겼나 (2026-09-24 · `d421e81`)

### 1.1 단계의 종류는 판별 칸이 정한다

`Step.Kind()` (`internal/contract/contract.go:798`) 가 `agent` · `run` · `acquire` · `ask`
넷 중 **정확히 하나가 있는지**로 종류를 정한다. 둘 이상이거나 없으면 400 이다. 굽기의 두
단계도 같은 방식이면 된다 — `sync`+`builds` 가 있으면 build, `merge` 가 있으면 merge.
판별 칸이 여섯이 된다.

JSON 의 모르는 칸은 조용히 버린다 (`DisallowUnknownFields` 0). 새 칸을 모르는 옛 Mediator 는
`effect` 를 적은 계약을 거절하지 않고 그 칸만 버린다. 이 저장소는 Mediator 와 `runctl` 을
같은 소스에서 빌드하므로 문제가 되지 않고, 노드 쪽은 finalize 유닛이 다룬다.

### 1.2 판정 조건의 오늘

```text
   조건        누가 쓰는 값인가                      어느 종류에 걸 수 있나
   produced    단계가 $OUT 에 낸 산출물 이름          모든 종류
   exit_code   명령의 종료 코드                      run 만 — 「명령이 아니면」으로 막는다
   changed     계약이 지목한 경로가 바뀌었나          workspace 칸이 있는 단계
   fleet_has   함대에 그런 노드가 광고하나             단계에 안 건다 (이 Run 밖을 본다)
```

- `exit_code` 는 **종류를 열거하지 않고 「run 이 아니면」 400** 이다 (`contract.go:1612`
  무렵의 주석이 그렇게 적었다). build · merge 를 더해도 이 줄은 안 고쳐도 둘을 막는다
- 판정 함수 `Verify` (`internal/store/verdict.go:128`) 는 `StepResult` 의 `ExitCode` ·
  `Produced` · `Changed` · `Exhausted` 만 본다. **「단계가 성공으로 끝났다」는 값이 없다** —
  `Error` 가 비면 종료 코드가 무엇이든 DONE 이다 (`claim.go:733`)
- `success_when` 이 비면 모든 단계가 실패해도 Run 이 성공한다. 오늘은 `runctl lint` 의
  **경고**다 (`cmd/runctl/shape.go:83`). 400 이 아니다 — ADR-061 §2 (「틀린 것을 알려주는 것과
  못 쓰게 막는 것은 다르다 · 경고 먼저」)가 그 순서를 정했다

### 1.3 계약 작성 도구

```text
   runctl example       internal/contract/examples/*.json 일곱.  굽기 예시가 없다
   예시 시험             TestExamples_EveryStepHasASuccessCondition — 모든 단계에 조건이
                        하나 이상 걸려야 한다.  굽기 예시를 더하는 순간 판정 방식을 묻는다
   runctl lint 의 제안   suggestCondition (shape.go:112).  run 이면 exit_code 0,
                        아니면 produced ["<artifact name>"].  build · merge 에는 뜻 없는 제안이 된다
   contract.Grammar     계획을 짓는 agent 에게 주는 문법 문장 (grammar.go:29).  종류 넷을 가르친다.
                        grammar_test.go 가 문장마다 「그 규칙을 어긴 계약이 실제로 거절되나」를 확인한다
   기간                 ask.timeout.after 가 time.ParseDuration 을 쓴다 (contract.go:1296).
                        예산과 merge.wait 도 같은 형식이다 (설계가 정했다)
```

### 1.4 공개 예시와 IR 대조가 맞물리는 자리

계획 2.3 은 IR 대조를 「sync 뒤 `.repo/manifests` 의 HEAD 에 그 태그가 붙었나」로 적었다.
**SunnyVM 시험의 poky 는 repo 가 아니라 git 으로 받았다** (ADR-077 §12 — 「poky `77d1feb` →
`cbd62bb`」, 처음 굽기는 「clone 43초」). 그래서 poky 로 공개 예시를 쓰면 `.repo` 가 없다.

poky 저장소에는 공개 annotated 태그 `yocto-5.0.1` ~ `yocto-5.0.20` 이 있다
(`git ls-remote --tags https://git.yoctoproject.org/poky`, 2026-09-24 확인). 예시의 IR 값으로
쓸 수 있다. 물음 7 이 이 자리를 묻는다.

---

## 2. 물음 일곱

답을 `[Answer]:` 뒤에 적는다. 권장을 **A** 에 둔다. 기호 옆에 뜻을 적었다.

### Question 1 — 굽기 Run 의 성공 판정

I3 (Run 의 성공은 계약에 선언된 기계적 조건으로만 정한다) 아래에서 굽기 계약이 「구웠다 ·
합쳤다」를 무엇으로 확인하나. build · merge 는 명령 단계가 아니라 `exit_code` 를 못 건다 (1.2).

A) **produced 로 본다 (후보 ③).** build 단계는 모든 명령이 0 으로 끝나고 IR 대조가 맞았을
때만 산출물 `manifest` 를 내고, merge 단계는 합쳤을 때만 산출물 `merged` 를 낸다. 이름은
제품이 정하고 계약은 `out` 을 적지 않는다(적으면 400). 계약은
`{ "step": "build", "produced": ["manifest"] }` · `{ "step": "merge", "produced": ["merged"] }`
로 판정한다. `Verify` 와 조건 어휘는 바뀌지 않는다. ADR-077 §2 (굽기는 일반 Run 이고 두
단계다)의 「build 는 manifest 를, merge 는 합친 사실과 새 ir 을 낸다」와 같은 말이다. 실패한
시도의 기록은 산출물이 아니라 result 진단과 `state.json` 의 `last_attempt` 에 남는다 —
ADR-075 §5 (actor 와 effect 를 나누는 결정)의 「반쯤 생긴 것은 정상 produced 가 아니다」

B) **새 조건 `succeeded` 를 연다 (후보 ①).** `{ "step": "build", "succeeded": true }`. 노드가
보고하는 새 값을 `StepResult` 에 더하고 `Verify` 에 대조 줄을 더한다. 조건 어휘가 다섯이 되고
Grammar · lint · 정본 run-contract.md 가 함께 는다

C) **fleet_has 로 광고를 본다 (후보 ②).** `repo.built.<이름>` 과 `ir` 광고를 본다. 코드 변경이
가장 적다. 대가 — 광고 주기만큼 늦고, 같은 IR 을 광고하는 **다른 노드**가 있으면 이 Run 이
실패해도 참이 된다

D) Other (please describe after [Answer]: tag below)

빠뜨림(굽기 계약에 조건이 없음)은 어느 답이든 오늘처럼 lint 경고로 짚는다. 400 으로 올리지
않는다 (ADR-061 §2).

[Answer]: A.

### Question 2 — effect(작업 성격)와 budget(예산)을 어느 종류가 받나

FR-1 (단계 결과를 Finalize 로 닫는다)은 기본값을 「명령 단계 build · agent 단계 edit」로
정했고, 설계는 값 넷(`read` · `edit` · `build` · `prepare`)과 예산 경계(finalize 1분 아래 400 ·
upload 와 wait 0 이하 400 · 위쪽 상한 없음)를 정했다. 남은 것은 **종류마다 어느 값을 받나**다.

A) **엄격한 표.**

```text
   종류       effect                          budget
   run        build (기본) · edit · read       받는다
   agent      edit (기본) · read               받는다
   build      prepare — 반드시 적는다           받는다
   merge      적으면 400                       적으면 400.  기다림은 merge.wait 이 정한다
   ask        적으면 400                       적으면 400.  노드에 안 간다
   acquire    적으면 400                       적으면 400.  노드에 안 간다
```

`prepare` 를 run · agent 에 적으면 400, `build` 를 agent 에 적으면 400 이다. build 단계에서
`effect` 를 안 적으면 400 이다 — ADR-077 §2 가 「upper 를 버릴지 합칠지가 이 값 하나로
정해지므로 유도하지 않고 Record 에 보이는 값으로 둔다」고 적었다. `read` 는 이번 회차에서
`build` 와 같이 diff 와 전체 훑기를 안 하고, source 를 쓴 것을 위반으로 잡는 일은 범위 밖이다

B) **느슨한 표.** 노드에 가는 종류(run · agent · build)는 `prepare` 를 뺀 어느 값이든 받고,
뜻이 없는 조합은 조용히 기본값처럼 돈다. ask · acquire · merge 에 적은 것은 버린다

C) **`read` 를 이번에 열지 않는다.** A 의 표에서 `read` 를 빼고 값 셋으로 연다. 쓰는 곳이 없는
값을 먼저 열지 않는다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3 — 굽기 계약의 모양

FR-5 (굽기 계약)은 「굽기는 일반 Run 이고 두 단계다」라고 적었다. 굽기 계약에 다른 단계를
섞을 수 있나, 그리고 두 단계가 어떤 칸을 받나.

A) **두 단계만 · 받는 칸을 정해 둔다.**

```text
   build   받는다    id · uses · needs · effect(prepare) · sync · builds · ir · env · budget
           400       workspace — sync 가 이 단계의 준비다.  reset --hard 와 clean 을 겹쳐 하지 않는다
                     run · agent · in · out · collect · schema · loop · repeat · dispatch ·
                     validate_with · expands · release · see
   merge   받는다    id · uses · needs · merge(wait)
           400       그 밖의 칸 전부
   계약     build 하나와 merge 하나.  merge 의 needs 는 [build] 이고 uses 가 같다.
           다른 단계가 있으면 400 — 합친 뒤의 시험은 다음 Run 이 한다
```

`sync` 는 빈 문자열이면 400, `builds` 는 하나 이상이어야 한다. `sync` 와 `command` 는
문자열이고 노드가 셸로 돌린다 (ADR-077 §2 의 「명령을 그대로 적는다」 · 어떻게 돌리는지는 bake
유닛이 닫는다)

B) **다른 단계를 섞을 수 있다.** build 와 merge 는 A 의 칸 규칙을 따르되, merge 뒤에 시험
단계를 두는 것처럼 계약에 다른 단계가 있어도 된다. build 와 merge 사이에 같은 역할(uses)의
단계는 못 온다

C) **build 에 workspace 를 허용한다.** A 와 같되 build 가 `workspace` 칸을 받아 sync 전에
reset 과 clean 을 한다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4 — 구울 IR 칸의 이름 · 허용 문자 · 환경 변수

계획 2.3 이 「계약이 구울 IR 태그를 정확한 값으로 적는다 · 필수 · 제품에 형식 규칙 없음」을
정했다. 이름과 허용 문자는 이 단계의 몫이다. 값은 노드가 환경 변수로 sync 와 builds 에 넘기고,
광고 키 `ir` 의 값이 되며, git 태그 이름으로 쓰인다.

A) **이름 `ir` · 환경 변수 `ENODE_IR`.** 광고 키 `ir` 과 metadata 의 `source.ir` 와 같은 낱말이다.
허용 문자는 영문자 · 숫자 · `.` · `_` · `-` · `/` 이고 1 ~ 128 자다. `-` · `/` · `.` 로 시작하지
않고, `/` · `.lock` 으로 끝나지 않으며, `..` 과 `//` 를 담지 않는다 — git 태그 이름으로 쓸 수 있는
것 중 셸 인용과 광고 값에서 문제가 없는 좁은 집합이다. 날짜 모양 같은 형식 규칙은 없다

B) **이름 `ir` · 환경 변수 `IR`.** A 와 같되 환경 변수를 `IN` · `OUT` 과 같은 결로 짧게 둔다.
대가 — 빌드 스크립트가 이미 쓰는 이름과 부딪힐 수 있다

C) **이름 `ir` · 허용 문자를 git 규칙 전체로.** `git check-ref-format` 이 받는 것은 다 받는다.
대가 — 제품이 git 규칙을 옮겨 적고, 광고 값과 셸 인용에서 따로 막을 자리가 생긴다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5 — 명시로 켜는 훑기 `discover` 의 문법

FR-1 은 전체 훑기를 기본에서 끄고, 계약 작성자가 산출물 경로를 찾을 길로 **명시로 켜는 훑기**를
두었다. 상한(방문 수 · 시간 · 메모리 · 결과 크기)이 있고 결과는 진단이다. 계약에 어떻게 적나.

A) **단계 칸 `"discover": true`.** run · agent 단계만 받는다. 상한은 노드가 정하고 계약은 바꾸지
못한다 — 팩 보안 표의 「계약이 노드 설정을 지정하지 못한다」. `workspace` 칸을 요구하지 않는다
(오늘의 전체 훑기도 요구하지 않았다). 결과는 `produced` 가 아니라 result 진단이다

B) **단계 칸 `"discover": { "max_files": …, "max_time": … }`.** A 와 같되 계약이 상한을 노드
값보다 **낮출** 수만 있다

C) **계약 칸을 두지 않는다.** 노드 쪽 스위치(노드 설정이나 `enodectl`)로만 켠다. 계약 작성자가
자기 계약의 경로를 찾으려면 노드 소유자에게 부탁해야 한다

D) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6 — 계획이 새 칸을 쓸 수 있나 · Grammar 가 무엇을 가르치나

ADR-022 (계획 위임)에서 계획은 agent 가 짓는 계약 조각이고, `contract.Grammar` 가 그 agent 에게
문법을 가르친다. 유닛 정의는 「Grammar 에 effect 와 예산을 더한다」고만 적었다. 굽기 단계와
`discover` 는 적지 않았다.

A) **계획은 effect(`read` · `edit` · `build`) · budget · discover 를 쓸 수 있고, 굽기 단계
(build · merge)와 `prepare` 는 못 쓴다.** 늘어난 계약을 붙일 때 400 으로 거절한다. Grammar 는
effect · 예산 · discover 를 가르치고 굽기는 가르치지 않는다. 근거 — 합치기는 되돌리기 어렵고
(실행 계획의 위험도 High), 굽기는 사람이 쓰는 계약이다

B) **계획도 굽기 단계를 지을 수 있다.** Grammar 가 여섯 종류를 모두 가르친다

C) **계획은 새 칸을 하나도 못 쓴다.** 기본값으로만 돈다. Grammar 는 바뀌지 않는다

D) Other (please describe after [Answer]: tag below)

[Answer]: B

### Question 7 — 공개 굽기 예시와 IR 대조 (1.4)

`runctl example bake` 는 사내 명령을 담지 않고 공개 도구로 쓴다. 예시가 IR 대조와 맞아야
복사해 돌릴 수 있다.

A) **poky 를 git 으로 받는 예시.** `ir` 은 `yocto-5.0.20`, sync 는
`git fetch --tags origin && git checkout --detach "refs/tags/$ENODE_IR"`, builds 는
`qemux86-64` · `qemuarm64` 두 구성(`MACHINE` 만 다른 `bitbake core-image-minimal`). SunnyVM 의
시험 lower 와 같은 모양이라 장면 시험에 그대로 쓴다. **bake 유닛에 한 줄을 넘긴다** —
`.repo` 가 없으면 워크스페이스 git 의 HEAD 태그를 본다

B) **repo manifest 를 쓰는 공개 프로젝트로 쓴다.** AOSP 의 `platform/manifest` 와 `android-*`
태그처럼. IR 대조는 `.repo/manifests` 하나만 본다. 사내 모양에 가깝지만 SunnyVM 에서 돌려 볼 수
없는 크기다

C) **명령을 자리표시로 둔다.** `"sync": "<your sync command>"` 처럼 모양만 보인다. 예시 시험은
통과하지만 복사해 돌릴 수 없다

D) Other (please describe after [Answer]: tag below)

[Answer]: C

---

## 3. 산출물 계획 (체크박스)

답이 들어오고 모호함이 풀린 뒤에 채운다. 자리는
`aidlc-docs/v4-run-finalize-bake/construction/contract-grammar/functional-design/` 이다.

- [x] 답을 읽고 모호함을 확인한다 — 있으면 되물음 파일을 만든다 (2026-09-25T03:56:21Z · 물음 셋 — `contract-grammar-functional-design-clarification-questions.md`)
- [x] `domain-entities.md` — Step 에 더하는 칸 · 새 타입(Effect · Budget · Build · Merge ·
      Discover) · 종류 여섯과 판별 규칙 · 굽기 두 단계의 고정 산출물 이름
- [x] `business-rules.md` — 종류마다 받는 칸과 거절하는 칸의 표 · effect 기본값 ·
      400 규칙과 영어 거절 문구 전부 · 경계값 · 계획(expands)이 쓸 수 있는 것
- [x] `business-logic-model.md` — `Validate` 에 규칙이 들어가는 순서 · `EffectOrDefault` ·
      `Budgets` · 굽기 판정 흐름 · `runctl example bake` 의 내용 · lint 의 조건 제안 ·
      Grammar 에 더하는 문장과 그 문장을 확인하는 시험
- [x] 다른 유닛에 넘기는 것 — finalize · bake 가 받을 값과 이름 (이 계획 0절의 「안 닫는다」)
- [x] 정본 되돌림 목록 — run-contract.md 와 ADR-077 에 올릴 줄
- [x] 표기 검사 (`enode-design/scripts/emphasis-check.py`) · 사용자가 싫어한 말투 검사 ·
      사내 이름 검사

---

## 4. 확장 준수 — 이 단계

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 「계약이 노드 설정이나 host 경로를 지정하지 못한다」는 물음 3 · 5 의 칸 규칙이 닫는다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 저장소의 표 시험 관례를 따른다 |
