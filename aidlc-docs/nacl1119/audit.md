# AI-DLC Audit Log — nacl1119

담당 handle `nacl1119` 의 Construction 감사 로그. 완료본 회차
(`v1-run-dhseo-cardnews`)의 초판 감사 기록은 `aidlc-docs/
v1-run-dhseo-cardnews/audit.md` 에 있다 — 여기는 그 뒤 업데이트만
잇는다. 타임스탬프는 `date -u +%Y-%m-%dT%H:%M:%SZ` (ISO 8601 UTC).

---

## card-news 업데이트 — main 재정렬
**Timestamp**: 2026-09-08T11:10:00Z (근사 — 사후 일괄 기록)
**User Input**: "main 기준으로 rebase 해서 다시 진행." 이어서 방향
질문에 "이번 세션에서 만든 aidlc-docs 산출물은 main과 같은 정책으로
적용." "코드 디자인 작업은 그대로 살린다." "열려있는 PR#2는 폐기."
**AI Response**: `v1-run-dhseo-cardnews` 브랜치가 main 보다 훨씬
앞서 있었고(카드 일러스트·영상·문구 수정·문서 정책), main 은 그
브랜치의 세션 시작 전 시점(`a2f5872`)만 병합한 뒤 팀장이 독자적으로
`aidlc-docs` layering(`941201c`)과 담당 배정(`c010841`)을 얹어
`aidlc-docs/<handle>/` 정책을 이미 확정해 뒀다는 것을 확인했다.
단순 `git rebase` 는 양쪽이 같은 문서를 독자적으로 재배치해 대량
충돌이 예상되므로, 대신 origin/main 에서 새 브랜치
`nacl1119-card-news-update` 를 따고 코드·디자인 diff(a2f5872..
46a91cb, `internal/` · `design/` 만)만 그대로 옮겨 실었다. AI-DLC
문서 루트 정책 커밋 2개(`488de0f` · `024ee2e`, main 이 이미 더 나은
버전을 가진 중복)는 버렸다. 이번 업데이트의 Functional Design ·
NFR · Build and Test 기록은 `aidlc-docs/nacl1119/construction/
card-news/update-2026-09-08.md` 하나로 정리해 새로 썼다 — 완료본
(`aidlc-docs/v1-run-dhseo-cardnews/`)은 얼린 채로 둔다.
**Context**: CONSTRUCTION 계속(담당 handle 기준). PR #2 는 폐기
대상 — main 이 이미 그 시점을 병합했고 브랜치가 새 기준(main)으로
옮겨졌으므로 그 PR 의 비교 기준 자체가 낡았다.

---

## Construction 인계 — panel · transcript
**Timestamp**: 2026-09-09T02:38:00Z
**User Input**: "main 에서 shin-son 님 PR merge 되었으면 panel 유닛(W3 · CP4) -> transcript(W4 · CP6) 순서로 작업 진행해." / "작업 계획부터 제출." / "이 계획으로 aws bedrock claude에게 넘기도록 handoff 문서 작성해두고, 전달할 프롬프트 작성." / "지금 경로에서 bedrock을 띄울거야." / "main 에서 새로 브랜치 따서 해야해." / "내가 git switch main을 직접 했어. 너 작업과 꼬인게 있으면 너가 맞추면 돼." / "bedrock을 이미 띄웠어."
**AI Response**: shin-son 의 PR 둘(#5 queue · #7 drain)이 main 에 병합된 것을 확인했다. panel 의 의존 obs(#4) · drain(#7) 이 둘 다 닫혀 W3 착수 조건이 섰다. panel -> transcript 순서의 작업 계획을 내고, 그것을 AWS Bedrock 위의 Claude 가 이어받을 수 있게 핸드오프 문서와 전달 프롬프트를 construction/plans/ 에 썼다. 브랜치 unit/panel 을 origin/main(50af6cf)에서 따고 upstream 을 끊었다 — 안 끊으면 git push 가 main 으로 간다. 착수를 막는 것이 하나 남아 있다: 제어판이 데몬 Capabilities{Caps,At} 를 읽는 계약(ADR-068)이 requirements/decisions.md 에 아직 없다. 이것은 진행자 결정이므로 panel Functional Design 첫머리에서 질문 파일로 낸다.
**Context**: CONSTRUCTION 인계. 이 커밋 이후의 panel · transcript 단계는 Bedrock 세션이 이어서 기록한다.

---

## panel Functional Design 착수 — ADR-068 질문과 FD 계획
**Timestamp**: 2026-09-09T02:47:43Z
**User Input**: "aidlc-docs/nacl1119/construction/plans/panel-transcript-handoff-prompt.md 를 읽고 그 안의 프롬프트대로 시작해."
**AI Response**: 핸드오프 프롬프트대로 착수했다(이 세션이 곧 인계받은 Bedrock 세션이다). 브랜치가 unit/panel(origin/main 50af6cf 위)임을 확인했다. 규약·핸드오프 문서·정본(unit-of-work §5 · construction-roster · decisions §2 「제어판 서버 위치」·「LAN 노출의 토큰」·§6.5 · scene-gates CP4)과 의존 표면(internal/enode/policy.go · internal/runctl/client.go)을 읽고, 착수 블로킹인 ADR-068 을 실측으로 좁혔다: Capabilities{Caps, At} 는 데몬 메모리에만 있고(detector.go:78 · main.go:167) 데몬은 나가는 http.Client 만 있어(main.go:155) 다른 프로세스인 제어판이 At 을 읽을 통로가 없다. Caps 는 GET /v1/nodes 로 읽히나 At 은 광고에 안 실린다(ADR-068). 이 미결을 갈래 넷(로컬 상태 파일 · 로컬 데몬 엔드포인트 · At 광고로 ADR-068 뒤집기 · At 미루고 seen_at 을 정직하게 표시)으로 정리해 질문 파일을 냈다: construction/panel/functional-design/adr-068-questions.md. panel Functional Design 계획을 construction/plans/panel-functional-design-plan.md 에 썼다.
**Context**: CONSTRUCTION · panel · Functional Design Part 1(계획). 승인 지점 1(ADR-068)과 2(FD 계획) 앞.

---

## ADR-068 진행자 결정 — A (로컬 상태 파일)
**Timestamp**: 2026-09-09T02:57:11Z
**User Input**: "ADR-068 질문 너의 추천은 뭐야" -> (A 추천) -> adr-068-questions.md 의 [Answer]: A -> "대답했어" -> (decisions 행 문안 확인 요청) -> "진행해"
**AI Response**: ADR-068 을 A 로 받았다 — 데몬이 정책 파일 옆 <stem>.status.yaml 에 Caps·At 을 쓰고 제어판이 읽는다. decisions.md §2 를 읽다가 정본과 부딪히는 자리를 찾아 진행자에게 보였다: §2 「S3 현재 작업의 출처」(decisions.md:82)가 상태 파일 방식을 「새 기계」로 한 번 물렸다. 그 행은 lease 에 한정되고 At 은 Mediator 에 없어 같은 논리가 안 닿음을 설명했다. 진행자가 진행해 확인. requirements/decisions.md §2 에 「탐지 능력 읽기 (ADR-068) · At 의 출처」 행을 더해 캐논에 닫았다.
**Context**: CONSTRUCTION · panel · 승인 지점 1(ADR-068) 닫힘.

---

## 작업 트리 흔들림 — 산출물 유실 후 unit/panel 로 복구
**Timestamp**: 2026-09-09T02:57:11Z
**User Input**: (AskUserQuestion) "panel 작업을 누가 진행할까요?" -> "이 세션이 다시 진행" / "현재 변경을 stash -> unit/panel 로 checkout -> pop -> 거기서 커밋(유실 방지) 진행해. 다른 세션 멈췄어." / "이 세션이 계속 소유해." / "너가 bedrock이야."
**AI Response**: 이번 세션 초반에 작업 트리가 두 번 흔들려(unit/panel 리셋 · main 으로 checkout) 첫 산출물이 유실됐다. 경쟁 세션이 있는 줄 알았으나 진행자가 "너가 bedrock이야" 로 정정 — 이 세션이 곧 인계받은 Bedrock 세션이고 단독 소유자다. 유실된 산출물(ADR-068 질문 파일 · FD 계획 · decisions §2 행 · state/audit 기록)을 전부 다시 만들었고, main 에 얹혀 있던 그 변경을 stash -> unit/panel checkout -> pop 으로 옮겼다(aidlc-state.md 는 3-way 자동 병합, 충돌 없음). 이 단계를 unit/panel 에 커밋해 유실을 막는다. main 에는 커밋하지 않는다.
**Context**: CONSTRUCTION · panel · 세션 소유권 확정 · unit/panel 복구 · 커밋.

---

## panel Functional Design 산출물 셋
**Timestamp**: 2026-09-09T03:11:36Z
**User Input**: (앞선 "진행해" 로 FD 산출물 작성 승인) — ADR-068=A 결정 뒤 FD 본문 작성.
**AI Response**: panel Functional Design 산출물 셋을 construction/panel/functional-design/ 에 냈다: domain-entities.md(Config·화면 값 다섯 묶음·상태 파일 <stem>.status.yaml·정책 파일 panel_token additive·proc 추출 경계) · business-logic-model.md(serve exec 위임·탐지 능력 읽기·stop cancel-먼저·drain 토글·흐름 ASCII) · business-rules.md(바인딩/LAN 토큰·값 없을 때 분기·임포트 경계·심볼 상한 CP4·상태 파일 쓰기·완료 조건 버튼 표·security-baseline). 정본 충돌 하나를 못 박았다: enode-features 3.1.2(:198)는 proc 함수 셋을 internal/panel 로 내리라 했으나, 그러면 cmd/enodectl 의 stop/start 가 그 패키지를 거쳐 net/http 에 닿아 enodectl.exe 심볼 상한(CP4)이 깨진다. 유닛 정본 unit-of-work §5(internal/proc · net/http 없음)가 상한을 지키는 유일한 갈래라 그것으로 간다. FD 계획 체크박스를 갱신했고(마지막 승인-대기 항목만 남김) 코드는 아직 안 썼다. 완료 메시지를 2지 선택으로 낸다.
**Context**: CONSTRUCTION · panel · Functional Design 산출물 완료 · 승인 지점 2(FD) 앞.

---

## panel Functional Design 승인 · 남은 작업 인계 요청
**Timestamp**: 2026-09-09T03:12:54Z
**User Input**: "진행해" (FD 승인 · NFR Requirements 로) 이어서 "남은 작업 동료분에게 전달하게 요약해줘."
**AI Response**: 진행자가 panel Functional Design 을 승인했다(승인 지점 2 닫힘). NFR Requirements 로 넘어가려던 참에, 진행자가 남은 작업을 동료에게 넘길 요약을 요청했다. FD 승인 표시를 계획에 반영해 커밋하고(브랜치를 깨끗이 넘김), 남은 작업 요약을 낸다. NFR 이후 단계는 인계받는 사람이 잇는다.
**Context**: CONSTRUCTION · panel · FD 승인 · 인계 준비.

---

## panel NFR Requirements
**Timestamp**: 2026-09-09T03:17:22Z
**User Input**: "환경 주의는 안해도 됨. 작업 진행해. 요약 커밋해둬."
**AI Response**: 인계 요약을 construction/plans/panel-remaining-handoff.md 로 저장해 커밋했다(1979743 · 환경 주의 절 제외). 이어 panel NFR Requirements 를 construction/panel/nfr-requirements/nfr-requirements.md 에 냈다: 성능·바이너리(심볼 상한 CP4 차단 게이트) · 폴링(팩 권장 5초) · 보안(바인딩 127.0.0.1·LAN panel_token 거부 경로·파일 권한 0600·기본 무인증 근거) · 신뢰성 저하 분기 · 시험성 커버리지 80%(internal/panel·internal/proc · DB 없이) · 이식성 크로스빌드·빌드 태그 짝 · 기술 스택 무변경(net/http 표준·새 의존 0·packaging 무변경) · 표기/언어 차단. 값을 새로 정한 자리 없음(폴링은 팩 권장을 딛음). 완료 메시지를 2지 선택으로 낸다.
**Context**: CONSTRUCTION · panel · NFR Requirements 완료 · 승인 지점 3 앞.

---
