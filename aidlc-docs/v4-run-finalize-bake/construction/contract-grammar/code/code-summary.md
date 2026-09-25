# `contract-grammar` — 코드 요약

계획은 `construction/plans/contract-grammar-code-generation-plan.md` (단계 열다섯 · 승인
2026-09-25T05:14:18Z), 설계는 `construction/contract-grammar/functional-design/` 의 셋이다.
아래의 FD 는 Functional Design 의 줄임이다.

- **브랜치** `unit/contract-grammar` · 기준 `2949bd0`
- **패키지 둘** `internal/contract` · `cmd/runctl`. 그 밖의 소스 diff 0 · DB 스키마 변경 0 · 새 import 0

---

## 1. 파일

| 파일 | 새 · 고침 | 무엇 |
|---|---|---|
| `internal/contract/contract.go` | 고침 (+82 −6) | Step 칸 일곱 · `KindBuild` · `KindMerge` · `Kind()` 의 판별과 문구 · `Validate` 를 `validate(stubs)` 로 옮김 · 호출 세 자리 · `ErrExitOnAgent` 문구 |
| `internal/contract/effect.go` | 새 (201 줄) | `Effect` 와 값 넷 · `Budget` · 예산 상수 셋 · `EffectOrDefault` · `Budgets` · `checkStepFields` · `checkEffect` · `checkBudget` · `aKind` |
| `internal/contract/bake.go` | 새 (348 줄) | `Build` · `Merge` · `DefaultMergeWait` · `ArtifactManifest` · `ArtifactMerged` · `EnvIR` · `MergeWait` · `otherFields` · build 와 merge 의 칸 규칙 · `irProblem` · `checkBake` · `checkPlannedBake` · `checkBakeCondition` |
| `internal/contract/checkplan.go` | 고침 (+10 −2) | 그루터기 이름을 `validate` 에 넘긴다 |
| `internal/contract/grammar.go` | 고침 (+50 −1) | 종류 표 여섯 · effect · budget · discover · 굽기 절 · 판정 조건 표 두 줄과 문장 하나 |
| `internal/contract/planshape.go` | 고침 (+11) | 굽기 모양 하나와 자리표시 안내 한 줄 |
| `internal/contract/examples/bake.json` | 새 | FD 흐름 6.1절의 JSON 그대로 |
| `cmd/runctl/shape.go` | 고침 (+108) | `suggestCondition` 의 build · merge 갈래 · `bakeWarnings` (경고 둘) · `lintWarnings` 를 둘로 나눔 |
| 시험 여섯 | 새 둘 · 고침 넷 | `effect_test.go` · `bake_test.go` (새) · `contract_test.go` · `checkplan_test.go` · `grammar_test.go` · `cmd/runctl/shape_test.go` |

파일 행렬(`unit-of-work-file-matrix.md` 1.1절) 밖의 파일은 계획 2절이 적은 넷 그대로다 —
`effect.go` · `bake.go` · `planshape.go` · `checkplan.go`. 그 밖은 0 이다.

---

## 2. 규칙이 어디에 있나

```text
   Validate 의 자리                          부르는 함수                 규칙 (FD business-rules.md)
   종류 판별 훑기 · Kind() 바로 뒤              checkStepFields             2 · 3 · 4 · 7절
     effect 표                                 checkEffect
     budget 을 받는 종류 · discover · ir 위치   checkStepFields 안
     build 의 칸과 값 · ir 문자                  checkBuildStep · irProblem
     merge 의 칸과 값                           checkMergeStep
     예산의 값                                  checkBudget
   adopt 덩어리 뒤 · success_when 앞            checkBake                   5절
                                              checkPlannedBake            6절
   success_when 덩어리 · exit_code 검사 뒤      checkBakeCondition          8절
```

**「그 밖의 칸」** — `otherFields` 가 Step 의 칸을 (JSON 이름, 적혔나) 목록으로 훑는다.
`TestOtherFields_CoverEveryStepField` 가 reflect 로 칸을 하나씩 채워 보고, 판별 칸이나 따로 보는
칸이 아닌데 목록이 이름을 안 대면 깨진다. `loop` 을 목록에서 빼 보고 시험이 깨지는 것을 확인했다.

---

## 3. 계획 3절의 결정 — 그루터기

`Validate()` 는 `validate(nil)` 을 부르고, `CheckPlan` 만 그루터기 이름을 넘긴다. `checkBake` 의
「build 앞에는 계획 단계만」 한 줄이 그 이름을 건너뛴다. 나머지 규칙은 그대로 걸린다.

**문제가 실재하는지 확인했다.** `CheckPlan` 이 `validate(nil)` 을 부르게 바꾸면
`TestCheckPlan_ABakePlanNamingTheApprovalPasses` 가 이 문구로 깨진다 —
`step "approve" comes before the bake but is not a planning step`. 되돌리면 지난다.
같은 시험이 같은 계획을 진짜 부모(plan · approve) 뒤에 붙인 계약은 `Validate()` 를 지나고, 훅이
만드는 모양을 `Validate()` 에 그대로 주면 거절되는 것까지 본다.

---

## 4. 설계를 다듬은 자리

설계의 규칙은 바꾸지 않았다. 문구와 가르치는 문장에서 넷을 다듬었다.

```text
   관사                 FD 문구의 "a %s step" 은 ask · acquire 에서 "a ask step" 이 된다.
                        aKind 가 "an ask" · "an acquire" · "an agent" 로 붙인다
   Grammar 의 effect    FD 초안의 "Without it the step returns no diff." 를 뺐다.
                        노드가 effect 를 읽는 것은 finalize 유닛이라 병합 뒤 그때까지 거짓이다
                        (ADR-053 — 문법은 있는 기계를 가르친다).  선언하는 것만 적었다
   Grammar 의 굽기       FD 초안에 더해 "A build step needs sync, at least one entry in builds,
                        and ir." · "A build step takes no workspace and no out" · merge.wait
                        한 줄을 적었다.  모양 예시는 PlanShape 에만 둔다 (두 벌 금지)
   PlanShape            "Angle brackets and your-ir-tag mark placeholders. ... do not invent them."
                        자리표시를 계획이 그대로 베끼면 Validate 는 지나고 노드에서 셸이 실패한다
   lint 경고             workspace.writes 경고 밑에 붙여넣을 한 줄을 단다 — 기존 경고가 제안을
                        붙이는 방식과 같다
```

---

## 5. 코드 검사 (`unit-of-work.md` 0절 · CI 와 같은 명령)

| 검사 | 결과 |
|---|---|
| `gofmt -l .` · `go vet ./...` · `go build ./...` | 빈 출력 · exit 0 · exit 0 |
| `eval "$(scripts/testdb.sh)"` 뒤 `go test ./... -count=1 -coverpkg=./... -json` | exit 0 · 통과 1,891 · 실패 0 · 스킵 0 |
| 패키지마다 커버리지 (`-coverpkg=./...` · CI 의 awk 그대로) | 스무 패키지 전부 80% 이상 · 전체 85.7% (Reverse Engineering 측정 85.3%). `internal/contract` 92.3% · `cmd/runctl` 94.5% |
| `internal/contract` 자기 시험만 | 80.6% -> 87.2% |
| 새 함수 | `bake.go` · `effect.go` 의 함수 전부 100% · `validate` 93.1% · `bakeWarnings` 97.1% |
| 크로스 빌드 셋 | windows/amd64 · linux/arm (GOARM=7) · darwin/arm64 모두 exit 0 |
| `enodectl.exe` 심볼 | crypto/tls 1 · net/http 6 (상한 10 · 50) |
| U+2605 · `go run ./scripts/glyphscan.go` | 0 · 134 파일에 장식 문자 0 |
| 코드 경계 시험 `TestImportBoundaries` | 통과 |
| `cmd/enodectl/probe.lock` | 측정이 바꿔 되돌렸다 (Reverse Engineering 이 적은 알려진 부채) |

---

## 6. `runctl schema steps`

코드 변경 없이 새 칸 일곱이 보인다. `Budget` · `Build` · `Merge` 의 안쪽 칸은 이 명령이 펼치지
않는다 — 오늘 `Ask` · `Loop` 과 같다. 안쪽 모양은 `runctl example bake` 와 Grammar 가 보인다.

```text
  loop           Loop                     optional
  effect         Effect                   optional
  budget         Budget                   optional
  discover       bool                     optional
  sync           string                   optional
  builds         []Build                  optional
  ir             string                   optional
  merge          Merge                    optional
```

---

## 7. 병합 뒤의 창 — 뒤 유닛이 들어오기 전

FD 흐름 7절의 표에 이 단계가 두 줄을 더한다 (「더함」).

```text
   칸 · 문장                   그 사이에 무슨 일이 생기나                          닫는 유닛
   effect · budget · discover  Mediator 가 받아 봉인한다.  노드는 모른다.  해 없음       finalize
   build · merge 단계          workspace.writes 를 요구한 계약은 422 (후보 없음).     bake
                              요구하지 않은 계약은 노드가 claim 하고
                              "run step has an empty argv" 로 실패한다
   Grammar 의 굽기 절 (더함)    계획을 짓는 agent 가 굽기를 배운다.  계약이 계획에게    bake
                              굽기를 맡기면 그 계획이 위 줄로 떨어진다.  맡기지 않으면
                              계획이 굽기를 지을 까닭이 없다
   runctl example 목록 (더함)  머리말이 "ready-to-run examples ... valid as printed"     -
                              다.  bake 는 Validate 를 지나지만 자리표시라 그대로는
                              안 돈다.  목록 문구는 이 유닛의 범위 밖이라 두었다
```

---

## 8. 뒤 유닛에 넘기는 것 (FD 흐름 8절 그대로)

```text
   step-phase    Claimed (internal/store/claim.go:66) 에 새 칸 일곱을 싣는다
   finalize      EffectOrDefault 로 RecordDiff 와 전체 훑기를 켜고 끈다 · Budgets 를 지킨다 ·
                 discover 의 상한과 진단
   lower-state   workspace.writes 광고 — 예시의 requires 와 lint 경고가 이 키를 쓴다
   bake          sync 와 builds 실행 · EnvIR · IR 대조 · ArtifactManifest 와 ArtifactMerged 쓰기 ·
                 MergeWait · 제자리에 쓰는 노드의 거절 · 7절 표의 틀린 문구 (empty argv)
```

**정본(enode-design)에 되돌려 올릴 것** — FD 흐름 9절 그대로다. 진행자가 올린다. 더할 것 하나 —
`ErrExitOnAgent` 의 새 문구가 ADR-053 1.2절이 인용한 옛 문구와 달라졌다.
