# 코드 품질 평가 — 오늘의 코드

**2026-09-15 전면 재측정.** 커버리지와 스킵은 이 기계에서 실제로 돌려 잰 값이다 —
2026-09-08 판이 계약 파일의 기준선을 인용한 것과 다르다.

```text
   측정 명령    go test ./... -count=1 -coverpkg=./... -coverprofile=... -json
   환경         ENODE_TEST_DATABASE_URL 을 세우고 (scripts/testdb.sh)
                .github/ci-stubs 를 PATH 앞에 둔다
   결과         exit 0 · 패키지 18 · 스킵 0 · 전체 7,122/8,146 = 87.4%
```

이것은 `.coverage-contract.yml` 이 못 박은 바로 그 명령이다. 그 명령이 아닌
측정치는 게이트 입력으로 안 쓴다.

---

## Test Coverage

### 게이트는 패키지별 80% 바닥이다

저장소 전체 합계가 아니라 **패키지마다 개별로** 넘어야 한다. 합계로 보면 0% 인
패키지가 높은 패키지에 가려지고, 이 저장소가 실제로 그 모양이었다. 차단이다 —
한 패키지라도 미달이면 CI 가 exit 1 이다.

### 오늘의 측정값 — 열여덟 전부 통과

| 패키지 | 덮은 것/문장 | 비율 |
|---|---|---|
| `internal/build` | 16/20 | 80.0% |
| `internal/api` | 781/951 | 82.1% |
| `internal/record` | 131/159 | 82.4% |
| `internal/store` | 1455/1762 | 82.6% |
| `cmd/enodectl` | 172/206 | 83.5% |
| `internal/panel` | 191/225 | 84.9% |
| `internal/config` | 103/121 | 85.1% |
| `internal/enode` | 2013/2314 | 87.0% |
| `internal/contract` | 526/589 | 89.3% |
| `cmd/runctl` | 332/352 | 94.3% |
| `cmd/iapadapter` | 701/728 | 96.3% |
| `cmd/mediator` | 240/249 | 96.4% |
| `internal/runctl` | 100/103 | 97.1% |
| `cmd/enode` | 138/142 | 97.2% |
| `internal/match` | 39/40 | 97.5% |
| `internal/api/ui` | 61/62 | 98.4% |
| `internal/proc` | 16/16 | 100.0% |
| `internal/schema` | 107/107 | 100.0% |

**`internal/build` 가 정확히 80.0% 다.** 남은 넷은 이 프로젝트의 표준 명령
(`-buildvcs` 미지정)에서 도달 불가라 산술 상한이고, 그 패키지에 문장 하나만 늘어도
게이트가 빨개진다. 그때 할 일은 하한을 낮추는 것이 아니라 그 문장을 덮는 것이다.

### 기준선 표가 낡았다 — 이것이 부채다

`.coverage-contract.yml` 의 `packages:` 표는 **열다섯 줄**이다. 오늘 `go list ./...`
는 열여덟을 낸다.

```text
   표에 없는 것    internal/panel · internal/proc · internal/api/ui
   그래도 걸리나   걸린다.  CI 의 awk 가 프로파일에서 직접 세므로 표와 무관하다
   무엇이 낡았나   measured_total_pct: 87.5 와 measured_at_commit: fa444f2b.
                   오늘 같은 명령이 내는 값은 87.4% 다
```

표가 게이트의 입력이 아니라서 빨개지지 않는다 — **조용히 낡는다.** 파일 자신이
「브랜치가 main 으로 들어갈 때 이 값을 갱신한다」고 적었고 그 갱신이 안 됐다.

### 스킵은 0 이 정본이다

이 실행에서 스킵이 **0 건**이었다. `.ci-allowed-skips` 에 예순여덟 줄의 면제 목록이
있지만 오늘은 하나도 안 쓰였다. 스킵 감시 스텝은 패키지 수까지 함께 세서
「볼 것이 없었다」를 「스킵이 없다」로 읽지 않는다 — 감시 장치의 유일한 실패 양식이
항진명제가 되는 것이라 그것을 막는다.

### 재는 값이 흔들리는 자리

```text
   DB 가 없으면           internal/api 의 통합 테스트 일흔여덟이 스킵된다.
                          그래도 go test 는 exit 0 이다
   -coverpkg 가 없으면    다른 패키지의 테스트가 덮은 문장이 빠진다
   하네스 스텁이 없으면    internal/enode 가 한 문장 낮게 읽힌다.  스킵 둘이 생긴다
   플랫폼이 다르면        windows 빌드 태그 파일 셋이 리눅스 프로파일에 안 나온다
```

넷 다 종료코드로는 구별되지 않는다. 그래서 명령과 환경과 플랫폼을 계약이 고정한다.

### 테스트의 모양

```text
   Go 테스트 함수    917
   테스트 파일       101 (전체 198 중)
   별도 테스트 패키지  없다 — 전부 같은 패키지 안의 *_test.go
   브라우저 테스트    internal/api/ui/tests/*.test.mjs 열넷
   단언 라이브러리    없다.  표준 testing 만 쓴다
```

---

## Code Quality Indicators

### 빌드와 vet

`go build ./...` 와 `go vet ./...` 가 이 기계에서 둘 다 exit 0 이다 (2026-09-15).

### 차단되는 것과 경고만인 것이 갈린다

```text
   차단    포맷 · vet · 테스트 · 커버리지 80% · 스킵 0 · U+2605 0 ·
           출력 문자열의 장식 문자 0 · govulncheck · enodectl.exe 심볼 상한
   경고    golangci-lint 하나.  CI 의 유일한 continue-on-error
```

린트 설정은 재현성에 맞춰져 있다 — `default: none` 으로 시작해 다섯을 이름으로
켠다(`errcheck` · `govet` · `ineffassign` · `staticcheck` · `unused`)고
`.golangci.yml` 이 적고, 보고를 안 자른다(`max-same-issues: 0`). 경고 전용이라
새 findings 가 조용히 쌓일 수 있다는 것이 이 정책의 약한 자리다. **이번 측정에서
린트를 못 돌렸다** — 이 기계에 `golangci-lint` 가 없다. 그 값은 CI 에서만 읽힌다.

### 표기 규약이 기계 검사다

```text
   U+2605 한 글자          grep -rlIP.  파일 수 상한 0.  이진 파일은 건너뛴다
   출력 문자열의 장식 문자   별도 스텝.  위 한 글자 검사의 구멍을 메운다
   emphasis-check.py       밀도와 뭉침.  enode-design/scripts/ 에 한 벌만 둔다
```

세 검사가 한 규약의 세 면이다. 첫 스텝은 한 글자만 보므로 「집행하는 것처럼
보이면서 아무것도 집행하지 않는」 구멍이 있었고, 둘째가 그것을 메웠다. 그 사실이
CI 파일 주석에 적혀 있다.

### 주석이 설계 논거를 진다

이 저장소의 주석은 무엇을 하는지가 아니라 **왜 그렇게 골랐는지**를 적는다. 실측
일자와 뒤집힌 결정이 그대로 남아 있다 (`claude.go` 의 권한 모드 1차 · 2차 · 3차,
`transcript.go` 의 「윈도우가 이 설계를 정했다」). 한국어인 것이 규약이다 —
`CONVENTIONS.md` 2.2 가 되먹임 경로에 안 실리는 것만 한국어로 남긴다.

---

## Technical Debt

### 진행 중 하네스 출력을 읽을 표면이 없다

2026-09-08 판이 「whole-buffer 캡처가 라이브 트랜스크립트를 막는다」로 적은 자리가
**반만 풀렸고 다른 쪽이 새로 닫혔다.**

```text
   풀린 것    Argv 가 이미 -p --output-format stream-json --verbose 다 (짝 팩 ⑮).
              하네스가 도는 동안 사건 줄이 실제로 흐른다
   안 풀린 것  Decode 가 아직 io.ReadAll 로 EOF 까지 읽고 final 사건 하나만 낸다
   새로 닫힌 것 하네스 단계의 링 tee 를 껐다 (짝 팩 ⑲).  그래서 제어판 카드가
              에이전트 단계 내내 비어 있다 — 옛 판의 「끝에 한 줄」보다 더 비었다
   중앙        올리는 PUT 만 있고 내려받는 GET 이 없다.  PUT 도 단계 끝 한 번이다
```

이것이 `requirements/transcript/` 팩이 여는 자리다.

### 봉인 로그가 원문이 아니게 됐다

`selectLogs` 가 `logs/` 를 허용목록으로 거른다 (짝 팩 ⑱). 근거는 자격증명 누출이고
사용자 결정이다 (⑰). 대가가 둘이다.

```text
   ADR-005 의 「원문 그대로」    코드가 더는 그렇지 않다.  정본에 되돌려 올릴 자리다
   사람이 읽을 것이 줄었다       도구 입력도 도구 결과도 assistant 의 text 도 안 남는다
```

### `AppendLog` 의 상한이 호출마다 걸린다

`record.AppendLog` 는 `io.LimitReader(r, limit)` 로 **그 호출**을 자른다. 오늘은
단계당 한 번만 부르므로 파일 상한과 같은 뜻이지만, 나눠 올리기 시작하면 파일
전체는 상한을 넘는다. 반환값도 총 길이가 아니라 이번 호출이 쓴 바이트 수다.

### 커버리지 기준선 표가 패키지 셋을 놓쳤다

위 「기준선 표가 낡았다」. 게이트는 안 뚫리지만 표가 진실이 아니다.

### `cmd/enodectl/probe.lock` 을 테스트가 건드린다

체크인된 8 바이트 픽스처인데 테스트가 제자리에서 변형한다 (이번 측정에서도
mtime 이 갱신됐다). dirty-tree 와 테스트 순서 취약성의 씨앗이다 — 임시 디렉터리로
복사해 쓰는 것이 정석이다.

### 심볼 캡의 취약함 — avprobe 사건의 흉터

Windows 크로스빌드 `enodectl.exe` 에 링커 도달 가능 `T` 심볼 상한이 걸려 있다 —
`crypto/tls` 10 이하, `net/http` 50 이하.

```text
   왜 있나    rc13 의 enodectl.exe 가 AhnLab 에, rc15 가 Defender 에 삭제됐다.
              setup 이 internal/enode 전체를 링크하며 네트워크 스택을 끌어왔다
   고침       enodectl setup 을 enode setup 으로 exec 위임.
              enodectl serve 도 같은 규율로 enode panel 을 exec 한다
   취약함     실수로 한 줄 임포트하면 상한이 깨진다.  CI 가 그것만 막는다
```

### 큰 파일 둘

```text
   internal/contract/contract.go   1,797 줄.  계약 문법 전부가 한 파일이다
   internal/api/api.go             1,200 줄.  라우팅 + 핸들러 열일곱
```

둘 다 「한 곳에 두어 갈리지 않게 한다」가 근거이고 그 근거가 주석에 있다. 나누면
규칙이 두 벌이 될 자리라 지금은 부채로만 적는다.

---

## Patterns and Anti-patterns

### 지킬 만한 패턴

```text
   허용목록으로 막는다      env · MCP 서버 · logs/ 사건.  지우는 쪽은 열리는 쪽으로 틀린다
   결정을 순수 함수로       match · Argv · Validate.  시험이 싸고 dry-run 이 공짜다
   가장자리를 한 자리로     exec 은 runHarness 하나.  SQL 은 internal/store 하나
   실패를 앞으로 당긴다     계장 치명 검사가 전부 exec 앞이다.  자격증명 복사 실패도 치명
   없음과 비어 있음을 가른다  lease 는 null, nodes 는 [].  화면이 「못 읽음」을 안다
   두 벌로 안 둔다          answerRoute 하나가 등록과 알림 링크를 함께 낸다.
                            emphasis-check 는 enode-design 에 한 벌
   빌드 태그를 안 늘린다     링 파일이 회전을 포기해 윈도우와 유닉스가 같은 코드로 돈다
```

### 냄새 · 안티패턴

```text
   경고 전용 린트           새 findings 가 조용히 쌓인다.  이 기계에서는 아예 못 잰다
   기준선 표의 수동 갱신     게이트 입력이 아니라 낡아도 안 빨개진다.  실제로 낡았다
   체크인된 픽스처를 테스트가 쓴다   cmd/enodectl/probe.lock
   한 파일에 몰린 문법       contract.go 1,797 줄
   쓰이지 않는 상태 상수     RESOLVING · ALLOCATING 을 쓰는 코드가 없다.
                            갤러리가 비교값으로 읽기만 하고 아무도 그 값을 안 쓴다
   CI 주석의 낡은 수         「np=15 want=15」가 주석에 남아 있다.  오늘은 18 이다
```
