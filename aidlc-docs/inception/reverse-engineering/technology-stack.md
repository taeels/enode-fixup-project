# 기술 스택 — 오늘의 값

**2026-09-23 전면 재측정.** 기준 커밋 `195a5d0`.

---

## 프로그래밍 언어

```text
   Go          1.26 (toolchain go1.26.6)   서버 · 노드 · CLI 전부.  비테스트 31,551 줄
   JavaScript  ES 모듈 (번들러 없음)        internal/api/ui/static · internal/transcriptui/card.mjs.
                                            브라우저가 직접 읽는다
   CSS         소스 그대로                  landing.css · demo.css · gallery.css · fleet.css
   SQL         PostgreSQL 방언              internal/store/schema.sql + 질의 문자열
   YAML        실행 환경 profile            enode.dev/v1alpha1 · kind execution-environment
   Python      3                            packaging/macos/check-signature.py ·
                                            enode-design/scripts/emphasis-check.py
   셸          bash                         scripts/*.sh · packaging/*/install.sh ·
                                            scripts/overlay-probe.sh · nested-runc-overlay-probe.sh
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
   임베드               //go:embed (정적 파일 · schema.sql · 예시 계약 · card.mjs)
   동시성               고루틴 · sync.Mutex · context.  워커 풀 라이브러리가 없다
   리눅스 시스템 호출    golang.org/x/sys/unix — mount(2) · mount_setattr · statfs · flock
   ELF 읽기            debug/elf — 투영할 실행파일의 동적 의존을 rootfs 안에서 확인한다
   테스트               표준 testing.  단언 라이브러리도 모킹 도구도 없다
   브라우저             ES 모듈 직접 로드.  프레임워크도 번들러도 없다
```

---

## 인프라

```text
   PostgreSQL 17     유일한 외부 상태 의존.  pgx/v5 풀로 붙는다
   파일시스템        봉인 Run Record (cfg.Artifacts.Root) · 진행 트리 (<Root>/progress/) ·
                     노드의 링 · 정책 · 상태 파일 · 준비 산출물 store · runtime scratch
   Docker            테스트 DB 뿐이다 (scripts/testdb.sh).  제품을 컨테이너로 안 낸다
   클라우드          없다.  CDK · Terraform · CloudFormation · Kubernetes 가 0 개다
```

---

## 격리 실행 (ADR-073) — 노드 기계에 있어야 하는 것

runc-overlay profile 을 쓰는 노드만 필요하다. native 노드는 아무것도 더 안 쓴다.

```text
   unshare (util-linux)   바깥 user · mount namespace.  --map-root-user --map-auto
   newuidmap · newgidmap  --map-auto 가 subordinate 범위를 쓴다 (uidmap 패키지)
   /etc/subuid · subgid   profile 의 host.require.subuid_size · subgid_size 이상
   runc                   단계 프로세스 하나를 OCI bundle 로 띄운다
   overlayfs              user namespace 안의 mount.  lowerdir 는 워크스페이스의 ro bind
   debootstrap            env apply 가 rootfs 를 짓는다 (noble · jammy · bookworm)
   sudo                   env apply 만 쓴다 — host 패키지 설치 · debootstrap · chroot ·
                          rootfs 봉인(chmod -R a-w).  euid 0 이면 안 붙인다.
                          단계를 돌리는 런타임 경로에는 없다
```

`enode env check` 가 이것들을 사실(fact)로 재고 상태 일곱 중 하나로 판정한다 —
`ready` · `installable` · `admin-required` · `external-blocked` · `unsupported` ·
`invalid` · `stale`. 전부 ready 일 때만 실제 runtime 으로 한 번 여닫는 smoke 를 돈다.

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
                        internal/api 의 통합 테스트가 스킵된다
   .github/ci-stubs/claude   가짜 하네스.  없으면 두 테스트가 조용히 스킵된다
   node                 internal/api/ui/tests/*.test.mjs 열넷.  브라우저 모듈 시험
   -tags integration    runc_overlay_integration_test.go 하나.  실제 rootfs 와 namespace.
                        CI 에 없다 — 사람이 환경변수 셋을 세우고 돈다
   scripts/*probe.sh    overlay 와 중첩 runc 가 되는 환경인지 재는 탐침 둘.  환경을 안 고친다
   emphasis-check.py    표기 규약 집행기 (enode-design/scripts/)
```

**측정 명령이 고정돼 있다.** `.coverage-contract.yml` 이 `env` 와 `flags` 와
`platform` 을 못 박는다 — 그 명령이 아닌 측정치는 게이트 입력으로 안 쓴다.

---

## 하네스

```text
   claude          유일한 등록 어댑터 (claudeHarness).  PATH 에서 찾는다
   호출 형태       claude -p --output-format stream-json --verbose [--model] [--max-turns]
                   [--permission-mode bypassPermissions] [--add-dir ...]
                   + 계장이 얹는 것 — --setting-sources "" · --settings ·
                   --plugin-dir= · --strict-mcp-config · --mcp-config=
   runc-overlay    하네스 실행파일 · enode 자신 · 계장 디렉터리 · 자격증명 helper 를
                   /run/enode/ 아래 고정 경로로 투영한다 (Project).  rootfs 안에서
                   동적 의존이 풀리는지 ELF 로 먼저 확인한다
   쓸 수 있는가    claude auth status --json 의 loggedIn.  형식을 모르면 쓸 수 있는 쪽
   버전            claude --version.  단계마다 다시 부른다 (캐시하면 드리프트를 숨긴다)
```

두 번째 하네스가 없어서 `acp.go` 를 안 만들었다 — `Harness` 인터페이스는 있고
구현이 하나다.
