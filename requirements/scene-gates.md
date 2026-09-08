# 장면 조각 게이트 — 진행의 단위는 유닛 완료가 아니라 장면 완주다

`dhseo` 장면 하나를 조각으로 잘랐다 — 장면 안 다섯과 장면 밖 넷이다.
**조각마다 실행 명령이 있고, 일정에 박히며, 조각 게이트가 초록이 아니면 다음
유닛을 착수하지 않는다.**

**CP8 은 `v1-run-dhseo-cardnews` 착수(2026-09-08)에서 더해졌다** — 온보딩
카드뉴스 + Guest Login. `dhseo` 장면에 안 나오고 함대를 안 보므로 장면 밖
넷째다. 근거 `decisions.md` 8절.

이것은 요구사항의 **수용 기준**이다. v1 의 Build and Test 는 맨 끝에 한 번
있다 — 그 구조를 그대로 두면 중간의 정합 측정이 빠지고 마지막에 데모가
깨진다. 그래서 각 유닛의 Code Generation 계획에 「이 조각의 명령을 실제로
돌린다」를 박는다.

집행자는 **그 유닛을 구현하지 않은 사람**이다. 표의 「집행자」 열이 조각마다
그것을 기계인지 사람인지 사람과 하드웨어인지로 가른다.

---

# 1. 장면 — `dhseo` 가 퇴근했고 돌아온다

```text
   ①  dhseo 의 보드가 함대에 있다.  dhseo 는 퇴근했다
   ②  taeels 가 계약을 낸다 — 「보드의 LED 점멸 패턴을 바꾸는 ko 또는
      프로그램을 작성하고 그 보드에서 돌려 확인하라」.  보드가 잡힌다
   ③  둘째 계약이 온다.  보드가 하나라 QUEUED 다.  현황판에 둘 다 보인다
   ④  dhseo 가 돌아왔다.  자기 기계의 제어판을 열어 at-boundary 로 drain 을 건다
   ⑤  도는 단계가 끝난다.  경계에서 임대가 풀린다.  현황판에 draining 이 보인다
   ⑥  dhseo 가 보드를 쓴다.  LED 가 새 패턴으로 깜빡인다 — 에이전트가 남긴 것이다
   ⑦  dhseo 가 drain 을 푼다.  대기 중이던 둘째 계약이 보드를 집는다
```

⑥ 이 이 장면의 가치다. 에이전트가 한 일이 **눈에 보이는 물건**으로 남고,
소유자가 그것을 **되찾은 자기 보드에서** 본다.

---

# 2. 조각 아홉 — 장면 안 다섯 · 장면 밖 넷

| | 조각 | 확인 (실동작) | 집행자 | 대상 | 판정 재료 | 재는 기능 | 먼저 서는 기능 | 일정 |
|---|---|---|---|---|---|---|---|---|
| CP0 | 기동이 안 깨졌다 | `scripts/testdb.sh` 로 Postgres 를 세운 뒤 기존 테스트 전부 초록 (DB 없이 돌리면 `cmd/mediator` · `internal/store` 가 하드 실패하고 `internal/api` 는 102개를 조용히 건너뛴다). 커버리지 · glyphscan · 포맷 · vet · 크로스 빌드 · 심볼 상한 통과. 기존 라우트 15개 회귀. 구조 불변식(`constraints.md`)은 CP0 에 검사할 패키지가 아직 없다 — 그 검사는 `internal/panel` 을 만드는 유닛이 함께 낸다 | 기계 | 스크래치 | 워킹트리 · Postgres | 3.1.1 의 바닥 | (없음) | 1일차 오전 |
| CP1 | 보인다 | curl 로 `GET /v1/runs` · `GET /v1/nodes` 가 실데이터 JSON. 임대(`lease.not_after`)와 `draining` 이 실려 있다 | 사람 | 스크래치+함대 | 노드 여럿 · 임대 하나 · draining 하나 · 상태 넷의 Run | 3.1.1 | 3.1.1 | 1일차 정오 |
| CP2 | 기다린다 | 노드 하나에 Run 둘을 던진다. 둘째가 `202` 와 `QUEUED`. 첫째가 끝나면 둘째가 `RUNNING` 으로 간다. 함대에 없는 능력은 여전히 `422` -> `FAILED` | 사람 | 함대 | 보드 하나 · 계약 셋 (하나는 갈래가 있다) | 3.2.1 | 3.1.1 | 1일차 오후 |
| CP3 | 돌려받는다 | `at-boundary` 로 drain. 도는 단계가 끝난 경계에서 임대가 풀린다. 풀린 노드는 후보에서 빠져 있다. 놓인 Run 은 `FAILED` 이고 `verdict` 가 `drain:<node_id>` 를 진다. drain 을 풀면 대기 Run 이 집는다 | 사람 | 함대 | 단계 둘 이상인 계약 · 정책 파일 | 3.2.2 | 3.2.1 | 2일차 오전 |
| CP4 | 한 장면 | 1절 ① ~ ⑦ 을 끝까지. 현황판과 제어판이 같은 사실을 보인다. LED 가 새 패턴이다 | 사람+하드웨어 | 함대 | 보드 · LED · 사람의 눈 | 전부 | 전부 | 2일차 정오 — 이후는 광내기 |
| CP5 | 감싼다 | `runctl mcp` 를 stdio 로 띄우고 `tools/list` 가 열을 낸다. `run.submit` 으로 낸 Run 이 현황판에 그대로 뜬다 — 같은 자료다. `fleet.list` 가 `GET /v1/nodes` 와 글자까지 같다 | 사람 | 스크래치 | MCP 클라이언트 설정 | 3.3.1 | 3.1.1 | 장면 밖 — CP1 뒤 아무 때나 |
| CP6 | 보인다 (하네스) | **두 국면이다.** ① 도는 것 — 계약을 던져 놓고 제어판을 열면 **단계가 끝나기 전에** 카드에 하네스 출력이 흐른다. 상한을 넘겨도 안 깨지고 오래된 줄부터 밀리며, **다음 단계가 첫 글자를 쓸 때** 갈린다. ② 지난 것 — 이 노드가 한 Run 목록이 뜨고, 하나를 누르면 결과(`verdict.checks`)와 봉인된 트랜스크립트가 보인다. 데몬 로그 카드는 그대로 있다 — 둘이 다른 물건임이 화면에서 보인다 | 사람 | 함대 | 도는 Run 하나 · 지난 Run 하나 | 3.1.3 | 3.1.2 | 장면 밖 — CP4 뒤 아무 때나 |
| CP7 | 물어본다 | 되묻기가 걸린 Run 을 만든다. 단계가 `ASKED` 가 되고 현황판이 그 Run 을 「사람을 기다림」으로 보인다 — `임대 중` 과 다르게 보이고 카운트다운이 `not_after` 가 아니다. `runctl asks` 가 같은 질문을 내고 `runctl answer` 로 답하면 Run 이 이어진다. MCP `asks.list` 가 `GET /v1/asks` 와 글자까지 같고 `run.answer` 로도 답해진다 | 사람 | 스크래치 | `ask` 단계가 있는 계약 · MCP 클라이언트 설정 | 3.2.3 | 3.1.1 · 3.3.1 | 장면 밖 — CP5 뒤 아무 때나 |
| CP8 | 배경을 준다 | 랜딩 화면(`GET /ui/`)을 연다. Guest Login 버튼이 관리자 토큰 입력과 분리돼 보인다. 처음 누르면 카드뉴스가 뜨고(데모 현황판이 같이 안 보인다), 끝까지 넘기면 데모 현황판 자리로 간다(카드뉴스 요소가 하나도 안 남는다). 로컬 저장소를 지우지 않고 다시 누르면 카드 없이 곧장 그 자리로 간다 | 사람 | 스크래치 | 브라우저 하나 (함대 불필요) | 3.4.1 | (없음) | 장면 밖 — CP0 뒤 아무 때나 |

**CP4 가 2일차 정오에 있는 것이 이 팩의 핵심이다.** 맨 끝에 두면 실패가
맨 끝에 온다.

표의 「재는 기능」은 그 조각이 **재는** 것이고, 「먼저 서는 기능」은 재기 전에
이미 초록이어야 하는 자리다. 아래 세 문단이 그 구분을 장면 밖 셋에 적용한다.

**CP6 도 그 장면 밖이다.** 1절의 `dhseo` 장면에 트랜스크립트가 안 나오므로
**CP4 의 조건이 아니고, CP6 이 빨개도 CP4 는 초록이다.** 먼저 서는 것이
호스트 제어판(3.1.2)의 화면과 `internal/enode` 의 실행 경로뿐이라 CP4 가
닫힌 뒤 언제든 돈다.

**CP5 는 그 장면 밖이다.** 1절의 `dhseo` 장면에 MCP 가 안 나오므로 **CP4 의
조건이 아니고, CP5 가 빨개도 CP4 는 초록일 수 있다.** 일정에 자리를 안 주는
이유도 같다 — 먼저 서는 것이 CP1 의 라우트(3.1.1)뿐이라 CP1 이 초록인
순간부터 언제든 돌고, drain(3.2.2) · 호스트 제어판(3.1.2)과 겹치는 자리가
없어 병렬로 간다.

**CP7 도 그 장면 밖이다.** 1절의 `dhseo` 장면에 되묻기가 안 나오므로 **CP4 의
조건이 아니고, CP7 이 빨개도 CP4 는 초록이다.** 표면이 셋(중앙 · `runctl` · MCP)이라
CP5 에 넣지 않고 조각을 따로 뒀다 — 넣으면 「재는 기능」 열이 흐려진다.

**CP8 도 그 장면 밖이다.** 1절의 `dhseo` 장면에 온보딩이 안 나오므로 **CP4 의
조건이 아니고, CP8 이 빨개도 CP4 는 초록이다.** 「먼저 서는 기능」이 없다 —
함대도 다른 어떤 조각의 산출물도 필요 없이 `internal/api/ui` 하나로 선다.
CP0(기동이 안 깨졌다) 뒤라면 나머지 조각과 무관하게 아무 때나 돈다.

## 2.1 화면 검증은 API 검증과 따로 적는다

API 로 통과되는 어휘(「임대가 풀린다」)는 화면 없이도 초록이 된다. 그래서 화면이
있는 조각은 **눈으로 보는 항목**을 따로 둔다. `design/README.md` 의 화면 번호를 쓴다.

```text
   CP1   S0   /ui/ 를 토큰 없이 연다.  정적 파일은 무인증이고 S0 이 뜬다
         S0b  틀린 토큰 -> 401 문구.  현황판으로 안 넘어간다
         S0   탭을 닫았다 열면 다시 묻는다 (세션 저장소)
         S1   카드가 노드 수만큼 있다.  임대된 카드에 그 임대가 언제까지인지가
              있다 — 노드가 살아 있으면 그 값이 주기적으로 뒤로 밀린다.
              채워지는 진행 막대로 그리지 않았다
              draining 카드에 배지.  마지막 갱신 시각이 있다
         S1b  Mediator 를 멈추고 15초 뒤 카운트다운이 멎고 실패 표시가 뜬다
   CP2   S1   Run 목록에 QUEUED 행이 있고 어느 카드에도 안 얹혀 있다
              그 행을 누르면 무슨 능력을 기다리는지 보인다
         S2   갈래가 있는 Run 을 열면 안 간 쪽이 SKIPPED 로 남아 있고
              chosen 이 true / false 로 갈려 보인다
   CP3   S1   drain 을 건 노드 카드에 배지가 뜬다.  두 국면의 갈림은
              CP4 가 잰다 — 제어판이 아직 없다
   CP5   —    화면 없음.  MCP 는 화면을 안 만든다 — 사용자 Claude 가 화면이다
   CP4   S3   drain 을 걸기 전 — 걸기 버튼과 모드 선택
              Mediator 마지막 응답 시각이 있다
         S3b  건 뒤 — 현재 모드 배지 · 「지금 도는 작업 없음」 · 풀기 버튼 (C3)
              걸기 전과 건 뒤는 서로 다른 장이다 — 한 장에 둘 다 못 그린다
              조작 넷이 전부 있다 — status · start · stop · logs.
              도는 노드에서는 start 자리에 재시작이 보인다 (stop 뒤 start 이고
              새 하위명령이 아니다)
              stop · 재시작은 누르기 전에 그 Run 을 어떻게 끝내는지 말한다
         S1   배지가 「draining · 진행 중」과 「draining · 대기 중」으로 갈린다 (C4 · C5)
         S5   멈춘 노드에서 start 가 보이고 누르면 뜬다
   CP6   S3   트랜스크립트 카드에 하네스 출력이 흐른다.  데몬 로그 카드가
              그 옆에 그대로 있고 둘이 다른 물건임이 화면에서 보인다
              지난 Run 목록에 이 노드가 한 것만 있다.  하나를 누르면
              결과(verdict.checks)와 봉인된 트랜스크립트가 열린다
              그 목록은 S3 안의 카드다.  S4 는 자리만 그대로 둔다
   CP7   S1   되묻기가 걸린 Run 의 카드가 「사람을 기다림」으로 갈린다.
              임대 중과 다르게 보이고 카운트다운이 not_after 가 아니다
              카드가 누가 답할 수 있는지를 적고, 칠 명령은 Run 목록 아래
              안내 상자가 진다
   CP8   —   화면은 있으나 S0 ~ S5 번호를 안 쓴다 (design/ 의 아홉 장 밖).
              랜딩에 Guest Login 버튼이 관리자 토큰 입력과 분리돼 보인다
              카드뉴스가 독립 화면으로 뜨고 전환에 애니메이션이 있다
              끝까지 넘기면 카드뉴스 요소가 하나도 안 남고 데모 현황판
              placeholder 로 넘어간다
              로컬 저장소를 지운 뒤 다시 열면 카드뉴스가 다시 뜬다
```

**화면이 있는 유닛의 완료 조건은 그 화면의 버튼을 전부 나열한다.** 「누르면 걸린다」
만으로는 「풀기」가 빠진 채 통과된다. 같은 이유로 CP4 는 조작 넷을 센다 —
`start` 하나만 보면 `stop` 이 무엇을 남기는지 아무도 안 본 채 통과된다.

---

# 3. 조각마다의 명령

값은 시작값이다. 유닛의 Code Generation 계획이 이것을 실제 스크립트로 굳힌다.

```text
   게이트를 돌리기 전에 (조각마다 다시 안 한다)

     eval "$(scripts/testdb.sh)"     CP0 의 테스트와 커버리지가 이것을 요구한다
     export M=http://<Mediator 주소>:8080     0.0.0.0 에 띄운다.  127.0.0.1 이면
                                             밖에서 안 붙는다
     export T=<bootstrap 토큰>
     jq 가 필요하다

   함대는 45초마다 광고를 다시 보내는 노드가 있어야 산다 — 광고는
   renew_seconds 60 x not_after_factor 3 = 180초에 만료된다.  만료되면
   CP1 의 jq 가 빈 출력을 내는데 그것은 초록이 아니라 잰 것이 없는 것이다.
```

```text
   CP0   eval "$(scripts/testdb.sh)"          <- 먼저 친다.  안 치면 아래가 빨갛다
         go test ./... && go vet ./... && go run ./scripts/glyphscan.go
         test -z "$(gofmt -l .)"
         test -z "$(git status --porcelain)"   시험이 추적 파일을 고치지 않았다
         go test ./... -count=1 -coverpkg=./... -coverprofile=/tmp/cover.out
           그리고 ci.yml 의 커버리지 스텝 awk (패키지별 하한 80)
         GOOS=windows GOARCH=amd64 go build -o /tmp/enodectl.exe ./cmd/enodectl
           go tool nm /tmp/enodectl.exe | grep -c ' T net/http\.'   -> 50 이하
         curl 로 기존 라우트 하나 — GET /v1/capabilities  (Mediator 는 위 블록이 세운다)

   린트는 이 저장소에서 차단이 아니다 (ci.yml:117 continue-on-error).  지적
   20건이 남아 있는 기준선 단계이므로 CP0 의 조건에서 뺀다.  돌려서 수가
   20 에서 늘었는지만 본다 — 늘었으면 이 유닛이 늘린 것이다.

   CP1   curl -H "Authorization: Bearer $T" $M/v1/nodes | jq '.nodes[] | {label, lease, draining}'
         curl -H "Authorization: Bearer $T" "$M/v1/runs?limit=5" | jq '.runs[] | {run_id, state}'
         브라우저로 $M/ui/ 를 연다 -> S0.  틀린 토큰 -> 401.  맞는 토큰 -> S1
         탭을 닫았다 다시 연다 -> S0 이 다시 뜬다 (세션 저장소)

   CP2   계약 셋은 runctl example 로 만들어 requires 를 고친다
         runctl submit a.json                     -> RUNNING
         curl -i -X POST $M/v1/runs -d @b.json    -> 202     (runctl 은 성공
                                                    응답의 코드를 안 찍는다)
         runctl status <b>                        -> QUEUED
         a 가 끝난 뒤 runctl status <b>            -> RUNNING
         runctl submit c.json (없는 능력)          -> stderr 에 422
         runctl status <c>                        -> FAILED
         QUEUED 행을 누른다 -> 무슨 능력을 기다리는지 보인다
         계약 셋 중 하나는 갈래가 있다 -> 안 간 쪽이 SKIPPED 이고 chosen 이 갈린다


         주의 — 게이트용 Mediator 를 시험 DB 와 공유하지 않는다.  공유하면
         광고가 살아 있어 「함대에 없다(422)」가 조용히 201 이 된다
         (scripts/testdb.sh 의 주석.  실제로 밟았다)

   CP3   단계 둘 이상인 계약을 돌린다 — 경계가 있어야 at-boundary 를 잰다
         정책 파일에 at-boundary 를 쓴다 (제어판 버튼은 CP4).
           그 파일의 경로와 형식은 drain(3.2.2)의 Functional Design 이 정한다
           (decisions.md 1절) — 이 줄은 그때 굳는다
         GET /v1/nodes 에 draining: at-boundary
         도는 단계가 끝난 뒤 GET /v1/nodes 에 lease 없음
         새 계약을 내면 그 노드로 안 간다 — 202 로 받고 QUEUED 에 선다
           (409 는 매처 안의 상황 이름으로만 남는다.  mediator-api.md §1.2)
         at-boundary 로 놓인 Run 을 GET /v1/runs 에서 본다
           -> FAILED 이고 verdict 가 drain:<node_id> 를 진다.
              끝난 단계의 산출은 record 에 남아 있다 (I4)
         drain 을 푼다 -> 대기 Run 이 RUNNING

   CP4   1절의 순서 그대로.  사람이 보드 앞에 선다
         그리고 조작 넷을 전부 누른다 — status · start · stop · logs
         stop 은 도는 Run 을 먼저 cancel 한다 (decisions.md 7.2) ->
           GET /v1/runs/{id} 의 verdict 에 cancelled by 가 남는다.
           lease expired: renewal stopped 가 아니다

   CP5   runctl mcp 를 stdio 로 띄운다
         tools/list          -> 열 (되묻기 둘을 포함한다)
         run.submit          -> 낸 Run 이 현황판에 그대로
         fleet.list          -> GET /v1/nodes 와 글자까지 같다

   CP6   계약을 던지고 제어판을 연다 -> 단계가 끝나기 전에 출력이 흐른다
         512 KiB 를 넘긴다 -> 안 깨지고 오래된 줄부터 밀린다
         다음 단계의 첫 글자 -> 카드가 갈린다
         지난 Run 을 누른다 -> 결과와 봉인된 트랜스크립트

   CP7   ask 단계가 있는 계약을 낸다 -> 그 단계가 ASKED 가 된다
         현황판 -> 그 Run 의 카드가 「사람을 기다림」이다.  임대 중이 아니다
         curl -H "Authorization: Bearer $T" $M/v1/asks \
           | jq '.asks[] | {run_id, seq, prompt, can_answer, deadline}'
         runctl asks                    -> 같은 질문
         runctl answer <run> <seq> --set verdict=approve  -> Run 이 이어진다
         runctl mcp 로  asks.list       -> GET /v1/asks 와 글자까지 같다
                       run.answer      -> 같은 답이 들어간다
         답한 뒤 GET /v1/asks 가 비어 있다 — 답한 것은 봉인에 있다

   CP8   함대도 Mediator 토큰도 필요 없다.  정적 파일 서버(mediator 나
         standalone 바이너리)만 띄우면 된다
         브라우저 사생활 창(로컬 저장소가 비어 있다)으로 $M/ui/ 를 연다
         Guest Login 버튼이 관리자 토큰 입력과 시각적으로 분리돼 보인다
         Guest Login 을 누른다 -> 카드뉴스가 뜬다.  주소창이 카드뉴스
           경로다 (데모 현황판 경로가 아니다)
         카드를 끝까지 넘긴다 -> 데모 현황판 자리로 이동한다.  DOM 에
           카드뉴스 요소가 남아 있지 않다 (개발자 도구로 확인)
         탭을 새로고침하고 다시 랜딩에서 Guest Login 을 누른다 -> 카드뉴스
           없이 곧장 데모 현황판 자리로 간다
         브라우저 저장소에서 enode.guest.onboarded 를 지운다 -> 다시
           누르면 카드뉴스가 다시 뜬다
```

---

# 4. 게이트가 빨간 채로 다음 유닛을 착수하면

착수하지 않는다. 그것이 규칙이다. 다만 **빨간 이유가 그 유닛 밖**(보드가
안 켜진다 · Mediator DB 가 죽었다)이면 진행자가 그 사실을 그 회차의
`aidlc-docs/<브랜치 이름>/audit.md` 에 적고 조각을 **보류**로 표시한
뒤 넘어간다. 보류는 통과가 아니다 — CP4 전에
전부 초록이어야 한다.
