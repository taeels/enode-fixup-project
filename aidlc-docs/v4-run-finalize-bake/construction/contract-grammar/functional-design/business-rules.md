# `contract-grammar` — 규칙

`Validate` 가 400 으로 막는 규칙과 그 거절 문구다. 문구는 영어다 (`CONVENTIONS.md` 2.1 —
에러 문자열은 밖으로 나간다). `%q` 자리에는 단계 id 나 값이 들어간다. 타입과 이름은
`domain-entities.md` 에 있다.

**원칙 하나** — 규칙은 「무엇이 틀렸나」와 함께 「무엇을 쓰면 되나」를 말한다. 이 저장소의
거절 문구가 이미 그렇게 쓴다 (예: `changed requires a workspace on that step; there is no
baseline to compare against`).

---

## 1. 종류 판별

| 규칙 | 거절 문구 |
|---|---|
| 판별 칸이 둘 이상 | `step %q: more than one of agent, run, ask, acquire, build (sync, builds), merge is set` |
| 판별 칸이 없음 | `step %q: none of agent, run, ask, acquire, build (sync, builds), merge is set` |

build 의 판별 칸은 `sync` 와 `builds` 둘이다. 둘 중 하나만 있어도 build 로 보고, 빠진 쪽은
3절이 거절한다.

---

## 2. effect 와 budget — 종류마다 (답 2 = A)

```text
   종류       effect                          budget
   run        build (기본) · edit · read       받는다
   agent      edit (기본) · read               받는다
   build      prepare — 반드시 적는다           받는다
   merge      400                              400.  기다림은 merge.wait
   ask        400                              400
   acquire    400                              400
```

| 규칙 | 거절 문구 |
|---|---|
| 모르는 값 | `step %q: unknown effect %q; use read, edit, build or prepare` |
| merge · ask · acquire 에 effect | `step %q: a %s step takes no effect` |
| run · agent 에 prepare | `step %q: effect prepare is only for a build step (sync and builds)` |
| agent 에 build | `step %q: effect build is not allowed on an agent step; use read or edit` |
| build 단계에 effect 가 없거나 prepare 가 아님 | `step %q: a build step must say effect "prepare"; it decides whether the upper is merged, so it is written, not inferred` |
| merge · ask · acquire 에 budget | `step %q: a %s step takes no budget` — merge 면 뒤에 `; merge.wait sets how long it waits` 를 붙인다 |
| 기간 형식이 아님 | `step %q: budget.finalize %q is not a duration (for example "90s" or "5m")` · upload 도 같은 모양 |
| finalize 가 1분 아래 | `step %q: budget.finalize %s is below the default 1m0s; it can only be raised` |
| upload 가 0 이하 | `step %q: budget.upload must be greater than zero` |

**경계값** — finalize 는 정확히 `1m` 이면 받는다 (기본값과 같다). upload 는 `1ns` 도 받는다 —
위쪽과 아래쪽 모두 상한을 두지 않는다 (Application Design Q6). `"budget": {}` 는 받는다 — 두
값 모두 기본값이다.

`read` 는 이번 회차에서 `build` 와 같이 diff 와 전체 훑기를 안 한다. source 를 쓴 것을 위반으로
잡는 일은 범위 밖이다 (ADR-075 §5 의 「edit effect 가 없는 source write 는 위반」은 뒤 회차).

---

## 3. build 단계 (답 3 = A · 답 4 = A)

**받는 칸** — `id` · `uses` · `needs` · `effect`(prepare) · `sync` · `builds` · `ir` · `env` ·
`budget`. 그 밖의 칸이 있으면 400 이다.

| 규칙 | 거절 문구 |
|---|---|
| `uses` 없음 | `step %q: a build step needs uses; it runs on a node` |
| `sync` 없음 · 빈 문자열 | `step %q: a build step needs sync` |
| `builds` 없음 | `step %q: a build step needs at least one entry in builds` |
| 이름 규칙 | `step %q: builds[%d].name %q must be 1 to 64 characters of a-z, 0-9 and -` |
| 이름 겹침 | `step %q: builds name %q appears twice` |
| 명령이 빈 문자열 | `step %q: builds[%d].command is empty` |
| `ir` 없음 | `step %q: a build step needs ir, the exact tag to bake` |
| `ir` 문자 규칙 (아래) | `step %q: ir %q is not a valid tag name: %s` — 마지막 자리에 어긴 규칙 하나 |
| `workspace` | `step %q: a build step does not take workspace; sync prepares the source` |
| `out` | `step %q: a build step does not take out; it produces "manifest" when the build succeeds` |
| 그 밖의 칸 | `step %q: a build step does not take %s` — 칸 이름 |

빈 문자열 판정은 앞뒤 공백을 걷은 뒤에 한다. 명령의 내용은 보지 않는다 — 제품은 그 문자열을
해석하지 않는다.

이름 길이 64 는 이 단계가 정한 경계다. 이름이 광고 키 `repo.built.<이름>` 의 한 조각이 되므로
키가 한없이 길어지지 않게 한다.

### 3.1 `ir` 의 문자 규칙

git 태그 이름으로 쓸 수 있는 것 중, 환경 변수 값과 광고 값과 셸 인용에서 문제가 없는 좁은 집합이다.
날짜 모양 같은 형식 규칙은 없다 — 사내 형식은 제품이 모른다.

```text
   길이           1 ~ 128 자
   글자           A-Z  a-z  0-9  .  _  -  /
   시작           - / . 로 시작하지 않는다
   끝             / . 로 끝나지 않는다
   조각           / 로 나눈 조각마다 . 로 시작하지 않고 .lock 으로 끝나지 않는다
   연속           .. 과 // 를 담지 않는다
```

「조각」과 「끝이 `.`」 줄은 답 4 의 문장에 없던 것이다. git 의 참조 이름 규칙
(`git check-ref-format`)이 거절하는 것이라 더했다 — 받으면 노드가 sync 뒤 대조에서 반드시
실패한다.

`ir` 이 build 가 아닌 단계에 있으면 400 이다 — `step %q: ir is only for a build step`.

---

## 4. merge 단계

**받는 칸** — `id` · `uses` · `needs` · `merge`. 그 밖의 칸이 있으면 400 이다.

| 규칙 | 거절 문구 |
|---|---|
| `uses` 없음 | `step %q: a merge step needs uses; it runs on the node that built` |
| `merge.wait` 형식 | `step %q: merge.wait %q is not a duration (for example "4h")` |
| `merge.wait` 가 0 이하 | `step %q: merge.wait must be greater than zero` |
| 그 밖의 칸 | `step %q: a merge step does not take %s` |

---

## 5. 굽기 계약의 모양 (답 3 = A · 되물음 1 = A)

build 나 merge 가 하나라도 있으면 굽기 계약이다. 굽기 계약은 아래를 모두 지킨다.

| 규칙 | 거절 문구 |
|---|---|
| build 가 둘 이상 | `the contract has %d build steps; a bake has exactly one` |
| merge 가 둘 이상 | `the contract has %d merge steps; a bake has exactly one` |
| build 뒤에 merge 가 없음 | `build step %q has no merge step; the built upper would wait forever` |
| merge 앞에 build 가 없음 | `merge step %q has no build step to merge` |
| merge 가 마지막이 아님 | `step %q comes after merge step %q; a bake ends with its build and merge` |
| build 와 merge 사이에 단계가 있음 | `step %q sits between build %q and merge %q; merge follows build directly` |
| merge 의 needs 가 정확히 [build] 가 아님 | `merge step %q must need exactly [%q]` |
| 역할이 다름 | `merge step %q must use the same role as build step %q (%q)` |
| build 앞에 계획 단계가 아닌 단계 | `step %q comes before the bake but is not a planning step; only an agent step with expands and the ask that adopts its plan may come first` |

needs 는 `NeedsOf` 로 기본값을 채운 뒤 본다 — merge 가 build 바로 뒤에 있고 needs 를 안
적었으면 [build] 이다.

**계획 단계** — `expands: true` 인 agent 단계와, `adopts` 를 적은 ask 단계. ask 가 승인하는
계획이 그 계약 안의 expands 단계여야 하는 규칙은 오늘의 `Validate` 가 이미 본다.

---

## 6. 계획이 짓는 굽기 (답 6 = B · 되물음 2 = A)

계획(계약 안의 expands 단계가 지어 붙이는 단계들)도 build 와 merge 를 지을 수 있다. 붙인 뒤의
계약이 `Validate` 를 다시 받으므로 1 ~ 5절이 그대로 걸린다.

**되물음 2 의 전제를 코드에 맞게 고쳤다.** 되물음 파일은 「ask 의 `adopts` 로 사람이 본 계획만
붙는다」고 적었다. 코드는 그렇지 않다.

```text
   계획의 단계     expands 단계가 끝나면 곧바로 계약 끝에 붙는다 (internal/store/expand.go)
   ask 의 adopts   계획이 제안한 success_when 을 채택할지를 정한다 (ADR-033)
   거절            ask 에 adopt_when 이나 dispatch 가 있을 때만 계획을 물리고 계획 단계로
                  되돌아간다 (retirePlan · rewindToPlanner — internal/store/ask.go:359).
                  둘 다 없으면 거절해도 붙은 단계가 그대로 돈다
   순서            붙은 단계의 needs 기본값은 「바로 앞 단계」다.  계획이 needs 를 적으면
                  ask 를 건너뛸 수 있다
```

그래서 「사람이 승인해야 굽는다」를 지키려면 아래 넷이 모두 필요하다. 굽기 계약에 expands
단계가 있을 때 건다.

| 규칙 | 거절 문구 |
|---|---|
| expands 단계가 `adopt: "yolo"` | `step %q adopts its plan without asking (adopt "yolo"), but the contract bakes; a bake merges into a shared lower, so a person must adopt the plan (an ask step with adopts)` |
| 그 expands 를 `adopts` 로 지목한 ask 가 없음 | `step %q builds a plan in a contract that bakes, but no ask step adopts it; a person must adopt the plan before the bake runs (an ask step with adopts: %q)` |
| 그 ask 에 `adopt_when` 도 dispatch 도 없음 | `ask step %q adopts plan %q in a contract that bakes, but says nothing about rejection; set adopt_when so that a rejected plan is rebuilt instead of baked` |
| build 가 그 ask 를 기다리지 않음 (ask 가 build 의 조상이 아님) | `build step %q does not wait for ask step %q; a bake runs only after a person adopts the plan (add it to needs)` |

규칙을 「계획이 지었나」가 아니라 「굽기 계약에 계획 단계가 있나」로 적는다. `Validate` 는 어느
단계를 계획이 지었는지 모르고, 알 필요도 없다 — 굽기 계약에서 계획 단계는 build 앞에만 올 수
있고, 그 계획이 무엇을 짓든 사람이 보고 난 뒤에 굽기가 돌아야 한다.

**이 규칙들이 걸리는 때** — 제출 때 계약에는 아직 build · merge 가 없으므로 걸리지 않는다.
계획이 굽기를 지어 붙이는 순간 늘어난 계약의 `Validate` 가 거절하고, 계획 단계는 오늘의 계획
거절 경로로 실패한다. 제출 때 미리 막지 않는 이유 — 계약이 계획에게 굽기를 맡겼는지를 제출
때 확실히 알 길이 없다. `success_when` 이 약속한 이름에 `manifest` 를 걸었다는 것이 단서지만,
보통 단계도 `manifest` 라는 산출물을 낼 수 있어 400 의 근거로는 약하다. 그 단서는 lint 가
경고로 쓴다 (9절).

## 7. `discover` (답 5 = A)

| 규칙 | 거절 문구 |
|---|---|
| run · agent 가 아닌 단계에 `discover` | `step %q: discover is only for a run or agent step` |

`discover` 는 `workspace` 칸을 요구하지 않는다 — 오늘의 전체 훑기도 요구하지 않았다. 상한은
노드가 정한다. 계약은 켜기만 한다.

---

## 8. 판정 조건 (답 1 = A)

| 규칙 | 거절 문구 |
|---|---|
| run 이 아닌 단계에 `exit_code` | `exit_code condition is allowed only on a run step: %q` |
| build 에 `produced` 외의 조건 · `manifest` 외의 이름 | `success_when: step %q is a build step; judge it by produced ["manifest"]` |
| merge 에 `produced` 외의 조건 · `merged` 외의 이름 | `success_when: step %q is a merge step; judge it by produced ["merged"]` |

- 첫 줄은 **기존 문구를 고친다.** 오늘은 `exit_code condition is not allowed on an agent step` 인데
  ask · acquire 에도 같은 문구가 나갔고, build · merge 까지 받으면 틀린 말이 넷이 된다. 오류 값
  `ErrExitOnAgent` 의 이름은 그대로 둔다 — `errors.Is` 로 부르는 쪽이 있다
- build · merge 에 `changed` · `within_attempts` · `exit_code` 를 못 거는 이유 — 셋 다 그 단계에서
  공허하게 참이거나 언제나 거짓이다. 조용히 통과하면 계약 저자가 판정이 없는 줄 모른다
- 판정 조건이 아예 없는 굽기 계약은 400 이 아니라 lint 경고다 (ADR-061 §2 — 경고 먼저).
  오늘의 경고 「no condition judges step」가 그대로 걸린다

---

## 9. 계약 작성 도구

```text
   runctl example bake   답 7 = C · 되물음 3 = A.  명령은 <...> 자리표시.  ir 은 "your-ir-tag",
                         구성 이름은 config-a · config-b.  예시 시험을 고치지 않는다
   runctl lint 의 제안   build -> { "step": "<id>", "produced": ["manifest"] }
                         merge -> { "step": "<id>", "produced": ["merged"] }
   runctl lint 의 경고   (새) build 단계의 역할이 requires 에 workspace.writes: isolated 를 안 적었다.
                         문구: warning: build step %q uses role %q, which does not require
                         workspace.writes=isolated; a node that writes in place cannot bake
                         (새) success_when 이 계획이 약속한 이름에 produced ["manifest"] 나
                         ["merged"] 를 건다 — 계획이 굽기를 지을 것이라는 단서다.  그런데 6절의
                         넷 가운데 제출 때 볼 수 있는 셋(yolo 아님 · 지목한 ask · adopt_when)이
                         안 맞으면 경고한다.  문구: warning: the plan of step %q is expected to
                         bake (success_when waits for %q), but %s; the plan will be rejected
                         when it is attached
   runctl schema         구조체에서 칸을 뽑으므로 저절로 보인다.  코드 변경 0
```

**설계가 더한 것 하나** — `workspace.writes: isolated` 경고는 답에 없던 것이다. 굽기는 공유
lower 위의 격리 runtime 에서만 돈다 (ADR-077 「공유 lower 위에 native 굽기 노드를 둔다」 기각).
400 으로 올리지 않은 이유 — `Validate` 가 광고 어휘(`workspace.writes`)를 문법으로 들이면 매칭의
어휘와 문법의 어휘가 두 벌이 된다.

---

## 10. Grammar 가 가르치는 것 (답 6 = B)

계획을 짓는 agent 에게 주는 문법(`contract.Grammar`)과 모양 예시(`contract.PlanShape`)가 여섯
종류와 새 칸을 모두 가르친다. 문장마다 「그 규칙을 어긴 계약이 실제로 거절되나」를 시험이
확인한다 (`grammar_test.go` 의 기존 장치 — 문법이 늘었는데 거절이 안 늘면 시험이 깨진다).
문장 초안은 `business-logic-model.md` 5절에 있다.
