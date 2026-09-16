# 기술 스택 — 오늘의 값

**2026-09-15 전면 재측정.**

---

## 프로그래밍 언어

```text
   Go          1.26 (toolchain go1.26.6)   서버 · 노드 · CLI 전부.  비테스트 24,534 줄
   JavaScript  ES 모듈 (번들러 없음)        internal/api/ui/static.  브라우저가 직접 읽는다
   CSS         소스 그대로                  landing.css · demo.css · gallery.css · fleet.css
   SQL         PostgreSQL 방언              internal/store/schema.sql + 질의 문자열
   Python      3                            packaging/macos/check-signature.py ·
                                            enode-design/scripts/emphasis-check.py
   셸          bash                         scripts/*.sh · packaging/*/install.sh
```

Go 의 `go` 와 `toolchain` 을 같은 버전으로 묶었다 — 슬랙을 없애려는 것이고,
근거가 `go.mod` 의 주석에 있다 (「넷이 서로 다른 컴파일러로 짜면 내 기계에서는
되는데 가 나온다」).

---

## 프레임워크

**없다.** 그 자리를 표준 라이브러리가 진다.

```text
   HTTP 서버 · 라우팅   net/http.  Go 1.22+ 의 ServeMux 가 메서드와 경로 패턴을 받는다
   HTTP 클라이언트      net/http.  internal/runctl 과 노드의 Client
   로깅                 log/slog.  TextHandler
   직렬화               encoding/json · gopkg.in/yaml.v3
   아카이브             archive/tar (Record 와 팩) · compress/gzip (팩)
   임베드               //go:embed (정적 파일 · schema.sql · 예시 계약)
   동시성               고루틴 · sync.Mutex · context.  워커 풀 라이브러리가 없다
   테스트               표준 testing.  단언 라이브러리도 모킹 도구도 없다
   브라우저             ES 모듈 직접 로드.  프레임워크도 번들러도 없다
```

---

## 인프라

```text
   PostgreSQL 17     유일한 외부 상태 의존.  pgx/v5 풀로 붙는다
   파일시스템        봉인 Run Record (cfg.Artifacts.Root) · 노드의 링 · 정책 · 상태 파일
   Docker            테스트 DB 뿐이다 (scripts/testdb.sh).  제품을 컨테이너로 안 낸다
   클라우드          없다.  CDK · Terraform · CloudFormation · Kubernetes 가 0 개다
```

---

## 빌드 도구

```text
   go build / go test   빌드와 테스트의 전부
   nfpm                 packaging/linux — .deb/.rpm
   wixl (msitools)      packaging/windows — 리눅스에서 MSI 를 만든다
   codesign             packaging/macos — ad-hoc 서명
   GitHub Actions       CI.  잡 둘 (test · cross)
   golangci-lint        경고 전용.  CI 의 유일한 continue-on-error
   govulncheck          차단.  DB 접근 실패는 경고로 떨어진다
```

---

## 테스트 도구

```text
   go test              단위 · 통합 전부.  같은 패키지 안의 *_test.go
   postgres:17          통합 테스트의 실제 DB.  ENODE_TEST_DATABASE_URL 이 없으면
                        internal/api 의 통합 테스트 일흔여덟이 스킵된다
   .github/ci-stubs/claude   가짜 하네스.  없으면 두 테스트가 조용히 스킵된다
   node                 internal/api/ui/tests/*.test.mjs 열넷.  브라우저 모듈 시험
   emphasis-check.py    표기 규약 집행기 (enode-design/scripts/)
```

**측정 명령이 고정돼 있다.** `.coverage-contract.yml` 이 `env` 와 `flags` 와
`platform` 을 못 박는다 — 그 명령이 아닌 측정치는 게이트 입력으로 안 쓴다.
`ENODE_TEST_DATABASE_URL` 을 빼면 전체가 58.6% 에서 34.2% 로 떨어지는데
`go test` 는 그래도 exit 0 이다.

---

## 하네스

```text
   claude          유일한 등록 어댑터 (claudeHarness).  PATH 에서 찾는다
   호출 형태       claude -p --output-format stream-json --verbose [--model] [--max-turns]
                   [--permission-mode bypassPermissions] [--add-dir ...]
                   + 계장이 얹는 것 — --setting-sources "" · --settings ·
                   --plugin-dir= · --strict-mcp-config · --mcp-config=
   쓸 수 있는가    claude auth status --json 의 loggedIn.  형식을 모르면 쓸 수 있는 쪽
   버전            claude --version.  단계마다 다시 부른다 (캐시하면 드리프트를 숨긴다)
```

두 번째 하네스가 없어서 `acp.go` 를 안 만들었다 — `Harness` 인터페이스는 있고
구현이 하나다.
