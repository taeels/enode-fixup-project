# 핸드오프 — panel (W3 · CP4) 과 transcript (W4 · CP6)

작성 2026-09-09 · 담당 handle **nacl1119**(문태호) · 넘기는 곳 **AWS Bedrock 위의 Claude**.

이 문서 하나로 착수할 수 있게 썼다. 앞선 대화의 맥락을 갖고 있지 않다고 보고,
딛을 정본의 경로를 전부 적었다. **정본과 이 문서가 어긋나면 정본이 이긴다** —
이 문서는 길잡이지 결정이 아니다.

---

# 1. 저장소와 규약 — 먼저 읽는다

```text
   저장소      github.com/taeels/enode-fixup-project
   Go 모듈     github.com/taeels/enode
   기준 시각    2026-09-09.  main 은 50af6cf
```

착수 전에 이 넷을 읽는다. 순서대로다.

```text
   CLAUDE.md                              문서 루트 layering 과 AI-DLC 진입점
   CONVENTIONS.md                         표기 · 언어 · 커밋 규약
   .aidlc/aidlc-rules/aws-aidlc-rules/    AI-DLC v1.0.1 워크플로 정본
   aidlc-docs/construction-roster.md      배정 · 웨이브 · 접점 · 병합 정책
```

## 1.1 표기 — 강조는 마크다운 굵게로만

문서에서 강조는 굵게 하나뿐이다. 특수문자로 어절을 감싸지 않는다.
**코드에는 장식 문자를 아예 넣지 않는다** — 주석 · 문자열 · 로그 · 테스트 메시지 ·
설정 어디에도 안 넣는다. 커밋 메시지에도 안 넣는다.

CI 가 이것을 센다. `scripts/glyphscan.go` 가 비테스트 Go 파일의 문자열 리터럴을
AST 로 걸어 장식 문자를 찾고, `.github/workflows/ci.yml:93` 이 그것을 돌린다.
주석은 안 본다 — 이 저장소의 주석 251줄이 설계 논거를 표와 기호로 적고 있어서다.

## 1.2 언어 — 밖으로 나가느냐로 가른다

```text
   영어로     에러 문자열 · 로그 메시지 · CLI 출력 · 테스트 이름과 실패 메시지 ·
             테스트 픽스처 문자열 · 에이전트 프롬프트

   한국어로   주석(godoc 포함) · 커밋 메시지 · testdata 안의 기록 ·
             aidlc-docs 아래의 모든 문서
```

기준은 「코드냐 아니냐」가 아니라 **「밖으로 나가느냐」**다. 이 저장소는
에이전트를 돌리는 제품이라, 프롬프트가 한국어면 제품이 실행 중에 그 언어를
에이전트에게 다시 주입한다. 주석은 하네스로 실려 나가지 않으므로 예외다.

## 1.3 커밋 · 브랜치 · 병합

```text
   브랜치     unit/panel · unit/transcript 를 main 에서 딴다
   커밋       단계 승인마다.  산출물이 나온 자리에서 역사에 넣는다
   병합       그 유닛의 장면 게이트가 초록이 된 뒤에만.  PR 로 main 에
   푸시       git push -u origin unit/<유닛>
```

**커밋에 안 싣는 것** — 루트 `aidlc-docs/aidlc-state.md` · 루트 `aidlc-docs/audit.md` ·
`design/*.pen`. 이 셋은 진행자가 병합 뒤에 정리한다. 유닛의 기록은
`aidlc-docs/nacl1119/` 아래 자기 자리에 적는다.

---

# 2. 문서 루트 — 어디에 쓰나

Construction 산출물의 문서 루트는 담당 handle 이다.

```text
   aidlc-docs/nacl1119/aidlc-state.md                      내 상태.  단계마다 갱신
   aidlc-docs/nacl1119/audit.md                            내 감사 로그.  이어 붙인다
   aidlc-docs/nacl1119/construction/plans/                 단계별 계획 파일
   aidlc-docs/nacl1119/construction/panel/                 panel 산출물
       functional-design/ · nfr-requirements/ ·
       nfr-design/ · code/
   aidlc-docs/nacl1119/construction/transcript/            transcript 산출물
```

`audit.md` 는 **절대 통째로 덮어쓰지 않는다** — 읽고 이어 붙인다. 덮어쓰면
기록이 두 벌이 된다. 모양은 AI-DLC 정본의 감사 로그 형식을 따른다
(타임스탬프 ISO 8601 · 사용자 입력 원문 그대로 · 요약 금지).

먼저 나온 유닛의 산출물이 좋은 본보기다 — `aidlc-docs/taeels/construction/obs/`
와 `aidlc-docs/taeels/construction/plans/obs-functional-design-plan.md` 를 보면
이 저장소가 각 단계에서 무엇을 어느 깊이로 내는지가 보인다.

---

# 3. 지금 상태 — 의존이 닫혔다

`main` 에 이미 들어와 있는 것.

```text
   PR #4   obs             unit/obs         MERGED  2026-09-08T15:21Z
   PR #5   queue           unit/queue       MERGED  2026-09-09T00:59Z
   PR #6   ui (실 함대)     unit/runixs-ui   MERGED  2026-09-09T01:32Z
   PR #7   drain           unit/drain       MERGED  2026-09-09T02:25Z
```

panel 의 의존은 **obs**(착수 · `runctl.Client.Nodes`)와 **drain**(완료 · 정책 파일
형식) 둘이고 **둘 다 닫혔다**. 그러므로 W3 착수 조건이 서 있다.
transcript 의 의존은 **panel**(카드가 사는 화면)과 **obs**(GET /v1/runs)다 —
panel 이 병합된 뒤에 착수한다.

## 3.1 drain 이 남긴 것 — panel 이 딛는 자리

`internal/enode/policy.go` 가 정책 파일의 정본이다. 제어판의 drain 토글은
**같은 파일을 쓴다**. 새 형식을 만들지 않는다.

```text
   경로     <설정파일 디렉터리>/<stem>.policy.yaml
            PolicyPath(configPath) 가 그 규칙을 진다
   내용     키 하나 — drain.  모르는 키는 무시한다
            (제어판의 panel_token 이 뒤에 같은 파일에 앉는다)
   읽기     데몬이 광고 직전마다 읽는다.  캐시 없음
   권한     유닉스 0600.  윈도우는 그 디렉터리의 상속 ACL
```

**`Policy` 구조체에 `panel_token` 을 더하는 것이 panel 유닛의 몫이다.** 정책
파일에 키가 하나 더 앉을 뿐이고, 데몬 쪽 `policyReader` 는 모르는 키를 이미
무시하므로 데몬을 안 건드린다.

## 3.2 obs 가 남긴 것

```text
   internal/runctl/client.go   Nodes(ctx) json.RawMessage    원문 JSON — panel · mcp 공유
                               Runs(ctx, RunsQuery)          transcript 의 지난 작업
                               Capabilities(ctx)             탐지 능력
                               Cancel(ctx, runID)            panel 의 stop 이 먼저 부른다
                               Record(ctx, runID, w)         transcript 의 봉인 트랜스크립트
   internal/api/api.go:96      GET /v1/capabilities
```

---

# 4. 착수 전에 닫아야 하는 것 — ADR-068 (블로킹)

**제어판이 도는 데몬의 `Capabilities{Caps, At}` 를 읽는 계약이 아직 안 정해졌다.**

```text
   여는 곳    aidlc-docs/v1-run-dhseo/inception/application-design/unit-of-work.md:201
              aidlc-docs/construction-roster.md 6절
              aidlc-docs/nacl1119/aidlc-state.md 열린 미정
   닫는 곳    requirements/decisions.md 에 진행자가 행을 더한다 — 2026-09-09 현재 없음
   시점       panel Functional Design 전
```

무엇이 걸려 있나 — 탐지 능력 값은 데몬 메모리의 `Capabilities{Caps, At}` 에만
있고, `ADR-068` 이 그 `At`(마지막 탐지 시각)을 광고에 싣는 것을 거절했다
(`aidlc-docs/taeels/construction/obs/functional-design/business-rules.md:249`).
제어판의 「탐지 능력 읽기 전용」 카드가 그 값을 어디서 읽을지가 안 정해진 것이다.

**Functional Design 첫머리에 질문 파일로 내고 사람의 답을 기다린다.** 갈래를
직접 정하지 않는다. 질문 파일 형식은 AI-DLC 정본의
`common/question-format-guide.md` 를 따른다 (A · B · C · D · E 와 `[Answer]:` 태그).
답이 오기 전까지 겉면에 자리만 두고 나머지를 설계한다 — 유닛 정본이 그렇게 적었다.

---

# 5. panel (W3 · CP4)

**목적** — 노드 소유자가 자기 기계에서 자기 노드 하나를 보고 통제한다.

정본은 `aidlc-docs/v1-run-dhseo/inception/application-design/unit-of-work.md` 5절이다.
아래는 그 요약이고, 어긋나면 정본이 이긴다.

## 5.1 단계 순서

각 단계 끝에 **2지 선택 완료 메시지**(변경 요청 / 다음 단계로)를 내고 승인을
기다린다. 3지 이상 메뉴를 만들지 않는다 — AI-DLC 정본이 금지한다. 승인이 오면
그 자리에서 커밋한다.

```text
   1  Functional Design       internal/proc 추출 경계 · internal/panel 겉면 ·
                              정책 파일 정합 · ADR-068 질문 파일

   2  NFR Requirements        심볼 상한 · 커버리지 · 바인딩과 토큰

   3  NFR Design              위를 설계로

   4  Infrastructure Design   건너뛰기를 제안한다 — 로컬 프로세스뿐이고
                              새 클라우드 자원이 없다.  사람이 판단한다

   5  Code Generation         Part 1 계획(체크박스)을 승인받고 Part 2 실행

   6  게이트 CP4              재고 초록이면 PR
```

## 5.2 낼 것

```text
   internal/proc          processAlive(pid) · signalStop(pid) · ownsConfig(path) 를
                          cmd/enodectl/proc_unix.go · proc_windows.go 에서 내린다.
                          빌드 태그 짝 그대로.  net/http 를 안 쓴다

   internal/panel         제어판 서버.  127.0.0.1:8081 기본
                          신원 표시 · 탐지 능력 읽기 전용 · 현재 작업(Client.Nodes) ·
                          drain 걸기와 모드(graceful · at-boundary)와 풀기(정책 파일 쓰기) ·
                          프로세스 제어 status · start · stop · logs ·
                          Mediator 마지막 응답 시각

   cmd/enodectl serve <name>   enode 제어판 프로세스를 exec 위임한다
   cmd/enode 제어판 하위명령    internal/panel 을 net/http 로 띄운다
   경계 검사 테스트             internal/panel 을 처음 만드는 유닛이 낸다
```

겉면의 시작 모양 (정본 unit-of-work.md 5절).

```go
type Config struct { Node, Listen, MediatorBase, Token, PanelToken string }
func New(cfg Config) (*Server, error)   // LAN 인데 PanelToken 비면 error
func (s *Server) Handler() http.Handler
```

## 5.3 안 넘으면 안 되는 선

```text
   임포트 경계    panel -> store 금지 · panel -> api 금지 · enode -> panel 금지
                 cmd/enode -> panel 은 허용.  테스트로 잡는다

   심볼 상한      enodectl.exe 의 net/http T 심볼 <= 50 · crypto/tls <= 10
                 .github/workflows/ci.yml:484 가 재고 continue-on-error 가 없다.
                 지금 값은 6 과 1 이다
```

**`serve` 를 exec 위임하는 이유가 이 상한이다.** 그 실행파일 안에서 HTTP 서버를
직접 부르면 앞선 회차 실측으로 39배 · 14배가 나서 게이트가 빨개진다
(`requirements/decisions.md` 2절 「제어판 서버 위치」 행). `cmd/enodectl/setup.go`
가 같은 모양을 이미 보인다 — 그것을 본뜬다.

```text
   바인딩       127.0.0.1:8081.  Mediator :8080 옆.  ADR-063 3.3 결정
   LAN 노출     --listen 이 127.0.0.1 밖이면 정책 파일의 panel_token 이 필수.
               없으면 기동 거부.  Mediator 토큰을 재사용하지 않는다 —
               그쪽은 함대 전체의 신뢰 경계라 기계 하나의 제어판에 안 흘린다.
               이 회차에 켜지는 않는다.  거부 경로만 선다

   stop 의 순서  POST /v1/runs/{id}/cancel 을 먼저 부르고 데몬을 끈다.
               Mediator 가 안 닿으면 확인 문구가 「임대 만료로 죽는다」를 말하고
               그대로 끈다.  아무것도 안 하면 그 Run 이
               lease expired: renewal stopped 로 죽어 소유자가 껐다는 사실이
               기록에 안 남는다 (decisions 6.5 표)
```

## 5.4 게이트 CP4 — 무엇을 재나

`requirements/scene-gates.md` 43행 · 90~99행이 정본이다.

```text
   S3    drain 을 걸기 전 — 걸기 버튼과 모드 선택.  Mediator 마지막 응답 시각이 있다
   S3b   건 뒤 — 현재 모드 배지 · 「지금 도는 작업 없음」 · 풀기 버튼
         걸기 전과 건 뒤는 서로 다른 장이다 — 한 장에 둘 다 못 그린다
         조작 넷이 전부 있다 — status · start · stop · logs
         도는 노드에서는 start 자리에 재시작이 보인다 (stop 뒤 start 이고
         새 하위명령이 아니다)
         stop 과 재시작은 누르기 전에 그 Run 을 어떻게 끝내는지 말한다
   S1    배지가 「draining · 진행 중」과 「draining · 대기 중」으로 갈린다
   S5    멈춘 노드에서 start 가 보이고 누르면 뜬다
```

**화면이 있는 유닛의 완료 조건은 그 화면의 버튼을 전부 나열한다.** 「누르면
걸린다」만으로는 「풀기」가 빠진 채 통과된다. 같은 이유로 조작 넷을 센다 —
`start` 하나만 보면 `stop` 이 무엇을 남기는지 아무도 안 본 채 통과된다.

여기에 더해 — 경계 검사 테스트 초록 · 심볼 상한 재측정 · `internal/panel`
커버리지 80%.

**CP4 는 2일차 정오에 있다.** 이 팩에서 가장 앞에 당겨 둔 게이트이므로 늦추지
않는다. CP5 · CP6 · CP7 은 장면 밖이라 빨개도 CP4 는 초록이다.

---

# 6. transcript (W4 · CP6) — panel 병합 뒤

**목적** — 하네스가 지금 뱉는 글자와 지난 작업의 결과 · 봉인 기록을 제어판에
보인다. **DB 를 안 만진다.**

정본은 unit-of-work.md 6절 · `requirements/decisions.md` 6절 전체다.

## 6.1 단계 순서

panel 과 같다. Functional Design 에서 **링 파일 로직(머리 · 몸통 · 감김 ·
비우기)을 확정**한다 — `decisions.md` 6.3 이 그 모양을 이미 좁혀 놨다.

## 6.2 낼 것

```text
   internal/enode/runner.go    에이전트 단계의 하네스 stdout/stderr 를 링 파일에 tee
   internal/enode/claim.go     명령 단계도 같이.  io.MultiWriter 한 겹이다
                               하네스 계약(--output-format)을 안 건드린다
                               drain 이 만진 advertise.go · policy.go 와 다른 파일이다

   internal/panel              트랜스크립트 카드.  GET /api/transcript?node=<이름> 을
                               1초 폴링.  데몬 로그 카드 옆에 나란히 — 다른 물건이다

   지난 작업                    GET /v1/runs 를 자기 node_id 로 걸러 목록.
                               누르면 verdict.checks 와
                               GET /v1/runs/{id}/record 의 tar 에서 logs/NN-*.log
```

## 6.3 링 파일 — 윈도우가 이 설계를 정했다

`decisions.md` 6.2 · 6.3 이 값과 모양을 이미 닫았다. **다시 논의하지 않는다.**

```text
   크기      512 KiB.  80자 줄로 대략 6,400줄
   갱신      1초.  화면이 열려 있을 때만.  Mediator 폴링(5초)과 별개 타이머다
   머리      magic · 판 · 용량 · 총 쓴 바이트 · 세대
   몸통      (총량 % 용량) 자리에 WriteAt.  끝에서 감긴다
   쓰기      WriteAt 만.  Truncate 안 함 · rename 안 함 · 삭제 안 함
   읽기      머리 · 몸통 · 머리를 다시 읽어 많이 움직였으면 한 번 더
   비우기    새 단계가 시작할 때 머리의 총량을 0 으로 쓴다.  파일은 그대로 둔다
            단계가 끝날 때가 아니다 — 떠나 있던 사람에게도 방금 끝난 것이 남아야 한다
   권한      유닉스 0600 · 윈도우는 그 디렉터리의 상속 ACL
```

이 모양을 고른 이유는 윈도우다. Go 가 `FILE_SHARE_DELETE` 를 안 줘서 읽는 쪽이
열고 있으면 rename 도 삭제도 막히고, `Truncate` 와 읽기가 겹치면 읽는 쪽이 깨진
것을 본다. 뒤의 셋을 아예 안 하는 모양이라 **윈도우와 유닉스가 같은 코드로
돌고 빌드 태그 쌍이 하나도 안 는다.**

## 6.4 게이트 CP6 — 두 국면이다

```text
   도는 것    계약을 던져 놓고 제어판을 열면 단계가 끝나기 전에 카드에 출력이 흐른다.
             상한을 넘겨도 안 깨지고 오래된 줄부터 밀린다.
             다음 단계가 첫 글자를 쓸 때 갈린다

   지난 것    이 노드가 한 Run 목록이 뜨고, 하나를 누르면
             결과(verdict.checks)와 봉인된 트랜스크립트가 보인다

   둘 다      데몬 로그 카드가 그 옆에 그대로 있고
             둘이 다른 물건임이 화면에서 보인다
```

`internal/enode` 커버리지 80%. S4 화면은 자리만 그대로 둔다 — 살아나는 것은
S3 의 카드다. **CP6 은 장면 밖이라 CP4 뒤 아무 때나 닫는다.**

---

# 7. 접점 — 부딪히지 않게

여러 담당이 만지는 파일이 있다. 로스터 5절이 정본이다.

```text
   internal/panel      panel 과 transcript 둘 다 nacl1119 다 — 한 손 안이라 접점이 아니다
   internal/enode      drain(shin-son · 광고와 정책) vs transcript(링 tee) — 다른 파일
   internal/api/api.go 등록 줄만 만진다.  구조 불변식은 requirements/constraints.md
   internal/api/ui     card-news(nacl1119) vs ui(runixs) — 이번 두 유닛은 안 건드린다
   cmd/mediator        queue(shin-son) · ui(runixs)와 접점 — 진행자가 직렬 병합
```

**이번 두 유닛이 안 건드리는 것** — `internal/store` · `internal/api/ui` ·
`cmd/mediator` · `design/` · 루트 상태 파일 둘.

---

# 8. 승인 지점 — 사람을 기다리는 자리

AI-DLC 는 단계마다 명시적 승인을 요구한다. **승인 없이 다음 단계로 넘어가지
않는다.** 각 지점에서 사용자 입력 원문을 `aidlc-docs/nacl1119/audit.md` 에
그대로 적는다 — 요약하지 않는다.

```text
   1   ADR-068 질문 파일의 답                    진행자 결정.  panel FD 를 막는다
   2   panel Functional Design 산출물
   3   panel NFR Requirements 산출물
   4   panel NFR Design 산출물
   5   Infrastructure Design 을 건너뛸지
   6   panel Code Generation 계획 (Part 1)
   7   panel Code Generation 산출물 (Part 2)
   8   CP4 판정과 PR 을 열지
   9   transcript 의 같은 자리들
   10  CP6 판정과 PR 을 열지
```

---

# 9. 첫 손

**브랜치는 이미 서 있다.** `unit/panel` 을 `origin/main` 의 끝(`50af6cf`)에서
따 두었고, 실수로 main 에 푸시되지 않게 upstream 을 끊어 두었다. 다시 따거나
옮기지 않는다 — `git branch --show-current` 로 확인만 한다.

```text
   지금        unit/panel  (50af6cf 위)
   처음 푸시    git push -u origin unit/panel
   transcript  그때 가서 origin/main 에서 unit/transcript 를 새로 딴다
```

3절의 의존을 눈으로 확인하고(`internal/enode/policy.go` ·
`internal/runctl/client.go`), 4절의 ADR-068 질문 파일을 내고, panel Functional
Design 계획을 `aidlc-docs/nacl1119/construction/plans/panel-functional-design-plan.md`
에 쓴다.

`aidlc-docs/nacl1119/aidlc-state.md` 의 panel 항목 체크박스를 단계마다 그
자리에서 갱신한다 — 일을 끝낸 상호작용과 같은 상호작용 안에서다.
