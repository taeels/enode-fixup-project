# `step-phase` — Functional Design 되물음

계획 파일(`step-phase-functional-design-plan.md`)의 답은 1 A · 2 A · 3 A · 4 A · **5 C** · 6 A · 7 A 다.
다른 답끼리는 서로 막지 않는다. **답 5 = C** (result 의 새 칸 여덟을 이 유닛에서 모두 타입으로 정한다)가
정하지 않은 자리 넷을 묻는다. 권장을 **A** 에 둔다.

C 가 여는 일 — 진단 · 보존 · build · merge 의 안쪽 모양을 이 유닛이 정한다. 그 모양을 채우는 유닛
(finalize · checkpoint · bake)은 이 타입을 쓰고, `internal/store/claim.go` 를 다시 만지지 않는다.

---

## 1. 실측 — C 를 받치는 사실

- **새 패키지 셋은 Mediator 가 링크하지 않는다.** Application Design 이 `internal/lower` ·
  `internal/merge` · `internal/scratch` 를 「표준 라이브러리와 x/sys 만 · 서로 임포트 안 함 · Mediator
  가 링크 안 함」으로 정했다 (`component-dependency.md:44` · `application-design.md:83`). 설계 스케치의 `scratch.Capture` ·
  `lower.BuildRecord` · `lower.Pinned` 를 `store.StepResult` 가 그대로 쓰면 Mediator 가 그 패키지를
  링크한다. 그래서 result 의 타입은 **Mediator 와 노드가 함께 가져오는 패키지**에 있어야 한다
- 노드(`cmd/enode`)와 Mediator(`cmd/mediator`)가 함께 링크하는 이 저장소의 패키지는 다섯이다 —
  `internal/contract` · `internal/environment` · `internal/schema` · `internal/build` ·
  `internal/transcript` (`go list -deps`). `internal/contract` 에는 이미 노드와 Mediator 사이의 전선
  어휘가 있다 — 광고 `Advert` · 정책 `Policy` (`advert.go`)
- 오늘 `StepResult.Environment` 가 `internal/environment` 의 `Record` 를 쓴다. 그 한 줄 때문에
  Mediator 가 environment 를 링크한다 — Reverse Engineering 이 적은 어긋남 ③ (두 코드 나무가 environment 에서 만난다)이다
- 설계 스케치가 있는 것 — `Capture` (`component-methods.md:284`) · `Diagnostics` (`:339`) ·
  `BuildManifest` (`:407`) · `BuildRecord` · `Pinned` (`:147` · `:152`). **`MergeResult` 는 스케치가
  없다** — 「ir · resumed · 셈」 한 줄뿐이다 (`:400`). `ChangesetDescriptor` 는 스케치가 있으나 그
  칸을 채우는 결과 adapter 가 순연됐다 (Units Generation Q3 = A)
- 정본의 result 는 평평하다 — `exited_at` · `finalize` · `upload` 가 가장 위의 칸이다 (`mediator-api.md:471`).
  ADR-075 의 `receipt` 묶음은 「의미상 자리」라고 스스로 적었다. 보존 칸의 이름은 정본의
  `checkpoint_capture` 를 쓴다 (ADR-076 §2)

---

## 2. 물음 넷

### 되물음 1 — 타입을 어느 패키지에 두나

A) **`internal/contract` 의 새 파일(`result.go`).** 노드와 Mediator 가 이미 함께 가져오고, 광고
   어휘가 이미 거기 있다. `store.StepResult` 가 이 타입을 쓴다. 새 패키지 셋(lower · merge · scratch)은
   표준 라이브러리만 쓰는 규칙을 지키려고 자기 타입을 그대로 두고, `internal/enode` 가 보고할 때 이
   타입으로 옮겨 담는다 (옮겨 담는 일은 그 칸을 채우는 유닛이 한다)

B) `internal/store` 에 두고, 노드 쪽은 같은 JSON 모양의 사본을 둔다. 두 벌이 되므로 모양이 같은지를
   시험이 확인한다

C) 새 패키지 `internal/wire` 를 만든다 (표준 라이브러리만). 계약 어휘와 결과 어휘가 섞이지 않는다.
   코드 경계 시험에 한 줄이 는다

D) Other (please describe after [Answer]: tag below)

[Answer]: A (「모두 권장대로」)

### 되물음 2 — changeset 칸

유닛 정의의 여덟(종료 시각 · 결과 확정 시각 · 예산 결과 · 원인 코드 · 진단 · 보존 상태 · build manifest ·
merge 결과)에는 changeset 이 없다. Application Design 의 result 목록에는 있다. 채울 결과 adapter 는 순연됐다.

A) **뺀다.** 순연을 푸는 회차가 그 adapter 와 함께 더한다. 지금 더하면 아무도 안 채우는 칸이 봉인에
   남는다

B) 모양까지 지금 넣는다 (설계 스케치 `base` · `digest` · `size` · `complete`)

C) Other (please describe after [Answer]: tag below)

[Answer]: A (「모두 권장대로」)

### 되물음 3 — 안쪽 모양

아래가 권장하는 모양이다. 스케치가 있는 것은 그대로 옮기고, 없는 것은 정본에서 뽑았다. 이름은 JSON 이다.

```text
   result 의 새 칸 (가장 위 · 평평하게)
     exited_at            시각 (노드 시계)
     finalized_at         시각 (노드 시계)
     finalize             ok | timeout | error
     upload               ok | timeout | error
     reason               finalize_timeout | upload_timeout | merge_wait_timeout | bake_in_progress
     diagnostics          Diagnostics
     checkpoint_capture   CheckpointCapture
     build                BuildManifest        build 단계만
     merge                MergeResult          merge 단계만

   Diagnostics           설계 스케치 (Q5) + 명시 훑기의 두 칸
     missing               []string   out 에 적었는데 안 나온 이름
     collect               [{name, why}]   collect 가 못 걷은 이유 (오늘의 HarvestNote)
     changes               measured | not_measured | partial   변경 목록을 실제로 확인했나
     effect                read | edit | build | prepare   이 단계가 따른 effect
     discovered            []string   discover 가 찾은 경로 (진단일 뿐 · produced 가 아니다)
     discovery_limit       visits | time | memory | size   상한에 닿았으면 그 이름 · 아니면 없음

   CheckpointCapture     설계 스케치 그대로 (ADR-076 §2)
     state                 not_requested | unsupported | rejected | captured | failed
     reason · id · scope · guarantee · node · expires_at

   BuildManifest         설계 스케치 그대로 (ADR-077 §5 의 metadata 중 build 단계가 아는 것)
     sync                  {name, command, started_at, finished_at, exit_code}
     builds                [ 같은 모양 ]
     head                  sync 뒤 manifest HEAD
     ir                    HEAD 에 정확히 붙은 태그 · 없으면 null
     pinned                {file, sha256} · manifest 가 없는 저장소면 null

   MergeResult           스케치가 없어 ADR-077 에서 뽑았다
     ir                    합친 뒤 lower 의 ir · 없으면 null
     previous_ir           합치기 전 lower 의 ir · 없으면 null
     merged_at             합치기가 끝난 시각 (노드 시계)
     resumed               이 합치기가 앞서 끊긴 합치기를 이어 끝냈나 (ADR-077 §7)
     ops                   {replaced, created, dirs, opaque, whiteouts, trashed, attrs}
                           합치기 연산의 셈 — ADR-077 §12 실측의 일곱 갈래
```

A) **위 모양 그대로.** 뒤 유닛은 칸을 **더할** 수 있다 (옛 Record 를 읽는 쪽이 안 깨진다). 있는 칸의
   이름이나 뜻을 바꾸려면 이 유닛의 설계 문서를 함께 고친다

B) 위 모양에서 명시 훑기의 두 칸(`discovered` · `discovery_limit`)을 뺀다 — 훑기 결과의 자리는
   finalize 유닛이 정한다

C) 위 모양에서 `MergeResult` 의 `ops` 를 뺀다 — 셈은 merge 단계 로그에만 남긴다

D) Other (please describe after [Answer]: tag below)

[Answer]: A (「모두 권장대로」)

### 되물음 4 — 값을 검사하나

A) **검사하지 않는다.** 모르는 `finalize` · `upload` · `reason` · `state` 값도 그대로 봉인한다. 타입은
   모양을 정할 뿐 400 의 근거가 아니다. 노드는 result 를 임대가 죽을 때까지 재시도하므로 400 은 그
   보고를 잃게 한다 (계획 1.2). 어휘 밖의 값은 로그에 경고 한 줄을 남긴다

B) 어휘 밖의 값이면 400 을 낸다

C) Other (please describe after [Answer]: tag below)

[Answer]: A (「모두 권장대로」)
