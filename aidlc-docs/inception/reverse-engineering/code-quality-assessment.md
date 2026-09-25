# 코드 품질 평가 — 오늘의 코드

**2026-09-23 전면 재측정.** 기준 커밋 `195a5d0`. 커버리지와 스킵은 이 기계에서
실제로 돌려 잰 값이다.

```text
   측정 명령    go test ./... -count=1 -coverpkg=./... -coverprofile=... -json
   환경         ENODE_TEST_DATABASE_URL 을 세우고 (scripts/testdb.sh)
                .github/ci-stubs 를 PATH 앞에 둔다.  linux/amd64
   결과         exit 0 · 패키지 21 (프로파일에 20) · 실패 0 · 스킵 0 ·
                전체 8,963/10,509 = 85.3% · 벽시계 1분 20초
   집계         CI 의 커버리지 스텝과 같은 awk (블록 키로 병합 · count 최댓값)
```

이것은 `.coverage-contract.yml` 이 못 박은 바로 그 명령이다. `integration` 태그는
그 명령에 없다 — 그래서 아래 표에도 없다.

---

## Test Coverage

### 게이트는 패키지별 80% 바닥이다

저장소 전체 합계가 아니라 **패키지마다 개별로** 넘어야 한다. 차단이다.

### 오늘의 측정값 — 스물 전부 통과

| 패키지 | 덮은 것/문장 | 비율 | 2026-09-15 |
|---|---|---|---|
| `internal/build` | 16/20 | 80.0% | 80.0% |
| `internal/enode` | 2653/3256 | **81.5%** | 87.0% |
| `internal/environment` | 480/584 | 82.2% | 신규 |
| `internal/api` | 877/1065 | 82.3% | 82.1% |
| `cmd/enodectl` | 187/227 | 82.4% | 83.5% |
| `internal/store` | 1478/1792 | 82.5% | 82.6% |
| `cmd/enode` | 186/224 | 83.0% | 97.2% |
| `internal/record` | 391/463 | 84.4% | 82.4% |
| `internal/config` | 103/121 | 85.1% | 85.1% |
| `internal/panel` | 232/262 | 88.5% | 84.9% |
| `internal/contract` | 526/589 | 89.3% | 89.3% |
| `cmd/runctl` | 332/352 | 94.3% | 94.3% |
| `internal/transcript` | 225/236 | 95.3% | 신규 |
| `cmd/iapadapter` | 701/728 | 96.3% | 96.3% |
| `cmd/mediator` | 240/249 | 96.4% | 96.4% |
| `internal/runctl` | 110/113 | 97.3% | 97.1% |
| `internal/match` | 39/40 | 97.5% | 97.5% |
| `internal/api/ui` | 64/65 | 98.5% | 98.4% |
| `internal/proc` | 16/16 | 100.0% | 100.0% |
| `internal/schema` | 107/107 | 100.0% | 100.0% |

`internal/transcriptui` 는 문장이 0 이라 프로파일에 안 나온다 — 그 패키지의 주석이
그것을 설계로 적는다.

### 여유가 얇아진 자리 — `internal/enode`

87.0% 에서 **81.5%** 로 내려왔다. 하한까지 1.5%p — 덮이지 않은 문장이 약 60 개 더
늘면(2653/3316) 하한 아래다. 파일별로 보면 새로 들어온 격리 실행 코드가 끌어내렸다.

```text
   overlay_linux.go           31/64    48.4%    overlay 탐침.  사다리 셋 중 CI 가 닿는 칸이 적다
   runc_overlay_linux.go     387/636   60.8%    실제 mount · runc 경로는 integration 태그만 잰다
   setup.go                  128/198   64.6%
   diff.go                    38/56    67.9%    git diff 를 $OUT 에 담는 수확 재료
   claim.go                  351/395   88.9%
   runtime.go                 33/35    94.3%
```

**격리 경로를 넓히는 변경은 이 패키지에 문장을 더한다.** 기본 테스트가 가짜 helper
프로세스로 돌 수 있는 부분(프로토콜 · 검증 · OCI config · 수확)은 덮이고, 실제
namespace 를 여는 부분은 CI 밖이다. `cmd/enode` 도 97.2% 에서 83.0% 로 내려왔다 —
`env` 하위명령과 기동 전 준비도 검사다.

### 기준선 표가 더 낡았다 — 이것이 부채다

`.coverage-contract.yml` 의 `packages:` 표는 **여전히 열다섯 줄**이다. 오늘
`go list ./...` 는 스물하나를 낸다.

```text
   표에 없는 것    internal/panel · internal/proc · internal/api/ui (2026-09-15 판에도 없었다)
                   internal/environment · internal/transcript · internal/transcriptui (신규)
   그래도 걸리나   걸린다.  CI 의 awk 가 프로파일에서 직접 세므로 표와 무관하다
   무엇이 낡았나   measured_total_pct: 87.5 와 measured_at_commit: fa444f2b.
                   오늘 같은 명령이 내는 값은 85.3% 다
```

### 스킵은 0 이 정본이다

이 실행에서 스킵이 **0 건**이었다. 패키지 스물하나가 전부 결과를 냈다
(`transcriptui` 는 「테스트 파일 없음」의 skip 사건 하나 — 테스트 스킵이 아니다).

### 재는 값이 흔들리는 자리

```text
   DB 가 없으면           internal/api 의 통합 테스트가 스킵된다.  그래도 exit 0 이다
   -coverpkg 가 없으면    다른 패키지의 테스트가 덮은 문장이 빠진다
   하네스 스텁이 없으면    internal/enode 가 낮게 읽힌다.  스킵 둘이 생긴다
   플랫폼이 다르면        windows 빌드 태그 파일과 linux 전용 파일(overlay_linux ·
                          runc_overlay_linux)이 분모를 바꾼다
   integration 태그       runc_overlay_integration_test.go 가 CI 에서 컴파일조차 안 된다
```

### 테스트의 모양

```text
   Go 테스트 함수    1,127  (func Test 로 시작하는 선언.  최상위 통과 사건은 1,114)
   테스트 파일       131 (전체 263 중)
   별도 테스트 패키지  없다 — 전부 같은 패키지 안의 *_test.go
   브라우저 테스트    internal/api/ui/tests/*.test.mjs 열넷
   단언 라이브러리    없다.  표준 testing 만 쓴다
```

2026-09-15 판의 「917」은 셈법이 적혀 있지 않아 오늘 값과 직접 비교하지 않는다.

---

## Code Quality Indicators

### 빌드와 vet

`go build ./...` 와 `go vet ./...` 가 이 기계에서 둘 다 exit 0 이다 (2026-09-23).

### 차단되는 것과 경고만인 것이 갈린다

```text
   차단    포맷 · vet · 테스트 · 커버리지 80% · 스킵 0 · U+2605 0 ·
           출력 문자열의 장식 문자 0 · govulncheck · enodectl.exe 심볼 상한
   경고    golangci-lint 하나.  CI 의 유일한 continue-on-error
```

**이번 측정에서도 린트를 못 돌렸다** — 이 기계에 `golangci-lint` 가 없다.

### 표기 규약이 기계 검사다

```text
   U+2605 한 글자          grep -rlIP.  파일 수 상한 0
   출력 문자열의 장식 문자   별도 스텝
   emphasis-check.py       밀도와 뭉침.  enode-design/scripts/ 에 한 벌만 둔다
```

### 주석이 설계 논거를 진다 — 언어가 갈리기 시작했다

이 저장소의 주석은 **왜 그렇게 골랐는지**를 적는다 (`CONVENTIONS.md` 2.2 가 주석을
한국어로 남긴다). 격리 실행 코드(`runtime.go` · `runc_overlay_linux.go` ·
`internal/environment`)는 주석이 짧고 한국어에 영어 용어가 섞인 문체다 — 앞 코드가
실측 일자와 뒤집힌 결정을 주석에 남긴 것과 결이 다르다. 규약 위반은 아니다.

---

## Technical Debt

### 명령 종료 뒤의 창이 크기에 비례하고 밖에서 안 보인다

단계의 명령이 끝나도 노드는 수확 · 세션 정리 · 업로드를 마친 뒤에야 결과를 보고하고,
그동안 단계는 `CLAIMED` 이고 임대는 쥐어져 있다. 그 창의 비용이 결과가 아니라
워크스페이스 크기를 따른다.

```text
   Discover     command 단계는 언제나, agent 단계는 완주했을 때.  changedSince 가
                워크스페이스 전체를 걷는다 (claim.go:693 · :879)
   RecordDiff   워크스페이스 설정과 계약의 workspace 가 둘 다 있으면.  git diff --binary
   RemoveAll    runc-overlay 세션을 닫을 때 runRoot 전체.  upper 의 항목 수에 비례
```

진행 조회에 phase 가 없고, 결과에 종료 시각이 없고, `steps.ended_at` 은 result 가
닿은 시각이다. 정본 ADR-075 · ADR-076 이 이 자리를 결정 · 초안으로 닫았고 코드는
0 이다.

### runc-overlay 의 위층이 버려진다

단계가 워크스페이스에 쓴 것은 upper 에 쌓이고 세션을 닫을 때 지워진다. 결과로 남는
것은 `$OUT` 과 수확이 담은 diff · 변경 목록뿐이다. 굽기처럼 워크스페이스 자체를 바꾸는
것이 목적인 일을 받을 경로가 없다 (ADR-077).

### `min_free_gb` 가 가리는 것이 좁다

워크스페이스 경로의 여유만 재고, 모자라면 **빌드 능력 키(`arch` · `arch.<이름>`)만**
광고에서 뺀다. 노드는 계속 광고하고 다른 계약을 받는다. runtime scratch 의 여유는
안 잰다.

### 경계 검사가 새 패키지 하나를 모른다

`internal/environment` 가 금지 표에 없다. Mediator 쪽 `store` 가 결과 타입 하나 때문에
그 패키지를 물어, `cmd/mediator` 가 apt-get · debootstrap 을 부르는 코드를 링크한다.

### 결과 보고가 노드의 자기 신고로 걸러진다

`ReportStep` 은 본문의 `node` 와 `state='CLAIMED'` 로만 거른다. 인스턴스를 안 본다.
토큰이 하나라 노드를 가리는 자격이 따로 없다 — 이 설계가 I1 의 전제(열쇠가 하나)와
묶여 있고, 새 노드 표면을 더하면 같은 모양을 물려받는다.

### 정본이 코드보다 늦은 자리

```text
   ADR-071   정본은 「결정 · 미구현」이다.  코드는 70d6258 에서 구현했다
   ADR-073   §11 「남은 것」이 E5(BitBake 완주)를 남은 것으로 적는다.
             SunnyVM 제품 경로의 완주가 원장에 있다 (EN-bf4045e7)
```

### 커버리지 기준선 표가 여섯 패키지를 놓쳤다

위 「기준선 표가 더 낡았다」. 게이트는 안 뚫리지만 표가 진실이 아니다.

### `cmd/enodectl/probe.lock` 을 테스트가 건드린다 — 이번에도 재현했다

체크인된 픽스처인데 테스트가 제자리에서 변형한다. 이번 측정 뒤 작업 트리가
`M cmd/enodectl/probe.lock` 으로 더러워졌고 (내용 `1094256` 이 `3862989` 로), 손으로
되돌렸다. 임시 디렉터리로 복사해 쓰는 것이 정석이다.

### 심볼 캡의 취약함 — avprobe 사건의 흉터

Windows 크로스빌드 `enodectl.exe` 에 링커 도달 가능 `T` 심볼 상한이 걸려 있다.
`enodectl env` 가 형제 exec 로 이 규율을 지킨다.

### 큰 파일 넷

```text
   internal/contract/contract.go        1,799 줄.  계약 문법 전부가 한 파일이다
   internal/enode/runc_overlay_linux.go 1,285 줄.  runtime · 세션 · helper · OCI · smoke
   internal/api/api.go                  1,122 줄.  라우팅 + 핸들러
   internal/enode/claim.go              1,030 줄.  Worker · 단계 실행 · 업로드 · 보고
```

---

## Patterns and Anti-patterns

### 지킬 만한 패턴

```text
   허용목록으로 막는다      env · MCP 서버.  지우는 쪽은 열리는 쪽으로 틀린다
   결정을 순수 함수로       match · Argv · Validate · transcript.Parse · environment.Parse
   가장자리를 한 자리로     하네스 exec 은 runHarness 하나.  단계 실행은 StepSession 하나.
                            SQL 은 internal/store 하나
   읽기와 고치기를 가른다    env check 와 env apply.  기동은 고치지 않는다
   불변 산출물에 이름을      prepared_environment_id 가 단계 결과에 남는다
   부모가 죽으면 자식도      Pdeathsig + unshare --kill-child.  고아 namespace 가 안 남는다
   실패를 앞으로 당긴다     계장 치명 검사 · runtime open 의 경로 검증 · 투영 전 ELF 의존 확인
   두 벌로 안 둔다          파서와 껍데기 짓기가 한 패키지.  카드 렌더러가 한 장
```

### 냄새 · 안티패턴

```text
   경고 전용 린트           새 findings 가 조용히 쌓인다.  이 기계에서는 아예 못 잰다
   기준선 표의 수동 갱신     게이트 입력이 아니라 낡아도 안 빨개진다.  두 번째로 낡았다
   체크인된 픽스처를 테스트가 쓴다   cmd/enodectl/probe.lock.  이번에도 재현했다
   임대 창의 크기 비례 일    Discover · RecordDiff · RemoveAll 이 보고 앞에 있다
   CI 밖의 핵심 경로         runc-overlay 의 실제 namespace 경로는 integration 태그만 잰다
   쓰이지 않는 상태 상수     RESOLVING · ALLOCATING
   CI 주석의 낡은 수         「np=15 want=15」.  오늘은 21 이다
```
