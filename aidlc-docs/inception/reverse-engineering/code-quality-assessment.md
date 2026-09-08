# 코드 품질 평가 — 오늘의 코드

이 문서는 **지금 저장소에 있는 코드**의 품질 장치와 부채를 적는다. 게이트는 전부
`.github/workflows/ci.yml` 이 진다. 커버리지 스냅샷은 `.coverage-contract.yml`, 린트
설정은 `.golangci.yml` 이다.

---

## Test Coverage

### 게이트는 per-package 80% 바닥이다

커버리지 게이트는 **패키지마다 80% 바닥**이다. `ci.yml:269` 의 `awk -v floor=80` 이
세고 (스텝 이름 `ci.yml:265`), 바닥 밑으로 떨어지는 패키지가 하나라도 있으면 `exit 1`
로 CI 를 붉힌다 (`ci.yml:294-298`). 전체 합산이 아니라 **패키지 단위**라, 한 패키지의
여유가 다른 패키지의 부족을 못 가린다.

전체는 대략 87% 다. `.coverage-contract.yml:33` 이 `measured_total_pct: 87.5` 로
선언하고, 그 파일의 `packages:` 블록을 더하면 5,979 statements / 5,231 covered =
87.49% 로 맞는다.

### `.coverage-contract.yml` 은 게이트가 아니라 baseline 스냅샷이다 — 그리고 낡았다

이 파일은 스스로 「게이트가 아니라 **측정 조건**(명령·환경·플래그·플랫폼)을 고정한다」
고 밝힌다 (`.coverage-contract.yml:3-7`). 실제 게이트는 위의 `ci.yml` 커버리지 스텝이다.

문제는 **스냅샷이 낡았다**는 점이다.

```text
   앵커 커밋      measured_at_commit: fa444f2b9c411970e9885a29c08cca54d4383d48
                 (.coverage-contract.yml:47)
   검증          git cat-file -t fa444f2  ->  fatal: Not a valid object name
                 그 커밋은 지금 히스토리에 없다
   결과          스냅샷이 자기 앵커에서 재측정될 수 없다.  87.5% 라는 숫자가
                 어느 트리에서 났는지 저장소가 답하지 못한다
```

앵커 커밋이 사라졌으므로 스냅샷(5,979 statements · 87.5%)과 현재 HEAD 사이의 **드리프트
는 측정되지 않은 채로 있다**. 라이브가 ~6,200 statements / ~87.4% 쪽으로 밀렸다는
관측이 있으나, 이 값은 이 스캔에서 **재현하지 못했다** — 재현하려면 전체 커버리지
게이트(Postgres 서비스 + claude stub)를 돌려야 한다. baseline 을 신뢰 지점으로 쓰기
전에 살아 있는 커밋에서 다시 재는 것이 먼저다.

### 재는 값이 흔들리는 자리들

```text
   internal/build     정확히 80.0% (16/20).  산술로 바닥에 붙어 있어 statement
                      하나만 더 늘어도 바닥이 붉어진다 (.coverage-contract.yml:56-59)
   internal/enode     claude stub 유무로 covered 가 1294 vs 1293 로 갈린다
                      (.coverage-contract.yml:42-43).  stub 스텝(ci.yml:63-70)이
                      이 흔들림을 재우려고 있다
   플랫폼 파일        //go:build windows 파일은 linux 프로파일에 안 잡혀 분모를
                      바꾼다.  그래서 contract 가 platform: linux/amd64 로 못 박는다
                      (.coverage-contract.yml:27-31)
   packaging/         커버리지에서 제외된다
```

측정 자체도 두 번 돈다 — `go test ./...` 를 테스트 게이트용으로 한 번(`ci.yml:216`),
커버리지 계측용으로 또 한 번(`ci.yml:267`). 그리고 커버리지 awk 는 중복 coverpkg 블록을
**max count 로 병합**한다(`ci.yml:264,272`) — 순진하게 더하면 58.6% 를 6.5% 로 잘못
읽는 footgun 이라 편집할 때 주의해야 한다.

### skip 은 0 이 정본이다

skip-watch 스텝(`ci.yml:341-343`)이 allowlist 밖의 `t.Skip` 을 하나라도 발견하면
`exit 1` 한다. allowlist `.ci-allowed-skips` 는 **의도적으로 비어 있다**(주석만, 0 항목)
— 어떤 skip 도 용인하지 않는다는 뜻이다. 형식은 5개 파이프 컬럼
(`test_name | file | condition | why_unprovisionable | requested_by`)으로 문서화돼 있다.

---

## Code Quality Indicators

### 차단되는 것과 경고만인 것이 갈린다

```text
   차단 (실패하면 exit 1)
      stub 체크            ci.yml:63
      U+2605 glyph grep    ci.yml:74-82   (매치 파일 수가 0 이어야 통과)
      glyphscan.go AST     ci.yml:92      (문자열/문자 리터럴의 장식 문자)
      gofmt                ci.yml:95
      go vet               ci.yml:97
      govulncheck          ci.yml:182     (DB 불가 -> warn + exit 0 의 tri-state)
      go test              ci.yml:215
      per-package 커버리지 ci.yml:265
      skip-watch           ci.yml:341
      cross-build          ci.yml:454
      enodectl 심볼 캡     ci.yml:479-485

   경고만 (continue-on-error)
      golangci-lint        ci.yml:117     저장소에서 continue-on-error: true 는 이 하나뿐
```

`gofmt` 와 `go vet` 은 **차단**이다 — 포맷·vet 위반은 merge 를 막는다. 반면
`golangci-lint` 는 **경고 전용**이다 (스텝 이름 `ci.yml:116`, `continue-on-error: true`
는 `ci.yml:117` 뿐). 린트가 무엇을 말하든 게이트를 붉히지 않는다.

### 린트 설정은 재현성에 맞춰져 있다

`.golangci.yml` 은 golangci-lint v2 에서 `default: none` 으로 시작해 다섯을 이름으로
켠다 — `errcheck`, `govet`, `ineffassign`, `staticcheck`, `unused` (`.golangci.yml:16-22`).
`max-same-issues: 0` · `max-issues-per-linter: 0` (`.golangci.yml:69-70`)으로 **보고를
자르지 않는다** — 기본값(3·50)은 무엇을 잘랐는지 말하지 않고 보고를 잘라내므로 껐다.

경고 전용 정책은 현재 **약 20건 규모의 baseline findings** 를 안고 간다. 이 수는
어느 contract 파일에도 못 박혀 있지 않고 린트의 라이브 출력이라, 코드가 바뀌면 흔들린다.
차단이 아니므로 새 findings 가 조용히 쌓일 수 있다는 점이 이 정책의 약한 자리다.

---

## Technical Debt

### 하네스 whole-buffer 캡처가 라이브 트랜스크립트를 막는다

`internal/enode/runner.go` 는 하네스 stdout/stderr 를 통째로 `bytes.Buffer` 에 담고
(`runner.go:94-96`), 프로세스가 끝난 **뒤에야** `Decode` 한다 (`runner.go:106`).
`io.MultiWriter` 나 라이브 tee 가 없다. `emit` 콜백은 있지만 `EventFinal` 하나만
나므로(`harness.go:213`) 사후 배치 디코드에서 한 번 튈 뿐이다. 명령 스텝도 같은 모양이라
버퍼를 완료 후 업로드한다 (`claim.go:563-576`). 결과적으로 **오래 도는 스텝은 완료
전까지 증분 로그를 하나도 내지 못한다** — 대시보드/라이브 트랜스크립트를 붙이려면
이 whole-buffer 지점을 먼저 걷어내야 한다.

### `cmd/enodectl/probe.lock` 을 테스트가 건드린다

`cmd/enodectl/probe.lock` 은 체크인된 테스트 픽스처(8 bytes)인데, 테스트가 이것을
변형한다. 저장소에 든 파일을 테스트가 만지므로 dirty-tree·테스트 순서 취약성의 씨앗이
된다 — 픽스처를 임시 디렉터리로 복사해 쓰는 것이 정석이지만 지금은 제자리에서 쓰인다.

### 심볼 캡의 취약함 — avprobe 사건의 흉터

Windows 크로스빌드 `enodectl.exe` 에 링커 도달 가능 `T` 심볼 상한이 걸려 있다 —
`crypto/tls` T <= 10, `net/http` T <= 50 (`ci.yml:479-485`, 실측 카운트 `grep -c` 는
`ci.yml:482-483`). rc14 실측은 1 · 6 이다 (`ci.yml:476`).

```text
   왜 있나    rc13 의 enodectl.exe 가 AhnLab V3 에 Trojan/Win.Generic.C5874069 로
              삭제됐다 (avprobe/README.md:3-4,22-24).  setup 이 internal/enode 전체를
              링크하며 net/http·crypto/tls 스택을 끌어와, 프로세스-kill 코드 옆에
              TLS-네트워킹 코드가 놓이자 generic AV 규칙이 반응했다
   고침        enodectl setup 을 enode setup 으로 exec 위임 (setup.go:10-27),
              crypto/tls 심볼을 24 로 되돌림
   취약함      캡은 import 그래프가 아니라 링커 심볼을 센다 (go list -deps 는 DCE 를
              놓치므로, ci.yml:469-471).  재-링크는 잡지만 다른 AV 휴리스틱은 못 잡는다.
              실제로 이후 rc15 에서 Defender 가 Trojan:Win32/Wacatac.C!ml 로 잡았고
              (avprobe/README.md:244-249) 이건 캡이 커버하지 않는다
```

캡은 좁은 가드다 — 한 가지 회귀(재-링크)만 막고, AV 판정이 다른 축으로 오면 다시 뚫린다.

### 그 밖의 부채

```text
   버전 없는 마이그레이션   Migrate 가 schema.sql 을 통째로 exec.  ALTER ... ADD COLUMN
                            IF NOT EXISTS 를 쌓는다 (store.go:79-84).  마이그레이션 도구 없음
   steps.state CHECK 없음   어휘가 늘 때 강제 마이그레이션을 피하려 일부러 뺐다.
                            새 상태는 reap/verify 열거를 다시 감사해야 한다 (verdict.go:26-28)
   죽은 상수                RESOLVING · ALLOCATING 상태 상수가 정의만 되고
                            (store.go:157-158) 어디에도 쓰이지 않는다
   unix 에서 죽은 코드      processAlive 가 정의돼도 (proc_unix.go:24) unix 경로에서
                            아무도 안 부른다.  windows ownsConfig 만 쓴다
   fire-and-forget 알림     ask 푸시는 goroutine 에서 재시도 없이 나간다 (ask.go:115-129).
                            인박스(PendingAsks)가 정본이라 손실을 견딘다
   claim tx 밖 stamping     stampRoleAttrs 등이 claim 트랜잭션 밖에서 best-effort 로 돈다
                            (claim.go:422-423).  재전달 claim 이 다른 프롬프트를 낼 수 있다
   삼켜진 에러              fail/write 가 JSON 인코드 에러를 버리고, getBlob/getRecord 는
                            헤더 전송 후 copy/tar 에러를 로그만 하고 클라이언트에 안 알린다
                            (api.go:797,830)
   blob 10 MiB 상한         데모 범위에 묶인 값.  풀 이미지 flash 는 곧 넘긴다
```

---

## Patterns and Anti-patterns

### 지킬 만한 패턴

```text
   순수 매처          internal/match 는 부작용 없는 결정론 정렬(attrCount·NodeID,
                      match.go:81-87).  제출과 dry-run 이 동일 함수를 쓴다
   단일 정본 문자열   answerRoute 상수(api.go:84)가 mux 등록과 알림 딥링크 둘의 원천.
                      CheckPlan 은 두 번째 검증기 대신 Contract.Validate 를 재사용한다
   계층 분리          핸들러는 SQL 을 안 쓴다.  영속은 store 로 위임하고, DB 의미는
                      store 안에 산다.  state 층은 HTTP 라우트를 만들지 않는다
                      (NotifyURL/AnswerPath 는 주입, store.go:38-46)
   단일 exec 초크포인트 runHarness(runner.go:88)가 하네스 실행의 유일 지점 — env
                      화이트리스트를 한 번만 강제한다.  os.Environ() 은 상속 안 한다
   pull-only 노드     안으로 포트를 안 연다 (advertise.go:16).  임대 권한은 시간 기반
                      (not_after)이라 일시적 하트비트 실패를 견딘다
   상수시간 토큰 비교  subtle.ConstantTimeCompare (api.go:116)
   savepoint 격리     tryGrab 이 lease INSERT 를 savepoint 로 감싸 unique 위반이
                      바깥 tx 를 무너뜨리지 않게 한다 (acquire.go:171-188)
   재현 가능한 게이트  린트 max-same-issues:0, schema 에러 키 정렬,
                      glyphscan 이 scanned-count 를 내 empty-input tautology 를 피함
```

### 냄새·안티패턴

```text
   whole-buffer 캡처   스트리밍 없음.  위 Technical Debt 참조 (runner.go)
   취약한 순서 제약    main.go defer LIFO — reaper stop 이 pool close 앞에 와야 한다
                      (main.go:89-101).  ReportStep/AnswerStep 의 rollback-before-effects
                      (claim.go:789-798)는 버그를 고치며 얻은 순서 규칙
   attrCount 휴리스틱  희소성 대리 지표라, 장식(비-capability) attr 이 광고되면 깨진다
                      (match.go:28-33).  두 번째 정렬 기준을 더하면 금지된 Rank 가 된다
   free-text 매칭      runctl nextStep 이 Mediator 의 자유 텍스트 reason 을 패턴 매칭
                      (main.go:402).  구조화된 에러 코드가 없어 문구가 바뀌면 깨진다.
                      verdictRe 는 RE2 \b 가 ASCII 전용이라 한국어를 위해 \b 를 피한다
                      (main.go:425-428)
   커버리지 awk 병합   중복 coverpkg 를 max count 로 병합 — 더하면 오독 (ci.yml:264,272)
   바닥에 붙은 패키지  internal/build 가 80.0% 에 산술로 붙어 있다
   probe.lock 변형     체크인 픽스처를 테스트가 제자리에서 건드린다
   삼켜진 에러         fail/write · 스트림 중간 copy/tar 에러 무시
```
