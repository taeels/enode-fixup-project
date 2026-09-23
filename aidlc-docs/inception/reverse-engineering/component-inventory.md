# 컴포넌트 목록 — 오늘의 트리

**2026-09-23 전면 재측정.** 2026-09-15 판(패키지 열여덟)을 대체한다. 오늘
`go list ./...` 는 **스물하나**를 낸다. 기준 커밋은 `195a5d0`
(`v4-run-finalize-bake` — `origin/main` 위에 `unit/runtime-environment-profile` 을
합친 나무)이다.

---

## Application Packages (바이너리 다섯)

| 패키지 | 비테스트 파일 · 줄 | 목적 |
|---|---|---|
| `cmd/mediator` | 2 · 576 | HTTP API 서버. 유일하게 포트를 여는 서버 프로세스. `setup` 이 DB 를 프로비저닝한다 |
| `cmd/enode` | 5 · 535 | 실행 노드 데몬. 하위명령 다섯 — `runtime-helper` · `hook` · `setup` · `panel` · `env` |
| `cmd/enodectl` | 8 · 599 | 노드 로컬 제어 CLI. `list·setup·env·id·start·stop·logs·serve·status·version` |
| `cmd/runctl` | 2 · 797 | 무상태 클라이언트 CLI. 열한 하위명령. **`mcp` 는 없다** |
| `cmd/iapadapter` | 7 · 2,078 | 이슈 추적기 브리지. 저장소에서 유일하게 바깥을 향한다 |

## Shared Packages (`internal/` 열여섯)

| 패키지 | 비테스트 파일 · 줄 | 유형 | 목적 |
|---|---|---|---|
| `internal/enode` | 41 · 10,137 | 노드 로직 | 데몬 전부. 가장 큰 패키지. `StepRuntime` 과 runc-overlay 구현이 여기 있다 |
| `internal/store` | 13 · 5,190 | 상태 층 | PostgreSQL. SQL 이 전부 여기 있다 |
| `internal/contract` | 6 · 2,404 | 도메인 모델 | 계약 문법의 단일 원천 + 문법 교재 |
| `internal/api` | 8 · 2,345 | HTTP 층 | Mediator 라우트 27. 데모 모드와 갤러리를 포함한다 |
| `internal/panel` | 6 · 1,453 | 노드 로컬 서버 | 제어판. 라우트 열하나 |
| `internal/environment` | 10 · 1,400 | 실행 환경 | **신규.** profile · `env check` · `env apply` · 준비 산출물 manifest (ADR-073) |
| `internal/record` | 2 · 1,179 | 파일시스템 | Run Record 짓기 · 봉인 · tar. **진행 파일이 신규** |
| `internal/transcript` | 4 · 916 | 순수 파서 | **신규.** 하네스 stdout 을 사건 열로 읽는 유일한 길 |
| `internal/config` | 2 · 408 | 설정 | Mediator 의 YAML 과 토큰 쓰기 |
| `internal/runctl` | 1 · 376 | HTTP 클라이언트 | runctl 과 제어판이 함께 쓴다 |
| `internal/schema` | 1 · 238 | 검증 | 일부러 불구가 된 form-only JSON Schema |
| `internal/api/ui` | 1 · 151 + 정적 | 정적 자산 | `GET /ui/` 아래 번들 넷. 보안 헤더 다섯 |
| `internal/match` | 1 · 150 | 순수 함수 | 결정 코어. 부작용 0 |
| `internal/proc` | 3 · 151 | 프로세스 | 잠금 파일 pid · 멈춤 신호 · 자식 떼기 |
| `internal/build` | 1 · 75 | 빌드 정보 | 버전 한 줄 |
| `internal/transcriptui` | 1 · 24 + `card.mjs` | 정적 자산 | **신규.** 트랜스크립트 카드 렌더러 한 장. 제어판과 현황판이 같은 바이트를 낸다 |

## Infrastructure Packages

**없다.** CDK · Terraform · CloudFormation · Kubernetes 매니페스트가 트리에 0 개다.
배포라 부를 것은 `packaging/` 의 OS 설치본 셋과 GitHub Actions 워크플로다.

```text
   packaging/linux     nfpm 으로 .deb/.rpm.  서비스 유닛은 일부러 안 싣는다
   packaging/macos     install.sh 가 ~/.local/bin 에 복사.  ad-hoc 코드 서명
   packaging/windows   wixl 로 MSI.  ServiceInstall 도 PATH 수정도 없다
```

**노드의 실행 환경은 인프라 패키지가 아니다.** runc-overlay 노드가 쓰는 rootfs 는
`enode env apply` 가 노드 기계 위에서 debootstrap 으로 짓는 준비 산출물이고
(`internal/environment`), 저장소에 이미지나 매니페스트로 들어 있지 않다.

## Test Packages

별도 테스트 패키지가 없다 — 테스트는 전부 같은 패키지 안의 `*_test.go` 다
(비테스트 132 · 테스트 131 · 전체 263 파일). 브라우저 쪽은 예외로
`internal/api/ui/tests/*.test.mjs` 열넷이 있고 Go 테스트가 그것을 돌린다.

빌드 태그 `integration` 이 붙은 시험이 하나 있다 —
`internal/enode/runc_overlay_integration_test.go`. 실제 rootfs 와 namespace 를 여는
게이트이고 환경변수 셋(`ENODE_RUNC_ROOTFS` · `ENODE_RUNC_TEST_ROOT` ·
`ENODE_RUNC_HELPER`)이 없으면 스킵한다. **CI 의 `go test ./...` 는 이 태그를 안
붙이므로 이 시험은 CI 에서 컴파일조차 안 된다.**

## Total Count

```text
   Go 패키지          21   (go list ./...)
   바이너리            5   (cmd/*)
   공용 패키지        16   (internal/* — api/ui 를 따로 센다)
   인프라 패키지       0
   비테스트 소스     132 파일 · 31,551 줄
   전체 Go 소스      263 파일
```

## 2026-09-15 판과 달라진 것

두 갈래가 들어왔다. 트랜스크립트 회차의 유닛 여덟(`main`)과 실행 환경 구현
(`unit/runtime-environment-profile` — `main` 에 아직 없고 이 회차 브랜치가 합쳤다).

```text
   늘어난 패키지 셋    internal/environment · internal/transcript · internal/transcriptui
   늘어난 파일         비테스트 97 -> 132 · 전체 198 -> 263
   늘어난 라우트       Mediator 26 -> 27 (GET .../log) · 제어판 10 -> 11 (GET /static/card.mjs)
   늘어난 하위명령     enode runtime-helper · enode env · enodectl env
   구현된 것           StepRuntime 경계와 runc-overlay 격리 실행 (ADR-073)
                       실행 환경 profile 과 준비 산출물 (ADR-073)
                       계약의 rev 제거 · machine · arch.<이름> · overlay 광고 (ADR-070 · ADR-072)
                       도는 동안의 원문 — 진행 파일 · GET log · 노드 업로더 (트랜스크립트 FR-1 ~ FR-6)
                       봉인 로그를 허용목록이 아니라 크기 절단으로 (ADR-071)
   되살린 것           하네스 단계의 트랜스크립트 링 tee (트랜스크립트 회차의 질문 2 = A)
```
