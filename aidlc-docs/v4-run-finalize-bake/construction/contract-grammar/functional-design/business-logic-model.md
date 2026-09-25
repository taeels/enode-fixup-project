# `contract-grammar` — 흐름

타입은 `domain-entities.md`, 규칙과 거절 문구는 `business-rules.md` 에 있다. 여기는 규칙이
어느 순서로 걸리나, 굽기가 어떻게 판정되나, 계약 작성 도구와 Grammar 가 무엇을 말하나, 그리고
다른 유닛에 무엇을 넘기나다.

---

## 1. `Validate` 에 규칙이 들어가는 순서

오늘의 `Validate` (`internal/contract/contract.go:1026`)는 단계 목록을 여러 번 훑으며 주제마다
한 덩어리씩 본다. 새 규칙도 같은 방식으로 덩어리를 더한다. 순서가 중요한 자리는 셋이다.

```text
   1   종류 판별                   kinds 표를 만드는 첫 훑기 (contract.go:1066 무렵).  여섯 종류
   2   종류마다 받는 칸             build · merge 의 허용 칸, effect 표, budget 표, discover, ir 위치
                                  -> business-rules.md 2 · 3 · 4 · 7절
   3   build · merge 의 값          sync · builds · 이름 · 명령 · ir 문자 · 기간
   ... 오늘의 덩어리들 (needs · dispatch · 예약 이름 · in.from · expands · adopt · ask · loop ·
       acquire · release) ...
   4   굽기 계약의 모양             business-rules.md 5절.  needs 는 NeedsOf 로 채운 뒤 본다
   5   계획이 짓는 굽기             business-rules.md 6절.  ask 가 build 의 조상인지는 오늘의
                                  ancestors 를 그대로 쓴다
   6   판정 조건                   오늘의 success_when 덩어리에 build · merge 줄을 더한다 (8절)
```

- **2 가 4 보다 먼저다.** 칸이 틀린 build 단계를 모양 규칙이 먼저 보면 「merge 가 없다」처럼
  원인이 아닌 문구가 나간다
- **4 가 오늘의 덩어리 뒤다.** needs 가 앞 단계만 가리키는지, dispatch 가 뒤만 가리키는지를
  오늘의 덩어리가 먼저 확인해야 모양 규칙이 needs 를 믿고 쓴다
- **5 는 4 뒤다.** 모양이 맞아야 「build 앞의 계획 단계」를 찾을 수 있다

첫 오류에서 멈춘다 — 오늘의 `Validate` 가 그렇다. 계약 저자는 고치고 다시 낸다.

---

## 2. 기본값

```text
   EffectOrDefault   effect 가 있으면 그 값.  없으면 run -> build · agent -> edit.
                     build 는 Validate 가 prepare 를 요구하므로 언제나 적혀 있다.
                     merge · ask · acquire 는 ""
   Budgets           finalize 없으면 1m · upload 없으면 3m.  파싱은 Validate 가 이미 통과시켰다
   MergeWait         wait 없으면 4h
```

세 메서드는 `Validate` 를 지난 계약에서만 부른다. 파싱 오류를 다시 다루지 않는다 — 지나지 않은
계약은 Mediator 에 저장되지 않으므로 노드에 오지 않는다.

---

## 3. 굽기의 판정 (답 1 = A)

```text
   계약                     노드 (bake 유닛)                    Mediator
   build: sync · builds · ir
                            sync -> builds 를 차례로
                            모두 0 이고 HEAD 에 ir 태그가 있다
                              -> $OUT/manifest 를 쓴다
                            하나라도 아니다
                              -> manifest 를 안 쓴다.  실패의 기록은
                                 result 진단과 state.json last_attempt
                            result (produced 에 manifest 가 있거나 없다)
                                                                 Verify: produced ["manifest"]
   merge: wait
                            형제를 wait 까지 기다린다 -> 합친다
                              -> $OUT/merged 를 쓴다
                            못 합쳤다 -> merged 를 안 쓴다
                                                                 Verify: produced ["merged"]
```

- `Verify` (`internal/store/verdict.go:128`)는 바뀌지 않는다. 오늘의 produced 대조가 그대로 판정한다
- build 가 실패해 manifest 가 없어도 merge 는 needs 를 따라 돈다. 합칠 upper 가 대기 자리에
  없으므로 합치지 않고 merged 도 없다 — 그것을 어떻게 알리는지는 bake 유닛이 닫는다. Run 은
  build 의 조건에서 이미 실패다
- 판정 조건이 없는 굽기 계약은 lint 경고다. `Validate` 는 막지 않는다 (ADR-061 §2)

---

## 4. 계획이 짓는 굽기 (답 6 = B · 되물음 1 · 2 = A)

```text
   제출         [plan(agent · expands · produces [build, merge]),
                 approve(ask · adopts: plan · adopt_when: approve)]
                success_when: build produced [manifest] · merge produced [merged]
                -> Validate: 굽기 계약이 아니다 (build · merge 가 아직 없다).  약속한 이름이라
                   조건의 종류 검사를 건너뛴다 (오늘 그대로)
   plan 끝      계획이 [build, merge] 를 짓는다 -> 계약 끝에 붙인다
                -> Validate (늘어난 계약): 굽기 계약이다.  1 ~ 5 덩어리가 모두 걸린다
                   yolo 아님 · approve 가 plan 을 지목 · adopt_when 있음 · build 가 approve 를 기다림
                   하나라도 어기면 붙이지 않고 plan 이 실패한다 (오늘의 계획 거절 경로)
   approve      사람이 답한다
                approve -> 계획이 제안한 success_when 채택.  build 가 돈다
                그 밖    -> 계획을 물리고 plan 으로 되돌아간다 (retirePlan · rewindToPlanner)
```

build 의 needs 기본값은 「바로 앞 단계」 = approve 다. 계획이 needs 를 따로 적어 approve 를
건너뛰면 5 덩어리가 거절한다.

---

## 5. Grammar 와 PlanShape 에 더하는 문장 (답 6 = B)

계획을 짓는 agent 에게 주입된다. 영어다 (`CONVENTIONS.md` 2.1). 아래는 초안이고 Code
Generation 이 다듬는다. 문장마다 `grammar_test.go` 의 기존 장치(「문법이 금지하는 것이 실제로
거절되나」)에 표 줄을 하나씩 더한다.

**Grammar — 종류 표를 여섯으로**

```text
### There are six kinds of step, with different fields

    run       uses, run (argv array), out
    agent     uses, agent, in, out, schema
    ask       no uses, ask, out (exactly one), schema (required)
    acquire   no uses, acquire; performed by the mediator, not by a node
    build     uses, effect "prepare", sync, builds, ir; bakes on the node
    merge     uses, merge; merges what build baked, performed by the node
```

**Grammar — effect · budget · discover**

```text
### What a step does to the workspace: effect

    run     build (default), edit, read
    agent   edit (default), read
    build   prepare, always written

Write "effect": "edit" on a run step when the command changes source files,
such as a formatter or a code generator. Without it the step returns no diff.
Other kinds take no effect.

### Time after the command ends: budget

    "budget": { "finalize": "5m", "upload": "10m" }

finalize defaults to 1m and can only be raised. upload defaults to 3m and must
be greater than zero. Only run, agent and build steps take a budget.

### Listing what changed: discover

"discover": true on a run or agent step lists what changed in the workspace,
within limits the node sets. The list is a diagnostic. Nothing in it counts as
produced.
```

**Grammar — 굽기**

```text
### A bake: build, then merge

A contract that bakes ends with exactly one build step and one merge step, in
that order, on the same role. merge needs exactly [build]. Only planning steps
may come before them: an agent step with expands, and the ask that adopts its
plan.

ir is the exact tag to bake. The node passes it to sync and builds as ENODE_IR.
builds names use a-z, 0-9 and -, and must not repeat.

Judge build by produced ["manifest"] and merge by produced ["merged"]. No other
condition applies to them.

A plan that builds a bake is never adopted with adopt "yolo". An ask step must
adopt it and set adopt_when, and the build step must wait for that ask.
```

**PlanShape — 모양 예시 하나** (자리표시 값은 되물음 3 = A 와 같다)

```text
### The shape of a bake
    { "id": "build", "uses": "baker", "effect": "prepare", "ir": "your-ir-tag",
      "sync": "<sync command>",
      "builds": [ { "name": "config-a", "command": "<build command>" } ] }
    { "id": "merge", "uses": "baker", "needs": ["build"], "merge": { "wait": "4h" } }
```

PlanShape 의 굽기 예시는 계약 하나에 넣었을 때 `Validate` 를 지나야 한다 — 시험을 하나 더한다.

---

## 6. 계약 작성 도구

### 6.1 `runctl example bake` (답 7 = C · 되물음 3 = A)

새 파일 `internal/contract/examples/bake.json`. 사내 명령과 이름을 담지 않는다.

```json
{
  "run_id": "example-bake",
  "work": {
    "id": { "system": "manual", "change_id": "example-bake" },
    "system": "manual"
  },
  "requires": [
    { "as": "baker", "capability": "agent.reason", "workspace.writes": "isolated" }
  ],
  "steps": [
    {
      "id": "build",
      "uses": "baker",
      "effect": "prepare",
      "ir": "your-ir-tag",
      "sync": "<your sync command; $ENODE_IR is the tag to bake>",
      "builds": [
        { "name": "config-a", "command": "<your build command for config-a>" },
        { "name": "config-b", "command": "<your build command for config-b>" }
      ]
    },
    {
      "id": "merge",
      "uses": "baker",
      "needs": ["build"],
      "merge": { "wait": "4h" }
    }
  ],
  "success_when": [
    { "step": "build", "produced": ["manifest"] },
    { "step": "merge", "produced": ["merged"] }
  ]
}
```

- `TestExamples_ParseAndValidate` 와 `TestExamples_EveryStepHasASuccessCondition` 을 고치지 않고
  지난다
- `requires` 의 `workspace.writes: isolated` — 격리 runtime 노드만 굽는다 (ADR-077 「공유 lower 위에
  native 굽기 노드를 둔다」 기각). 이 광고를 내는 lower-state 유닛이 들어오기 전에는 이 예시를 베낀
  계약이 422(후보 없음)로 떨어진다. 7절의 창을 닫는 효과가 있다

### 6.2 `runctl lint`

```text
   조건 제안     suggestCondition 에 build · merge 갈래를 더한다.  run 갈래 앞에 둔다
                 build -> { "step": "<id>", "produced": ["manifest"] }
                 merge -> { "step": "<id>", "produced": ["merged"] }
   새 경고 둘     business-rules.md 9절 — workspace.writes: isolated 를 안 요구한 build 역할 ·
                 굽기를 지을 것으로 보이는 계획의 승인 설정
```

`runctl schema` 는 구조체에서 칸을 뽑으므로 저절로 보인다. `runctl capabilities` 와 `dry-run` 은
이 유닛과 무관하다.

---

## 7. 문법이 먼저 들어오고 뜻은 뒤 유닛이 채운다

이 유닛은 한 줄 순서의 첫째다. `main` 에 병합된 뒤, 뜻을 채우는 유닛이 병합되기 전까지의 창을
적는다.

```text
   칸                        그 사이에 무슨 일이 생기나                             닫는 유닛
   effect · budget · discover  Mediator 가 받아 계약과 함께 봉인한다.  노드는 모른다     finalize
                              (claim 이 싣지 않는다).  오늘처럼 돈다.  해 없음
   build · merge 단계         Mediator 가 받는다.  예시처럼 workspace.writes 를 요구하면   bake
                              후보가 없어 422.  요구하지 않은 계약은 노드가 claim 하고
                              「run step has an empty argv」로 실패한다
                              (internal/enode/claim.go:625).  합치기가 일어나지 않으므로
                              해는 없지만 문구가 틀린 말을 한다
```

---

## 8. 다른 유닛에 넘기는 것

```text
   step-phase   Claimed (internal/store/claim.go:66) 에 새 칸을 싣는다 — effect · budget ·
                discover · sync · builds · ir · merge.  노드가 받는 칸은 Claimed 뿐이다.
                유닛 정의와 파일 행렬에 없던 일이다.  claim.go 는 step-phase 의 파일이라
                거기 둔다.  권장 — 계약의 칸을 그대로 싣고, 기본값은 노드가 contract 의
                메서드로 채운다 (Mediator 와 노드가 같은 상수를 읽는다)
   finalize     effect 에 따라 RecordDiff 와 전체 훑기를 켜고 끈다 · read 는 build 와 같다 ·
                두 예산을 지킨다 · discover 의 상한 값과 진단을 싣는 자리
   lower-state  workspace.writes 광고.  예시의 requires 와 lint 경고가 이 키를 쓴다
   bake         sync 와 builds 를 셸로 돈다 · ENODE_IR 을 넘긴다 · IR 대조
                (SunnyVM 의 poky 는 git 으로 받아 .repo 가 없다 — 되물음 파일 끝) ·
                성공했을 때만 manifest · 합쳤을 때만 merged · merge.wait 을 센다 ·
                제자리에 쓰는 노드(native)가 굽기 단계를 받으면 거절한다 · 7절의 틀린 문구
```

---

## 9. 정본에 되돌려 올리는 것

진행자가 `enode-design` 에 올린다 (`README.md` 3.1 — 설계 정본은 여기서 안 고친다).

```text
   protocol/run-contract.md   단계 종류 여섯 · effect 표 · budget · discover · build 와 merge 의 칸 ·
                              ir 과 문자 규칙 · 고정 산출물 이름 · 굽기 계약의 모양 ·
                              계획이 짓는 굽기의 승인 규칙 · exit_code 거절 문구
   ADR-077 §2                 merge 의 표기 — 스케치의 kind: merge 대신 merge 객체가 판별 칸이다 ·
                              성공 판정은 produced manifest · merged
   ADR-077 §5 · §11           ir 은 계약이 적는 값이다 (계획 2.3 이 이미 올릴 목록에 넣었다)
   ADR-075 §16                effect 를 새 필드로 적는다 — read · edit · build 몫도 닫는다
                              (prepare 몫은 ADR-077 §2 가 닫았다)
```

---

## 10. 확장 준수 — Functional Design

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 「계약이 노드 설정이나 host 경로를 지정하지 못한다」는 build · merge 의 허용 칸(`workspace` 거절)과 `discover` 의 상한을 노드가 정하는 규칙이 닫는다 |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 — 저장소의 관례다 |

NFR Requirements 와 NFR Design 은 건너뛴다 (유닛 정의 — 성능 표면이 없다). 다음 단계는 Code
Generation 이다.
