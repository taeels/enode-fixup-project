# 컴포넌트 목록 — 오늘의 트리

**2026-09-15 전면 재측정.** 2026-09-08 판(패키지 열둘)과 2026-09-11 부분
갱신판을 대체한다. 오늘 `go list ./...` 는 **열여덟**을 낸다.

---

## Application Packages (바이너리 다섯)

| 패키지 | 비테스트 파일 · 줄 | 목적 |
|---|---|---|
| `cmd/mediator` | 2 · 576 | HTTP API 서버. 유일하게 포트를 여는 서버 프로세스. `setup` 이 DB 를 프로비저닝한다 |
| `cmd/enode` | 4 · 386 | 실행 노드 데몬. 하위명령 셋 — `hook` · `setup` · `panel` |
| `cmd/enodectl` | 7 · 557 | 노드 로컬 제어 CLI. `list·setup·id·start·stop·logs·serve·status·version` |
| `cmd/runctl` | 2 · 797 | 무상태 클라이언트 CLI. 열한 하위명령. **`mcp` 는 없다** |
| `cmd/iapadapter` | 7 · 2,078 | 이슈 추적기 브리지. 저장소에서 유일하게 바깥을 향한다 |

## Shared Packages (`internal/` 열셋)

| 패키지 | 비테스트 파일 · 줄 | 유형 | 목적 |
|---|---|---|---|
| `internal/api` | 7 · 2,056 | HTTP 층 | Mediator 라우트 26. 데모 모드와 갤러리를 포함한다 |
| `internal/api/ui` | 1 · 135 + 정적 | 정적 자산 | `GET /ui/` 아래 번들 넷. 보안 헤더 다섯 |
| `internal/store` | 13 · 5,093 | 상태 층 | PostgreSQL. SQL 이 전부 여기 있다. QUEUED 를 포함한다 |
| `internal/enode` | 33 · 7,769 | 노드 로직 | 데몬 전부. 가장 큰 패키지 |
| `internal/contract` | 6 · 2,402 | 도메인 모델 | 계약 문법의 단일 원천 + 문법 교재 |
| `internal/panel` | 5 · 909 | 노드 로컬 서버 | 제어판. 라우트 열 |
| `internal/record` | 1 · 410 | 파일시스템 | Run Record 짓기 · 봉인 · tar |
| `internal/config` | 2 · 408 | 설정 | Mediator 의 YAML 과 토큰 쓰기 |
| `internal/runctl` | 1 · 344 | HTTP 클라이언트 | runctl 과 제어판이 함께 쓴다 |
| `internal/schema` | 1 · 238 | 검증 | 일부러 불구가 된 form-only JSON Schema |
| `internal/match` | 1 · 150 | 순수 함수 | 결정 코어. 부작용 0 |
| `internal/proc` | 3 · 151 | 프로세스 | 잠금 파일 pid · 멈춤 신호 · 자식 떼기 |
| `internal/build` | 1 · 75 | 빌드 정보 | 버전 한 줄 |

## Infrastructure Packages

**없다.** CDK · Terraform · CloudFormation · Kubernetes 매니페스트가 트리에 0 개다.
배포라 부를 것은 `packaging/` 의 OS 설치본 셋과 GitHub Actions 워크플로다.

```text
   packaging/linux     nfpm 으로 .deb/.rpm.  서비스 유닛은 일부러 안 싣는다
   packaging/macos     install.sh 가 ~/.local/bin 에 복사.  ad-hoc 코드 서명
   packaging/windows   wixl 로 MSI.  ServiceInstall 도 PATH 수정도 없다
```

## Test Packages

별도 테스트 패키지가 없다 — 테스트는 전부 같은 패키지 안의 `*_test.go` 다
(비테스트 97 · 전체 198 파일). 브라우저 쪽은 예외로
`internal/api/ui/tests/*.test.mjs` 열넷이 있고 Go 테스트가 그것을 돌린다.

## Total Count

```text
   Go 패키지          18   (go list ./...)
   바이너리            5   (cmd/*)
   공용 패키지        13   (internal/* — api/ui 를 따로 센다)
   인프라 패키지       0
   비테스트 소스      97 파일 · 24,534 줄
   전체 Go 소스      198 파일
```

## 2026-09-08 판과 달라진 것

```text
   늘어난 패키지 셋    internal/panel · internal/proc · internal/api/ui
   늘어난 파일         비테스트 75 -> 97 · 전체 143 -> 198
   늘어난 라우트       15 -> 26
   늘어난 하위명령     enode panel · enodectl serve
   구현된 것           QUEUED · WakeQueued (ADR-064).  옛 판은 「코드가 안 쓴다」로 적었다
   꺼진 것             하네스 단계의 트랜스크립트 링 tee (짝 팩 decisions 6절 ⑲)
   바뀐 것             logs/ 가 원문에서 허용목록 선별로 (짝 팩 decisions 6절 ⑱)
                       Argv 가 --output-format stream-json --verbose 로 (⑮)
```
