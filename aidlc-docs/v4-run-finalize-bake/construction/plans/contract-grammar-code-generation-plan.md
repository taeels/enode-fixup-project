# `contract-grammar` — Code Generation 계획

**유닛** `contract-grammar` (한 줄 순서의 첫째) · **브랜치** `unit/contract-grammar` ·
**기준** `2949bd0` (Functional Design 커밋) · **맡는 조각** 없음 · **병합 조건** 코드 검사

**이 계획이 이 유닛 Code Generation 의 기준이다.** 여기 없는 것은 짓지 않는다. 설계는
`construction/contract-grammar/functional-design/` 의 셋이다 — 타입은 `domain-entities.md`,
규칙과 거절 문구는 `business-rules.md`, 순서와 흐름은 `business-logic-model.md`. 아래의 FD 는 Functional Design 의 줄임이다 —
「FD 규칙 3절」은 `business-rules.md` 3절, 「FD 흐름 5절」은 `business-logic-model.md` 5절을 뜻한다.

- **작성 시각**: 2026-09-25T05:03:44Z 이후 (Functional Design 승인 뒤)
- **입력**: Functional Design 산출물 셋 · `unit-of-work.md` 1절 · `unit-of-work-file-matrix.md` 1.1절 ·
  `unit-of-work-story-map.md`

---

## 1. 유닛 맥락

**하는 일** — 계약 문법만 늘린다. 노드는 새 칸을 아직 모르고(Claimed 가 싣지 않는다) Mediator 는
받아서 봉인할 뿐이다. 뜻은 뒤 유닛이 채운다 (FD 흐름 7절의 창).

```text
   Step 의 새 칸 일곱     effect · budget · discover · sync · builds · ir · merge
   단계 종류              넷에서 여섯.  build (판별 칸 sync 또는 builds) · merge (판별 칸 merge)
   400 규칙               FD 규칙 1 ~ 8절.  거절 문구는 그 표의 영어 그대로
   기본값 메서드 셋        EffectOrDefault · Budgets · MergeWait
   계약 작성 도구          Grammar · PlanShape · runctl example bake · runctl lint
```

**받치는 스토리** — US-8 (예산) · US-9 (effect 로 diff 를 계속 받는 길) · US-15 (merge 대기 상한) ·
US-19 (구울 IR). 이 유닛이 **완성하는** 스토리는 0 이다 (`unit-of-work-story-map.md` 2절). 스토리의
체크는 그 스토리를 완성하는 유닛이 한다.

**앞 유닛** — 없다. **뒤 유닛에 넘기는 것** — FD 흐름 8절 그대로다.

```text
   step-phase    Claimed (internal/store/claim.go:66) 에 새 칸 일곱을 싣는다
   finalize      effect 에 따른 diff 와 전체 훑기 · 예산 둘 · discover 의 상한
   lower-state   workspace.writes 광고.  예시의 requires 와 lint 경고가 이 키를 쓴다
   bake          sync 와 builds 실행 · ENODE_IR · IR 대조 · manifest 와 merged 쓰기 · merge.wait
```

**새로 내보내는 이름** (`internal/contract`)

```text
   타입     Effect · Budget · Build · Merge
   값       EffectRead · EffectEdit · EffectBuild · EffectPrepare · KindBuild · KindMerge
   상수     DefaultFinalizeBudget (1분) · MinFinalizeBudget (= 기본값.  늘리기만 한다) ·
            DefaultUploadBudget (3분) · DefaultMergeWait (4시간) ·
            ArtifactManifest ("manifest") · ArtifactMerged ("merged") · EnvIR ("ENODE_IR")
   메서드   Step.EffectOrDefault · Step.Budgets · Step.MergeWait
```

**자료** — DB 스키마 변경 0. `steps.kind` 칸에 `"build"` · `"merge"` 값이 새로 들어갈 수 있으나 그
칸에는 값 제약이 없다 (`internal/store/store.go:396` · `expand.go:306` 이 `Kind().String()` 을 그대로
적는다).

**경계** — `internal/contract` 에 새 import 0 (표준 라이브러리 `time` · `strings` 는 이미 쓴다).
Mediator 와 노드가 이미 이 패키지를 가져오므로 코드 경계 시험의 금지 표가 바뀌지 않는다.

---

## 2. 고치는 파일

| 파일 | 새 · 고침 | 행렬 | 무엇 |
|---|---|---|---|
| `internal/contract/contract.go` | 고침 | 있음 | Step 칸 일곱 · StepKind 둘 · `Kind()` · `Validate` 에 호출을 거는 자리 · `ErrExitOnAgent` 문구 |
| `internal/contract/effect.go` | 새 | 밖 | `Effect` · `Budget` · 상수 · `EffectOrDefault` · `Budgets` · effect · budget · discover 규칙 |
| `internal/contract/bake.go` | 새 | 밖 | `Build` · `Merge` · 상수 · `MergeWait` · build · merge 칸 규칙 · ir 문자 규칙 · 굽기 모양 · 계획이 짓는 굽기 · 판정 조건 |
| `internal/contract/checkplan.go` | 고침 | 밖 | 그루터기를 굽기 모양 규칙에서 빼는 자리 (3절) |
| `internal/contract/grammar.go` | 고침 | 있음 | 종류 표 여섯 · effect · budget · discover · 굽기 절 · 판정 조건 절 |
| `internal/contract/planshape.go` | 고침 | 밖 | 굽기 모양 예시 하나 (FD 흐름 5절) |
| `internal/contract/examples/bake.json` | 새 | 있음 | FD 흐름 6.1절의 JSON 그대로 |
| `cmd/runctl/shape.go` | 고침 | 있음 | `suggestCondition` 의 build · merge 갈래 · 경고 둘 |
| `internal/contract/contract_test.go` | 고침 | 시험 | `TestStepKind` 에 여섯 종류 · 기존 문구를 부르는 줄 |
| `internal/contract/effect_test.go` · `bake_test.go` | 새 | 시험 | 규칙마다 표 한 줄 |
| `internal/contract/grammar_test.go` · `checkplan_test.go` | 고침 | 시험 | 문법 문장마다 거절 한 줄 · PlanShape 의 굽기 모양 · 그루터기 |
| `cmd/runctl/shape_test.go` | 고침 | 시험 | 조건 제안 · 경고 둘 |

**행렬 밖 파일 넷을 왜 만지나** (`unit-of-work-file-matrix.md` 머리말 — 계획에 적는다)

- `effect.go` · `bake.go` — 같은 패키지 안에서 주제별로 나눈다. `contract.go` 는 이미 1,799 줄이고
  `Validate` 한 함수가 600 줄이다. 새 규칙을 거기 직접 넣으면 한 함수가 800 줄을 넘는다. `Validate`
  에는 호출 네 줄만 더하고 규칙 본문은 새 파일에 둔다. 이 패키지가 `checkplan.go` · `advert.go` 를
  따로 둔 방식과 같다
- `planshape.go` — FD 흐름 5절이 굽기 모양 예시를 더했다. 행렬을 쓸 때는 없던 결정이다
- `checkplan.go` — 이 계획이 찾은 자리다 (3절)

---

## 3. 이 계획이 찾은 것 — 계획 훅의 그루터기가 굽기 모양 규칙에 걸린다

**무엇이 걸리나.** `CheckPlan` (`internal/contract/checkplan.go:42`) 은 계획을 짓는 agent 가 끝나기
전에 Stop 훅에서 계획을 미리 본다. 계획의 단계가 계획 밖 단계(보통 부모의 승인 ask)를 `needs` 로
가리키면, 그 이름을 **run 단계 그루터기**(`Run: ["true"]`)로 앞에 채워 참조만 성립시키고 같은
`Validate` 를 부른다 (`stubsFor` · `:89`).

계획이 굽기를 지으면서 `needs: ["approve"]` 를 적은 경우를 따라가면 이렇다.

```text
   계획                       [build (needs: approve), merge]
   훅이 만드는 계약            [approve (그루터기 · run 단계), build, merge]
   FD 규칙 5절의 마지막 줄     build 앞에는 계획 단계만 올 수 있다
                              -> "step "approve" comes before the bake but is not a planning step"
   실제 계약 (applyExpands)    [plan (expands), approve (ask · adopts), build, merge]  -> 통과
```

훅이 **맞는 계획을 틀린 이유로 거절한다.** 그 문구가 그대로 모델에게 되먹여지므로 모델은 approve 가
계획 단계가 아니라는 거짓을 보고 계획을 고친다. `needs` 를 안 적은 계획은 그루터기가 없어 걸리지
않는다 — 그래서 FD 의 예시 흐름에서는 안 보였다.

**이 계획이 고르는 것** — 그루터기를 모양 규칙의 「build 앞」 검사에서 뺀다.

```text
   contract.go    Validate() 는 c.validate(nil) 을 부른다.  공개 표면은 그대로다
   checkplan.go   CheckPlan 은 c.validate(그루터기 이름들) 을 부른다
   bake.go        「build 앞에는 계획 단계만」 검사가 그루터기 이름을 건너뛴다.
                  나머지 규칙(개수 · 순서 · needs · 역할 · 판정 조건)은 그대로 건다
```

오늘의 훅이 이미 정한 선과 같다 — 「그루터기는 참조만 성립시키고, 실존 여부는 `applyExpands` 가
본다」(`checkplan.go:78` 주석). 그루터기의 **종류**도 같은 부류다. 권위 있는 검사는 `applyExpands`
가 늘어난 계약 전체에 `Validate()` 를 다시 부를 때 그대로 걸린다. FD 의 규칙은 한 줄도 바뀌지 않는다.

**고르지 않은 것** — 그루터기를 ask 단계로 만든다. ask 는 `prompt` · `out` · `schema` 가 있어야
하고 `adopts` 로 계획 단계를 지목해야 계획 단계로 인정되므로, 그루터기를 진짜처럼 꾸미는 코드가
늘어난다. 꾸민 그루터기는 다른 규칙(ask 폼 · adopt 짝)에 새로 걸릴 수 있다.

---

## 4. 단계 — 열다섯

규칙 순서는 FD 흐름 1절을 따른다 — 종류 판별 -> 종류마다 받는 칸 -> 값 -> (오늘의 덩어리들) ->
굽기 모양 -> 계획이 짓는 굽기 -> 판정 조건. 오류는 처음 하나에서 돌아온다 (오늘의 `Validate` 와 같다).

### Step 1 — 타입과 상수

- [x] `effect.go` 를 만든다 — `Effect` 와 값 넷 · `Budget` · `DefaultFinalizeBudget` ·
      `MinFinalizeBudget` · `DefaultUploadBudget`. 주석은 한국어 (CONVENTIONS 2.2)
- [x] `bake.go` 를 만든다 — `Build` · `Merge` · `DefaultMergeWait` · `ArtifactManifest` ·
      `ArtifactMerged` · `EnvIR`
- [x] `contract.go` 의 `Step` 에 칸 일곱을 더한다 (`:443` 구조체의 끝). 모두 `omitempty` —
      안 적은 계약의 JSON 이 오늘과 같다. 칸마다 무엇이고 누가 읽나를 한 덩어리 주석으로
      (FD 엔티티 1절)

### Step 2 — 종류 판별

- [x] `KindBuild` · `KindMerge` 를 상수 블록 끝(`KindAcquire` 뒤)에 더한다. 기존 번호가 안 바뀐다
- [x] `String()` 이 `"build"` · `"merge"` 를 낸다
- [x] `Kind()` (`:798`) — build 판별은 `Sync != "" || Builds != nil` (둘 중 하나라도),
      merge 판별은 `Merge != nil`. 거절 문구 둘을 FD 규칙 1절로 바꾼다

### Step 3 — 종류마다 받는 칸과 값 (FD 규칙 2 · 3 · 4 · 7절)

- [x] `Validate` 의 종류 판별 훑기(`:1066`)에서 `Kind()` 바로 뒤, 역할 검사(`:1080`) **앞**에
      `checkStepFields(s, k)` 를 부른다. 앞이어야 uses 없는 build 가 「undeclared role ""」 대신
      「a build step needs uses」를 받는다
- [x] effect — 모르는 값 · 종류마다 허용 값 · build 는 prepare 필수
- [x] budget — 받는 종류 · 기간 형식 · finalize 1분 아래 · upload 0 이하. merge 문구의 꼬리
      (`; merge.wait sets how long it waits`)
- [x] discover — run · agent 만
- [x] ir — build 가 아닌 단계에 있으면 거절
- [x] build — uses · sync (공백을 걷은 뒤) · builds 개수 · 이름 규칙(1 ~ 64 자 `[a-z0-9-]`) · 이름 겹침 ·
      빈 명령 · ir 필수 · ir 문자 규칙 여섯 줄 (FD 규칙 3.1) · workspace · out · 그 밖의 칸
- [x] merge — uses · `merge.wait` 형식과 0 이하 · 그 밖의 칸
- [x] 「그 밖의 칸」은 Step 의 칸을 (JSON 이름, 적혔나) 쌍의 목록으로 훑는다. 칸이 늘면 목록도 느는
      것을 시험이 확인한다 (Step 7)

### Step 4 — 굽기 모양과 계획이 짓는 굽기 (FD 규칙 5 · 6절)

- [x] `Validate()` 의 본문을 `validate(stubs map[string]bool)` 로 옮기고 `Validate()` 는
      `validate(nil)` 을 부른다 (3절)
- [x] 모양 검사를 adopt 덩어리(`:1546`) 뒤, success_when 덩어리(`:1573`) 앞에 건다 — needs ·
      dispatch · adopt 를 오늘의 덩어리가 먼저 확인한 뒤다 (FD 흐름 1절)
- [x] 모양 — build 개수 · merge 개수 · 짝 없음 둘 · merge 가 마지막 · 둘 사이의 단계 ·
      merge 의 needs (`NeedsOf` 로 채운 뒤) · 같은 역할 · build 앞은 계획 단계만 (그루터기 빼고)
- [x] 계획이 짓는 굽기 — 굽기 계약에 expands 단계가 있으면 넷: yolo 아님 · 지목한 ask ·
      adopt_when 이나 dispatch · build 의 조상에 그 ask (`ancestors` 를 그대로 쓴다)

### Step 5 — 판정 조건 (FD 규칙 8절)

- [x] `ErrExitOnAgent` 의 문구를 `exit_code condition is allowed only on a run step` 으로 바꾼다.
      이름은 그대로 (`errors.Is` 로 부르는 쪽이 있다). loop.until 자리(`:1340`)도 같은 값을 쓴다
- [x] success_when 덩어리에서 build 는 정확히 `produced ["manifest"]`, merge 는 정확히
      `produced ["merged"]` 만 받는다. exit_code · changed · within_attempts 가 있거나 이름이
      다르면 거절

### Step 6 — 기본값 메서드

- [x] `EffectOrDefault` — 적힌 값, 없으면 run -> build · agent -> edit, 그 밖은 `""`
- [x] `Budgets` — 없는 값은 기본값. 파싱 오류는 다시 다루지 않는다 (Validate 를 지난 계약만)
- [x] `MergeWait` — 없으면 4시간

### Step 7 — 계약 패키지의 단위 시험

- [x] `effect_test.go` · `bake_test.go` — FD 규칙 표의 거절 문구마다 한 줄. 문구의 핵심 구절이
      오류에 들어 있는지 본다
- [x] 경계값 — finalize 정확히 `1m` 받음 · `59s` 거절 · upload `1ns` 받음 · `0s` 거절 ·
      `"budget": {}` 받음 · ir 128 자 받음 · 129 자 거절 · 이름 64 자 받음 · 65 자 거절
- [x] ir 문자 규칙 — 여섯 줄마다 거절 하나와 받는 값 몇 (`yocto-5.0.1` · `release/v1.2` ·
      `your-ir-tag`)
- [x] 받는 계약 — 사람이 쓴 굽기 · 계획이 지은 뒤의 굽기 (plan · approve · build · merge)
- [x] 안 적은 계약이 오늘과 같은 JSON 인지 — 기존 예시를 풀었다 다시 묶어 새 칸 이름이 안 나온다
- [x] 「그 밖의 칸」 목록이 Step 의 JSON 칸을 빠짐없이 덮는지 — reflect 로 칸 이름을 세어 대조한다.
      칸을 더하고 목록을 잊으면 깨진다
- [x] 기본값 메서드 셋
- [x] `contract_test.go` — `TestStepKind` 에 build · merge · 판별 칸 겹침 줄. `ErrExitOnAgent`
      줄은 `errors.Is` 라 그대로 지난다

### Step 8 — 계획 훅의 그루터기 (3절)

- [x] `CheckPlan` 이 그루터기 이름을 `validate` 에 넘긴다
- [x] `checkplan_test.go` — `needs: ["approve"]` 를 적은 굽기 계획이 훅을 지난다 · 같은 계획을
      실제 모양(plan · approve · build · merge)에 붙이면 `Validate()` 도 지난다 · 그루터기가 아닌
      run 단계가 build 앞에 있으면 여전히 거절된다

### Step 9 — Grammar 와 PlanShape (FD 흐름 5절 · FD 규칙 10절)

- [x] `Grammar` — 「There are four kinds of step」 절을 여섯으로 · effect · budget · discover 절 ·
      굽기 절 · 「success_when conditions differ by step kind」 절에 build · merge 줄. 문장은 FD
      흐름 5절의 초안을 다듬는다. 영어 · 장식 문자 0
- [x] `grammar_test.go` 의 표에 문장마다 한 줄 — `mustSay` 가 Grammar 에 있고, 그 규칙을 어긴 계약이
      실제로 거절되는지. 굽기 절의 문장 아홉 · effect · budget · discover 넷
- [x] `PlanShape` 에 굽기 모양 하나. 시험 — PlanShape 의 그 절에서 JSON 객체를 뽑아(사본을 두지
      않는다) 역할 하나와 판정 조건을 붙인 계약이 `Validate()` 를 지난다

### Step 10 — 예시 `bake.json`

- [x] `internal/contract/examples/bake.json` — FD 흐름 6.1절 그대로. 사내 명령 · 이름 0.
      명령은 `<...>` 자리표시 · ir 은 `your-ir-tag` · 구성 이름 `config-a` · `config-b`
- [x] 기존 시험 셋이 고치지 않고 지나는지 본다 — `TestExamples_ParseAndValidate` ·
      `TestExamples_EveryStepHasASuccessCondition` · `TestExamples_LintClean` (Step 11 뒤)

### Step 11 — `runctl lint` (FD 규칙 9절 · FD 흐름 6.2절)

- [x] `suggestCondition` — build · merge 갈래를 run 갈래 앞에. `Kind()` 로 판별한다
- [x] 경고 하나 — build 단계의 역할(requires 와 acquire.want 에서 찾는다)이 `workspace.writes`
      를 `isolated` 로 요구하지 않는다. 문구는 FD 규칙 9절
- [x] 경고 둘 — success_when 이 계획이 약속한 이름에 `produced ["manifest"]` 나 `["merged"]` 를
      건다. 그 약속을 한 expands 단계의 설정이 FD 규칙 6절의 넷 가운데 제출 때 볼 수 있는 셋을
      안 지키면 경고. 어긴 것을 문구의 `%s` 자리에 적는다
- [x] `shape_test.go` — 제안 둘 · 경고 둘이 나오는 계약과 안 나오는 계약 · 예시 전부 경고 0

### Step 12 — `runctl schema`

- [x] 코드 변경 0 예상. `runctl schema steps` 가 새 칸 일곱을 내는지 눈으로 확인하고
      `code-summary.md` 에 출력을 적는다. `Budget` · `Build` · `Merge` 의 안쪽 칸은 이 명령이 안
      펼친다 — 예시와 Grammar 가 보여 준다 (오늘 `Ask` · `Loop` 와 같은 처지)

### Step 13 — 코드 검사 (`unit-of-work.md` 0절 · CI 와 같은 명령)

- [x] `gofmt -l .` 빈 출력 · `go vet ./...` · `go build ./...`
- [x] `eval "$(scripts/testdb.sh)"` 뒤 `go test ./... -count=1` — 실패 0 · 스킵 0
- [x] `go test ./... -count=1 -coverpkg=./... -coverprofile=...` — 패키지마다 80% 이상.
      `internal/contract` 는 자기 시험만으로 오늘 80.6% 라 여유가 얇다. 새 코드의 줄마다 표 시험이
      닿게 한다
- [x] 크로스 빌드 셋 — `GOOS=windows GOARCH=amd64` · `GOOS=linux GOARCH=arm GOARM=7` ·
      `GOOS=darwin GOARCH=arm64`
- [x] `enodectl.exe` 의 심볼 상한 (crypto/tls 10 · net/http 50)
- [x] U+2605 0 (`grep -rlIP '\x{2605}'`) · `go run ./scripts/glyphscan.go` (Go 문자열의 장식 문자)
- [x] 코드 경계 시험이 지나는지 (`go test ./...` 안에 있다)
- [x] `git status` 로 행렬과 2절 밖의 파일이 없는지. `cmd/enodectl/probe.lock` 이 바뀌면 되돌린다
      (R/E 가 적은 알려진 부채)

### Step 14 — 코드 요약 문서

- [x] `construction/contract-grammar/code/code-summary.md` — 고친 파일과 새 파일 · 새 이름 ·
      규칙이 어느 함수에 있나 · 3절의 결정 · 코드 검사 결과(숫자) · `runctl schema steps` 출력 ·
      뒤 유닛에 넘기는 것 · 정본에 되돌려 올릴 것 (FD 흐름 9절 · 진행자가 한다)
- [x] 표기 검사 — `enode-design/scripts/emphasis-check.py` 를 새 문서와 이 계획에 돌린다

### Step 15 — 상태 · 감사 · 커밋

- [x] 이 계획의 체크박스를 채운다 (단계를 끝낸 그 자리에서)
- [x] `aidlc-state.md` 의 U1 절 · `audit.md` 에 완료와 승인 요청을 적는다
- [x] 한 커밋 — 코드 · 시험 · 이 계획 · code-summary · 상태 · 감사 (CONVENTIONS 3.4). 승인 뒤에 넣었다 (CONVENTIONS 3.3)

---

## 5. 이 단계가 하지 않는 것

```text
   Claimed 의 새 칸                step-phase 의 파일 (internal/store/claim.go)
   노드가 칸을 읽고 하는 일          finalize · bake
   produce 칸 (결과 adapter)        순연 (Units Generation Q3 = A)
   정본(enode-design) 되돌림        진행자가 올린다 (FD 흐름 9절).  이 저장소의 서브모듈을 안 고친다
   PR 과 병합                      코드 검사가 초록인 뒤.  외부로 나가는 일이라 올리기 전에 묻는다
   파일 행렬 문서 고침              2절이 행렬 밖 파일과 이유를 적는다 (행렬 머리말의 절차)
```

---

## 6. 확장 준수 — Code Generation

| 확장 | 판정 | 근거 |
|---|---|---|
| security-baseline | N/A (꺼짐) | Requirements 질문 4 = B. 팩 보안 표의 「계약이 노드 설정이나 host 경로를 지정하지 못한다」는 build · merge 의 `workspace` 거절과 「그 밖의 칸」 거절로 닫힌다 (Step 3) |
| resiliency-baseline | N/A (꺼짐) | Requirements 질문 5 = B |
| property-based-testing | N/A (꺼짐) | Requirements 질문 6 = X. 규칙마다 표 시험 한 줄 (Step 7) |
