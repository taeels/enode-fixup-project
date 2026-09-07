# 결정표 — 이미 정해진 것. 다시 논의하지 않는다

이 표는 `enode-features.md` 의 미정을 **착수 전에 값으로 닫은 것**이다.
게이트에서는 확인만 한다. 새로 묻지 않는다.

**게이트 규칙 한 줄 — 권장을 벗어나면 근거를 적는다.** 벗어나는 것은
허용이다. 근거 없이 벗어나는 것이 금지다.

```text
   1절   사용자가 확정한 것          값이다.  게이트에서 묻지 않는다
   2절   팩이 채운 권장값            값이다.  벗어나면 근거를 적는다
   3절   보안 확장 아래서의 취급      SECURITY 규칙마다 처리를 미리 적었다
   4절   이월                        이번에 정하지 않는다.  물어도 답이 「이월」이다
   5절   범위에 들어온 것             이월이던 것을 되돌린 자리.  근거를 함께 적는다
```

정본과 어긋나면 `enode-design/protocol/INVARIANTS.md` 가 이긴다
(`canon.md`).

---

# 1. 사용자가 확정한 것 (2026-09-04)

| 항목 | 값 | 근거 |
|---|---|---|
| 범위 | MVP 넷 — 중앙 현황판 · 대기열 · drain · 호스트 제어판. 그리고 `scene-gates.md` 의 장면 절단면. **5절이 2026-09-05 에 다섯째(Mediator MCP)를 더했다 — 이 행은 그날의 기록이므로 고치지 않는다** | 이틀 예산. 넷이 `dhseo` 장면 하나를 만든다 |
| 제어판 서버 위치 | `enodectl serve` 별도 프로세스. 새 실행파일 없음 | 멈춘 데몬을 `start` 하려면 데몬 밖에 있어야 한다. `packaging/` 비용 회피 |
| drain 표면 | 노드 로컬 정책 파일이 정본. 데몬이 광고 직전에 읽어 광고에 실어 보내고, 중앙은 후보에서 빼고 광고 응답으로 통보한다. **중앙 라우트 신설 없음** | `ADR-042` 방향. 소유의 증거가 「그 기계의 파일을 쓸 수 있다」로 닫힌다. 중앙 라우트로 받으면 토큰만 있으면 남의 노드를 뺄 수 있다 |
| 정책 저장 | 정본은 노드의 파일. 중앙은 광고로 받은 복사본을 `nodes` 열에 둔다 — 현황판과 매칭이 그것을 읽는다 | drain 표면에서 따라온다. `UpsertAdvert` 가 정책도 갱신한다 |
| 정책 파일 위치 · 형식 | 모델 몫 — Functional Design 에서 정한다. 기존 `enode` 설정 파일 옆, 같은 형식(yaml) | 사람이 고를 값이 아니다 |
| drain 기본 모드 | `graceful` | 놀라움이 적다. `at-boundary` 로 올리는 것은 소유자의 명시다. CP3 는 `at-boundary` 를 명시해서 돈다 |
| drain 중 광고 | 계속 보낸다. `draining` 을 실어서 | 안 보내면 중앙이 draining 을 모르고 통보 경로가 끊긴다. 현황판에 「있는데 안 빌려준다」로 보여야 한다 (`ADR-063` §4) |
| drain 해제 | 소유자가 명시적으로 푼다. 도는 Run 이 끝나도 자동으로 안 풀린다. 푸는 순간 후보로 돌아가고 대기 Run 이 집을 수 있다 | 정책은 만료되지 않는다 (`ADR-063` §2). 자동 복귀는 「돌려받았다」를 무효로 만든다 |
| `at-boundary` 의 뜻 | 도는 단계는 끝까지 간다. 결과를 보고한 뒤 임대를 놓고 더 집지 않는다. **놓인 Run 은 취소 경로로 종료한다** — 끝난 단계의 산출은 Record 에 남고, 남은 단계는 새 Run 으로 다시 낸다. 닫는 쪽은 Mediator — `postResult` 끝에서 그 노드가 `at-boundary` 면 `store.Cancel(run, "drain:<node_id>")`. 다중 노드 Run 도 Run 전체가 `FAILED` 다 | 단계 도중에 끊으면 선점이다 (`INVARIANTS` §2 재개 없음). 회수당하고도 죽지 않는 것은 `ADR-064` §3.4 — 이월. 새 신호를 만들지 않고 기존 취소 경로를 쓴다. `ADR-063` §2.1 「drain 은 Run 을 죽이지 않는다」는 graceful 에만 맞는 문장이라 개정한다 (사용자 결정 2026-09-04, `canon.md` 3절) |
| 대기 상한 | 없다. `QUEUED` 는 무기한. 빠져나가는 길은 `cancel` 뿐 | 상한은 정책이고 정책 틀은 안 만든다 (`constraints.md` §2). drain 무기한 x 대기 무기한의 상호작용은 CP3 가 실동작으로 덮는다 (`ADR-064` §5) |
| `at-boundary` 지연 상한 | 없다. 바닥(도는 단계의 남은 시간 + 광고 주기)이 곧 값 | 상한을 두면 단계 도중에 끊어야 한다. 더 급하면 소유자가 제어판에서 `stop` 한다 — 그쪽이 명시적 폐기다 |
| 대기 응답 코드 | `202 Accepted` | `201` 은 「만들었다」, `202` 는 「받았고 나중에」. `QUEUED` 의 뜻과 맞다 |
| `READY` 상태 | 만들지 않는다 | `INVARIANTS` §1.2 에서 `READY` 와 `QUEUED` 는 같은 순간의 두 이름이다 (`ADR-064` §1.1) |
| 임대 표면 | `GET /v1/nodes` 에 내장. 별도 `/v1/leases` 없음 | `ADR-065` 결정의 응답 모양 그대로 (`lease{run_id, not_after}`) |
| 우선순위 | 이월. FIFO 만 | 예산. 우선순위는 큐 순서에만 닿는다는 경계는 `ADR-064` §2.2 에 이미 있다 |
| 깨우기 | 임대가 지워지는 모든 지점 뒤에서 `store.WakeQueued` 를 부른다 — `postResult` · `postCancel` · `Reap`(만료 회수) · `postNodes` 의 재기동 감지(`FailRestarted` -> `SettleIfDone`) · `applyStepEffects` 의 부분 반납 · drain 해제를 받은 광고 처리 | reaper 주기에 얹으면 지연이 주기에 묶인다. 다만 임대를 지우는 자리는 셋이 아니라 여섯이다 — 만료 회수와 재기동 감지 뒤에 안 부르면 대기 Run 이 다음 종료까지 멎는다 (검토 2026-09-04 에서 넓힘) |
| 확장 opt-in | **security-baseline 켬.** resiliency-baseline · property-based-testing 끔 | 사용자 결정 2026-09-04. 취급은 3절 |
| `dhseo` 장면의 일 | 보드의 **LED 점멸 패턴을 바꾸는 ko 또는 프로그램을 작성**하고 그 보드에서 돌려 확인한다 | 사용자 결정 2026-09-04. 눈으로 성패가 보이는 일이라 CP4 의 판정이 사람 눈으로 닫힌다 |

---

# 2. 팩이 채운 권장값

1절이 닫지 않은 미정이다. 값과 근거를 적었다. **벗어나면 근거를 적는다.**

| 항목 | 권장값 | 근거 |
|---|---|---|
| 호스트 제어판 바인딩 · 포트 | `127.0.0.1:8081`. LAN 노출은 `--listen` 명시 | `127.0.0.1` 은 `ADR-063` §3.3 결정. 포트는 Mediator `:8080` 옆 |
| 중앙 현황판 위치 | Mediator 에 embed. `GET /ui/` 아래 정적 파일. Mediator 기존 포트 | 새 프로세스 없음. `constraints.md` §6 이 리버스 프록시를 뺀다 |
| 폴링 간격 | 둘 다 5초. 화면은 값 대신 「마지막 갱신 시각」을 보인다 | 광고 주기보다 짧으면 의미가 없고 길면 CP4 의 눈이 늦다 |
| `GET /v1/runs` 응답 | `{observed_at, runs:[{run_id, state, verdict, work_id, created_at, ended_at, assigned}]}`. 시간 역순. 필터 `state` · `since` · `work` · `limit`(기본 100) | `protocol/mediator-api.md` §5 가 이 자리(`?state=&since=&limit=`)를 이미 잡아 뒀다. 열은 `runs` 표의 열만. `verdict` 는 drain 으로 닫힌 `FAILED` 와 진짜 실패를 목록에서 가르는 유일한 열이다. `principal` 은 `ADR-065` 와 같은 이유로 안 낸다 |
| 페이지네이션 | 없다. `limit` 뿐 | `ADR-065` §4 「노드 수가 커지면 페이지 경계를 정한다」와 같은 조건. 이틀 안에 안 온다 |
| `QUEUED` 크래시 복구 | `QUEUED` 는 임대도 노드도 쥐지 않으므로 **재기동 후에도 `QUEUED` 가 참**이다. 기동 시 `WakeQueued` 를 한 번 부른다 (`cmd/mediator/main.go`) | `ADR-064` §8 이 결정으로 올리는 조건으로 지목한 자리. 아무것도 안 쥔 상태라 되돌릴 것이 없다 — `ASKED` 와 같은 성질 (`ADR-064` §3.4) |
| `GET /v1/nodes` 의 `draining` 필드 | 노드 항목에 `draining: "" \| "graceful" \| "at-boundary"` 를 더한다 | 중앙 배지의 자료 출처. `ADR-065` 개정이 필요하다 — `canon.md` 선행 조건 |
| 제어판 LAN 노출의 토큰 | `--listen` 이 `127.0.0.1` 밖이면 정책 파일의 `panel_token` 이 필수. 없으면 기동 거부. Mediator 토큰 재사용 안 함 | 사용자 결정 2026-09-04. Mediator 토큰은 함대 전체의 신뢰 경계라 기계 하나의 제어판에 흘리지 않는다. 이 회차에는 켜지 않는다 — 한 줄이면 된다 |
| dry-run 과 draining | `POST /v1/runs/dry-run` 은 `busy` 를 안 보듯 draining 도 안 본다. 응답 변경 없음 | dry-run 은 「능력이 있는가」만 답한다 (`ADR-014` 결정 3 · `api.go` `submit` 의 `if !dry`). draining 은 점유와 같은 축이라 같은 편에 둔다 |
| 새 패키지의 커버리지 | `.coverage-contract.yml` 의 바닥 80% 를 새 패키지에도 그대로 건다. `internal/panel` · `internal/api/ui` 는 DB 없이 도는 테스트로 채운다 | 계약 파일을 안 고친다. 두 패키지는 store 를 안 부르므로 `ENODE_TEST_DATABASE_URL` 이 필요 없다 |
| 만료 임박 임계값 | `expires_at` 까지 30초 이하면 카드가 흐려진다 | `design/README.md` §5. 값이 어느 문서에도 없어서 여기서 정한다 |
| 갱신 실패 임계값 | 연속 3회 실패하면 카운트다운을 멈추고 S1b 로 간다 | 한 번은 흔들림, 셋은 끊김 |
| S3 「현재 작업」의 출처 | Mediator 조회 — `GET /v1/nodes` 에서 자기 `node_id` 행의 `lease`, 그 `run_id` 로 `GET /v1/runs/{id}` 의 `CLAIMED` 단계. 데몬 로그를 읽지 않는다 | 관측 기준을 하나로 둔다 (`ADR-065` §3). 데몬이 상태 파일을 남기는 것은 새 기계다 |
| S3 의 Mediator 연결 상태 | 「Mediator 마지막 응답 시각」을 보인다. 실패가 이어지면 drain 통보가 늦어진다는 문구 | S1b 의 짝. 로컬 기능은 살아 있고 drain 통보만 Mediator 를 거친다 |
| 트랜스크립트 | **6절이 범위로 들여왔다.** `runner.go` 의 tee 와 링 파일과 제어판의 카드. S4 화면은 여전히 자리만이고 살아나는 것은 S3 의 카드다 | 이 행은 처음에 MVP 밖이었다. 되돌린 근거는 6.1 — 제어판에 하네스 출력이 0 줄이었다 |

---

# 3. 보안 확장 아래서의 취급

`security-baseline` 이 켜져 있다. 규칙은 **차단성**이다 — 미준수는 단계 완료를
막는다. 그래서 규칙마다 처리를 미리 적는다. **범위가 새지 않게 하는 것**이
목적이다.

```text
   SECURITY-01  전송 암호화     기존 코드 사실로 기록만.  수정 안 함
                               (평문 HTTP · sslmode=disable 은 constraints §6)
   SECURITY-03  로깅            기존 코드 사실로 기록만.  수정 안 함
                               (bootstrapToken 이 stderr 에 토큰을 찍는다)
   SECURITY-07  네트워크        기존 코드 사실로 기록만.  수정 안 함
                               (Mediator Listen :8080).  새 표면은 127.0.0.1 기본
   SECURITY-08  접근 제어       새 표면에 적용.  객체 수준 권한만 해당 없음 —
                               ADR-015 §1 (principal 은 식별이지 인증이 아니다).
                               제어판 127.0.0.1 은 명시적 public 표면이다
   SECURITY-12  인증 · 자격증명  새 표면에 적용.  토큰 하나가 신뢰 경계다 (ADR-010).
                               제어판 LAN 노출의 토큰은 2절.  비밀 관리자 도입은
                               constraints §6
```

**새 표면에는 적용한다.** `enodectl serve` 의 바인딩 기본값 · 정책 파일 권한 ·
`GET /v1/runs` `GET /v1/nodes` 의 토큰 보호 · 입력 검증(`limit` 상한 · 정책 값
enum). 그것이 이 확장을 켠 이유다.

**하지 말 것** — TLS 를 넣는 것. `cmd/mediator` · `internal/config` 를 보안
이유로 고치는 것. 둘 다 `constraints.md` §6 이고 유닛 경계 밖이다.

---

# 4. 이월 — 이번에 정하지 않는다

```text
   역할 맵 키 이름               역할 주입이 MVP 밖.  ADR-045 의 roles 와 충돌 회피는 그때
   stream-json 전환 여부         같음
   트랜스크립트 위치 · 보존       **2026-09-06 에 범위로 들어왔다 (6절)**
   우선순위 범위 · 기본값 · 문법  FIFO 만
   기아 · 대기 상한               ADR-064 §6.  상한 없음이 이번 값
   소유자 별칭 · 메모              ADR-063 §2.1 자리만
```

---

# 5. 범위에 들어온 것 (2026-09-05)

4절이 「이월」로 둔 것 중 하나가 되돌아왔다. **이월을 되돌리는 것은 진행자의
권한이고, 되돌린 근거를 여기 남긴다.**

## 5.1 Mediator MCP — 이월에서 범위로

진행자 결정. CP2 가 닫힌 뒤에 들어왔다.

**되돌린 근거 — 이월의 이유가 결합이 아니었다.**

```text
   기록된 이유       decisions.md 4절 「MCP 전송 · 토큰 전달」이 미정
                     enode-features.md 4절 「전송 · 토큰 전달 미정인 채로 둔다」

   결합은 없다       ADR-022 §9.3 이 S1 MCP 어댑터를 「독립」으로 적었다
                     「이미 있는 REST 를 도구로 감싼다 — 새 의미가 0 개다」
                     파일 겹침이 0 이다.  U-drain 도 U-제어판도 안 만지는 자리다

   그래서            막고 있던 것은 값 둘이고, 아래가 그 둘을 닫는다
```

## 5.2 닫은 값 넷

| 항목 | 값 | 근거 |
|---|---|---|
| 전송 | **stdio 하나.** streamable HTTP 를 안 연다 | 사용자 Claude 가 로컬에서 붙는 첫 실물이다. 포트도 TLS 도 안 는다 — `constraints.md` §6 이 TLS 와 리버스 프록시를 범위 밖으로 뒀으므로 HTTP 를 열면 그 문장과 정면으로 걸린다. 원격이 필요해지면 그때 더한다 |
| 토큰 전달 | **환경변수 그대로.** `ENODE_MEDIATOR` · `ENODE_TOKEN` 을 `runctl` 과 같은 규칙으로 읽는다 | 새 인증 경로가 0 이다. 도구 인자로 받으면 토큰이 대화 기록과 로그에 남고 SECURITY-12 에 걸린다. MCP 클라이언트 설정에 `env` 로 적는 것이 표준 관행이다 |
| 도구 집합 | **여덟.** `ADR-022` §9.3 의 다섯 + 이번 회차가 만든 둘 + `run.get` (5.3절) | 이 팩의 전제가 「대시보드 시각화에 쓰인 것과 **동일한 자료**가 MCP 로 나가야 한다」이다. 다섯만 내면 `GET /v1/nodes` · `GET /v1/runs` 가 MCP 로 안 나가서 현황판과 자료가 갈린다 |
| 자리와 의존 | **`runctl mcp` 하위명령 + `internal/mcp` 패키지.** JSON-RPC 2.0 을 손으로 짠다 | `internal/runctl/client.go` 가 `Submit` · `Status` · `Cancel` · `Record` · `Capabilities` 를 이미 갖고 있어 HTTP 클라이언트를 새로 안 짠다. 새 실행파일 0 이라 `packaging/` 이 안 바뀌고 「배포 모델 무변경」이 지켜진다. SDK 를 받으면 `go.mod` 직접 의존이 셋에서 넷이 되는데, 유닛마다 「새 Go 의존 0」을 게이트로 세어 온 것이 끊긴다 |

## 5.3 도구 여덟

```text
   capabilities.list    GET /v1/capabilities        Client.Capabilities
   fleet.list           GET /v1/nodes               새로 더한다 (이번 회차의 라우트)
   runs.list            GET /v1/runs                새로 더한다 (이번 회차의 라우트)
   run.submit           POST /v1/runs               Client.Submit(dry=false)
   run.plan             POST /v1/runs/dry-run       Client.Submit(dry=true)
   run.cancel           POST /v1/runs/{id}/cancel   Client.Cancel
   record.get           GET /v1/runs/{id}/record    Client.Record
   run.get              GET /v1/runs/{id}           Client.Status
```

**여덟째를 두기로 닫았다 — `run.get`.** `ADR-022` §9.3 의 목록은 `ADR-025`
보다 앞이라 `record.get` 하나로 조회를 덮었는데, `record.get` 은 종료 전이면
`409` 다(`I4`). 즉 **나머지 일곱만으로는 「지금 어떻게 돌고 있나」를 못 묻는다.**
`GET /v1/runs/{id}` 가 그것을 내고 `Client.Status` 가 이미 있다. U-MCP 의
Functional Design 첫 질문이 이것이었고 **진행자가 2026-09-05 에 여덟으로
승인했다.** 새 REST 0 · 새 의미 0 · 새 Go 의존 0 은 그대로다.

## 5.4 안 하는 것 — 이 결정이 안 넓히는 자리

```text
   트랜스크립트 구독    여전히 이월이다.  툴은 한 방에 답하는 물건이라 구독을
                       어떻게 표현할지가 별도 결정이다.  도는 동안의 출력
                       자체는 6절이 범위로 가져왔다 — 구독만 남는다
   streamable HTTP     안 연다.  전송이 하나다
   새 REST 라우트       0.  감싸는 것이지 만드는 것이 아니다
   Mediator 의 변경     0.  MCP 는 클라이언트다.  cmd/mediator 를 안 건드린다
   장면의 변경          scene-gates.md 1절의 dhseo 장면은 그대로다.  MCP 는 그
                       장면에 안 나오고 CP4 의 조건이 아니다
```

---

# 6. 범위에 들어온 것 둘째 (2026-09-06)

4절의 「트랜스크립트 위치 · 보존」이 되돌아왔다. 진행자 결정이고, CP4 가
닫힌 뒤에 들어왔다. **되돌리는 것은 진행자의 권한이고 근거를 여기 남긴다.**

## 6.1 트랜스크립트 — 이월에서 범위로

**되돌린 근거 — 제어판에 그 물건이 아예 없다.**

처음에 「단계가 끝나면 기록에 봉인되므로 절반은 있다」고 셌는데 **그것은
운영자가 `runctl record` 로 tar 를 푸는 이야기이고 제품 표면의 이야기가
아니다.** 제어판의 「로그」는 `Node.Tail` 이 읽는 데몬 로그(`<이름>.log`)이고
`design/README.md` 가 「헤더의 로그 는 데몬 로그다. 다른 물건이다」로 이미
갈라 두었다. 하네스가 뱉는 글자는 제어판에 한 줄도 안 나온다.

```text
   제어판에서 지금 보이는 것   데몬 로그 꼬리 (R24, 기본 50줄)
   제어판에서 안 보이는 것     하네스의 출력.  0 줄이다
   기록에 있는 것             단계가 끝난 뒤의 전문.  runctl record 로만 닿는다
```

**그리고 도는 동안은 어디에도 없다.** `internal/enode/runner.go` 가
`bytes.Buffer` 로 통째로 받아 `cmd.Run()` 이 끝나야 `Decode` 를 부른다.
데몬 자신도 프로세스가 끝나기 전에는 아무것도 못 본다.

## 6.2 닫은 값 셋

```text
   Q1  어디까지 실시간인가    파일 tee.  데몬이 도는 동안 stdout 을 노드의 파일에
                            흘리고 제어판이 그것을 읽는다.  정책 파일이 만든 결
                            그대로다 — 데몬과 제어판이 파일 하나로 이야기한다.
                            새 포트도 IPC 도 없고 NFR-P4 를 안 건드린다

   Q2  무엇을 흘리나         하네스의 원문 stdout.  runner.go 에서 io.MultiWriter
                            한 겹이다.  하네스 계약(--output-format)을 안 건드린다

   Q3  얼마나 남기나         고정 크기 링 파일.  기록에 이미 봉인되므로
                            두 벌로 안 남긴다
```

**값 셋이 2026-09-06 에 채워졌다.**

```text
   크기      512 KiB       80자 줄로 대략 6,400줄
   갱신      1초           화면이 열려 있을 때만.  Mediator 폴링(5초)과 별개 타이머다.
                          로컬 파일 읽기라 그 이유(R13 의 네트워크 부하)가 안 걸린다
   비우기    새 단계가 시작할 때.  단계가 끝날 때가 아니다 — 떠나 있던 사람에게도
            방금 끝난 것이 남아 있어야 한다.  S5 의 마지막 관측과 같은 태도다 (R15)
```

## 6.3 링 파일 — 윈도우가 이 설계를 정했다

파일로 이야기하면 윈도우에서 밟을 것이 넷이고, 이 회차가 그중 하나에 이미
한 번 당했다 (`92242c3` 의 `LockFileEx` 범위 잠금).

```text
   쓰는 중에 읽기     된다.  Go 의 os.OpenFile 이 윈도우에서
                     FILE_SHARE_READ | FILE_SHARE_WRITE 로 연다.  잠금을 안 건다

   회전하려고 rename  막힌다.  Go 가 FILE_SHARE_DELETE 를 안 준다 —
                     읽는 쪽이 열고 있으면 이름을 못 바꾼다
   자르면서 읽기      Truncate 와 읽기가 겹치면 읽는 쪽이 깨진 것을 본다
   끝나고 삭제        위와 같은 이유로 열려 있으면 못 지운다
```

**뒤의 셋을 아예 안 하는 모양을 고른다.**

```text
   크기      한 번 만들고 안 바뀐다
   머리      magic · 판 · 용량 · 총 쓴 바이트 · 세대
   몸통      (총량 % 용량) 자리에 WriteAt.  끝에서 감긴다
   쓰기      WriteAt 만 쓴다.  Truncate 안 함 · rename 안 함 · 삭제 안 함
   읽기      머리 · 몸통 · 머리를 다시 읽어 많이 움직였으면 한 번 더
   비우기    단계가 끝나면 머리의 총량을 0 으로 쓴다.  파일은 그대로 둔다
```

커지지 않고 이름이 안 바뀌고 안 지워지고 잠금이 없다. **윈도우와 유닉스가
같은 코드로 돈다 — 빌드 태그 쌍이 하나도 안 는다.**

권한은 정책 파일과 같은 자리에 선다 — 유닉스는 `0600`, 윈도우는 그 디렉터리의
상속 ACL 이다. **U-제어판이 그것을 실측으로 발견했으므로 여기서는 설계에
미리 적는다** (`construction/u-panel/code/code-generation-summary.md` 6.6).

## 6.4 안 하는 것 — 이 결정이 안 넓히는 자리

```text
   stream-json 전환    여전히 이월이다.  이벤트로 예쁘게 내려면 claude.go 의
                       Argv 와 Decode 를 같이 바꿔야 하고 그것은 하네스
                       스트리밍 전환이라는 별개 유닛이다.  claude.go 의 주석이
                       이미 「R4 를 열 때」로 자리를 잡아 두었다
   SSE · WebSocket     안 연다.  constraints.md 5절 그대로
   Mediator 중계        안 한다.  트랜스크립트는 노드 밖으로 안 나간다
   트랜스크립트 구독    여전히 이월이다 (5.4 와 같다)
   Mediator 의 변경     0.  cmd/mediator 도 internal/api 도 안 건드린다.
                       6.5 가 이 약속을 지키느라 tar 를 고른다
   장면의 변경          scene-gates.md 1절의 dhseo 장면은 그대로다.
                       트랜스크립트는 그 장면에 안 나오고 CP4 의 조건이 아니다
```

## 6.5 지난 작업도 제어판에서 본다 (2026-09-06 추가)

**요청** — 「한번 노드에서 진행한 작업은 제어판에서 트랜스크립트와 작업 수행
결과를 볼 수 있으면 좋겠다」. 링 파일은 지금 도는 것만 담으므로 자리가 둘로
갈린다.

```text
   지금 도는 것   노드의 링 파일.  1초 폴링.  이 유닛이 만든다
   지난 것        Mediator 가 이미 갖고 있다.  제어판이 읽어 온다
```

**Mediator 를 안 고친다.** `GET /v1/runs` 응답의 `assigned` 가 노드를 이미
싣고 있어서(`[{as, nodes: [{node, label}]}]`) 제어판이 목록을 받아 자기
`node_id` 로 거르면 된다. `node=` 필터를 새로 안 만든다. 결과도 그 목록이
이미 진다 — `state` · `ended_at` · `verdict.checks`.

**지난 트랜스크립트만 라우트가 없다.** 올리는 쪽(`PUT .../steps/{seq}/log`)만
있고 내려받는 짝이 없다. 갈래 둘 중 **제어판이 tar 를 푸는 쪽을 골랐다** —
`GET /v1/runs/{id}/record` 를 받아 `logs/NN-<step>.log` 만 꺼낸다.

```text
   고른 이유    archive/tar 는 표준 라이브러리라 새 의존이 0
                tar 는 지난 것을 눌러 볼 때만 나가고 1초 폴링과 무관하다
                이 회차가 「Mediator 는 안 건드린다」로 유닛 둘의 경계를
                세웠는데 지금 무르면 그 경계가 값을 잃는다

   안 고른 것   Mediator 에 GET /v1/runs/{run}/steps/{seq}/log 를 내는 것.
                깔끔하고 가볍지만 라우트가 18 -> 19 가 되고 6.4 를 무른다
```

**유닛을 안 가른다.** U-트랜스크립트 하나가 두 국면을 지고 CP6 도 둘을 본다.
사용자에게는 한 기능이고 화면도 같은 카드 자리다.
