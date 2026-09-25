# `contract-grammar` — 도메인 엔티티

계약 문법에 더하는 칸과 타입이다. 입력은 계획의 답 일곱과 되물음 셋
(`construction/plans/contract-grammar-functional-design-plan.md` 2절 ·
`...-clarification-questions.md`)이다. 규칙은 `business-rules.md`, 흐름은
`business-logic-model.md` 에 있다.

```text
   답     1 A  굽기 판정은 produced          2 A  effect · budget 의 엄격한 표
          3 A  굽기 계약은 build · merge 로 끝난다   4 A  ir · ENODE_IR · 좁은 문자
          5 A  "discover": true              6 B  계획도 굽기 단계를 짓는다
          7 C  굽기 예시의 명령은 자리표시
   되물음  1 A  굽기 앞에는 계획 단계만        2 A  굽기를 지은 계획은 사람이 승인
          3 A  예시의 ir 과 이름은 규칙을 지키는 자리표시
```

---

## 1. `Step` 에 더하는 칸

모두 `omitempty` 다. 안 적은 계약은 오늘과 같은 JSON 이다.

| JSON 칸 | Go 타입 | 받는 종류 | 뜻 |
|---|---|---|---|
| `effect` | `Effect` | run · agent · build | 작업 성격. 노드가 무엇을 결과로 거두나를 정한다 (ADR-075 §5 — actor 와 effect 를 나누는 결정) |
| `budget` | `*Budget` | run · agent · build | 명령이 끝난 뒤 결과를 확정하고 올리는 구간의 시간 상한 둘 (FR-3) |
| `discover` | `bool` | run · agent | 명시로 켜는 훑기. 결과는 진단이다 (FR-1 · ADR-075 §8) |
| `sync` | `string` | build | 소스를 구울 IR 로 맞추는 명령 한 줄 (ADR-077 §2) |
| `builds` | `[]Build` | build | 이름 붙은 빌드 명령들. 적힌 순서대로 돈다 |
| `ir` | `string` | build | 구울 IR 태그의 정확한 값. 필수 (계획 2.3 · 답 4) |
| `merge` | `*Merge` | merge | 합치기 단계의 표시이자 설정. 명령이 아니라 노드의 내장 단계다 |

`sync` 와 `builds[].command` 는 제품이 해석하지 않는 문자열이다. 내용은 계약을 쓰는 쪽의
것이고, 노드가 격리 runtime 안에서 셸로 돌린다 (어떻게 돌리는지는 bake 유닛).

---

## 2. 새 타입

```go
// Effect 는 단계의 작업 성격이다 (ADR-075 §5).
type Effect string

const (
	EffectRead    Effect = "read"    // 읽기만.  이번 회차에서는 build 와 같이 diff 와 전체 훑기를 안 한다
	EffectEdit    Effect = "edit"    // source 를 고친다.  diff 를 거둔다
	EffectBuild   Effect = "build"   // 빌드와 시험.  diff 를 안 거둔다
	EffectPrepare Effect = "prepare" // 굽기의 build 단계 전용.  upper 를 합치기로 넘긴다
)

// Budget 은 명령이 끝난 뒤 구간의 시간 상한 둘이다 (FR-3).  Go duration 문자열.
type Budget struct {
	Finalize string `json:"finalize,omitempty"` // 기본 1분.  1분 아래는 400 (늘리기만 한다)
	Upload   string `json:"upload,omitempty"`   // 기본 3분.  0 이하는 400
}

// Build 는 굽기 build 단계의 구성 하나다.
type Build struct {
	Name    string `json:"name"`    // [a-z0-9-] · 1 ~ 64 자 · 계약 안에서 겹치지 않는다
	Command string `json:"command"` // 빈 문자열이 아니다
}

// Merge 는 굽기 merge 단계의 설정이다.  "merge": {} 도 merge 단계다.
type Merge struct {
	Wait string `json:"wait,omitempty"` // 형제를 기다리는 상한.  기본 4시간.  0 이하는 400
}
```

`discover` 는 객체가 아니라 `bool` 이다 (답 5 = A). 상한은 노드가 정하므로 계약에 적을 값이
없다. 나중에 계약이 상한을 낮출 필요가 생기면 그때 객체로 바꾼다 — 그때까지 `true` 한 값이다.

---

## 3. 단계의 종류 — 넷에서 여섯

종류는 **판별 칸이 정확히 하나 있는가**로 정한다 (`Step.Kind()` · ADR-019 의 방식 그대로).

| 종류 | 판별 칸 | 누가 수행하나 | `uses` |
|---|---|---|---|
| `agent` | `agent` | 노드 — 하네스 | 필수 |
| `run` | `run` | 노드 — argv | 필수 |
| `ask` | `ask` | 사람 | 없음 |
| `acquire` | `acquire` | Mediator | 없음 |
| `build` (새) | `sync` 또는 `builds` | 노드 — 굽기의 짓기 | 필수 |
| `merge` (새) | `merge` | 노드 — 굽기의 합치기 | 필수 |

- `KindBuild` · `KindMerge` 는 상수 블록의 끝(`KindAcquire` 뒤)에 더한다. 기존 값의 번호가
  안 바뀐다. `String()` 은 `"build"` · `"merge"` 를 내고, 그 값이 `steps.kind` 칸에 그대로
  저장된다 (`internal/store/store.go:396` · 스키마에 값 제약이 없다)
- build 는 `sync` 와 `builds` 둘 다 있어야 하지만, 판별은 **어느 하나라도** 있으면 build 로
  본다. 그래야 하나를 빠뜨린 계약이 「종류를 모른다」가 아니라 「build 단계에 sync 가 없다」로
  거절된다
- `ir` · `effect` · `budget` · `discover` 는 판별 칸이 아니다

---

## 4. 제품이 정한 이름과 값

```text
   산출물 이름     build 단계    manifest   모든 명령이 0 으로 끝나고 IR 이 맞았을 때만 낸다
                  merge 단계    merged     합쳤을 때만 낸다
                  계약은 build · merge 에 out 을 적지 않는다 (답 1 = A)

   환경 변수      ENODE_IR      노드가 sync 와 builds 명령에 ir 값을 넘기는 이름 (답 4 = A)

   기본값         finalize 예산   1분      MinFinalizeBudget 과 같다 — 늘리기만 한다
                  upload 예산     3분
                  merge 대기      4시간
```

상수는 `internal/contract` 에 둔다. 노드(`internal/enode`)가 이미 이 패키지를 가져오므로
(`advertise.go` · `claim.go` 등) finalize 와 bake 유닛이 같은 값을 읽는다. 기본값을 두 곳에
적지 않는다.

---

## 5. 메서드

```go
// EffectOrDefault 는 적힌 effect 를, 없으면 종류의 기본값을 준다.
//   run -> build · agent -> edit · build -> prepare 가 적혀 있어야 하므로 그 값
//   merge · ask · acquire -> "" (effect 가 없는 종류)
func (s Step) EffectOrDefault() Effect

// Budgets 는 두 예산을 기본값을 채워 준다.  Validate 를 지난 계약에서만 부른다.
func (s Step) Budgets() (finalize, upload time.Duration)

// MergeWait 는 merge 대기 상한을 기본값을 채워 준다.
func (s Step) MergeWait() time.Duration
```

기본값을 메서드가 채우는 것은 `Condition.Want()` 의 `min_count` 와 같은 이유다 — 읽는 쪽이
여럿이므로 각자 기본값을 알게 두면 언젠가 한 곳이 어긋난다.

---

## 6. 굽기 계약의 모양 (답 3 = A · 되물음 1 = A)

```text
   사람이 처음부터 쓴 굽기            계획이 지은 굽기

   steps                             steps (제출 때)
     build   (uses: baker)             plan     agent · expands · produces [build, merge]
     merge   (uses: baker,             approve  ask · adopts: plan · adopt_when: approve
              needs: [build])        steps (계획이 붙은 뒤)
                                       plan
                                       approve
                                       build    (uses: baker)      <- 계획이 지었다
                                       merge    (uses: baker, needs: [build])
```

- 굽기 계약은 **build 하나와 merge 하나로 끝난다.** merge 가 마지막이고 build 가 바로 앞이다
- build 앞에 올 수 있는 것은 **계획 단계**뿐이다 — 계획을 짓는 agent 단계(`expands: true`)와
  그 계획을 승인하는 ask 단계(`adopts`)
- 계획이 지은 단계는 언제나 계약의 끝에 붙는다 (`internal/store/expand.go` — 기존 단계 뒤에
  이어 붙인다). 그래서 계획 단계가 앞에 있고 굽기가 뒤에 오는 모양이 저절로 나온다
- 계획이 지은 굽기는 **사람이 승인한 뒤에만 돈다** — ask 가 그 계획을 지목하고(`adopts`),
  거절하면 계획을 물리며(`adopt_when`), build 가 그 ask 를 기다린다(`needs`). 규칙은
  `business-rules.md` 6절

---

## 7. 이 유닛이 안 만드는 것

```text
   produce 칸 (결과 adapter)       순연 (Units Generation Q3 = A)
   Claimed 의 새 칸                노드에 계약 칸을 실어 보내는 자리.  internal/store/claim.go 는
                                   step-phase 유닛의 파일이다 (business-logic-model.md 7절)
   노드가 칸을 읽고 하는 일          finalize · bake 유닛
```
