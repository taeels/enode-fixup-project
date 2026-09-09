# 넘길 프롬프트 — AWS Bedrock 위의 Claude 에게

핸드오프 문서는 `aidlc-docs/nacl1119/construction/plans/panel-transcript-handoff.md`.
아래를 그대로 붙여 넣는다.

**저장소 작업 디렉터리에서 그대로 띄우는 것을 전제로 한다.** 브랜치는 이미
준비돼 있다 — `unit/panel` 을 `origin/main`(50af6cf)에서 따 두었고, 실수로
main 에 푸시되지 않게 upstream 을 끊어 두었다. 처음 푸시할 때
`git push -u origin unit/panel` 로 자기 자리를 잡는다.

---

```text
너는 이 저장소(github.com/taeels/enode-fixup-project)의 Construction 담당
nacl1119(문태호)의 일을 이어받는다. 유닛 둘을 순서대로 끝내는 것이 일이다 —
panel(웨이브 W3 · 게이트 CP4) 을 먼저, 그것이 main 에 병합된 뒤 transcript
(웨이브 W4 · 게이트 CP6) 를.

## 먼저 읽는다

aidlc-docs/nacl1119/construction/plans/panel-transcript-handoff.md 가 이 일의
핸드오프 문서다. 착수에 필요한 것이 거기 다 있다 — 규약, 문서 루트, 지금
의존 상태, 막고 있는 미정 하나, 유닛 둘의 책임과 게이트 판정 기준, 접점,
승인 지점 목록. 그것을 먼저 읽고, 거기서 가리키는 정본들을 읽는다.

정본과 핸드오프 문서가 어긋나면 정본이 이긴다. 핸드오프 문서는 길잡이지
결정이 아니다.

그다음 CLAUDE.md, CONVENTIONS.md, .aidlc/aidlc-rules/ 아래의 AI-DLC v1.0.1
워크플로, aidlc-docs/construction-roster.md 를 읽는다. 이 저장소는 AI-DLC 로
도는 저장소라 단계와 승인 지점이 정해져 있다.

## 반드시 지킬 것

- 답변과 문서와 커밋 메시지는 한국어로 쓴다. 에러 문자열, 로그, CLI 출력,
  테스트 이름과 실패 메시지, 테스트 픽스처 문자열은 영어로 쓴다. 기준은
  「코드냐 아니냐」가 아니라 「밖으로 나가느냐」다 (CONVENTIONS 2절).
- 강조는 마크다운 굵게로만 한다. 코드에는 장식 문자를 넣지 않는다 — 주석,
  문자열, 로그, 테스트 메시지, 설정 어디에도. 커밋 메시지에도 안 넣는다.
  scripts/glyphscan.go 가 CI 에서 이것을 센다 (CONVENTIONS 1절).
- AI-DLC 단계마다 사람의 승인을 기다린다. 승인 없이 다음 단계로 넘어가지
  않는다. 완료 메시지는 2지 선택(변경 요청 / 다음 단계로)으로 낸다 —
  3지 이상 메뉴를 만들지 않는다.
- 산출물은 aidlc-docs/nacl1119/ 아래에만 쓴다. 루트의
  aidlc-docs/aidlc-state.md 와 aidlc-docs/audit.md 와 design/ 은 진행자
  몫이라 건드리지 않는다.
- aidlc-docs/nacl1119/audit.md 는 이어 붙인다. 통째로 덮어쓰면 기록이 두
  벌이 되므로 절대 덮어쓰지 않는다. 사용자 입력은 원문 그대로 적고
  요약하지 않는다.
- 브랜치는 unit/panel 과 unit/transcript 를 main 에서 딴다. 커밋은 단계
  승인마다, 병합은 그 유닛의 장면 게이트가 초록이 된 뒤에 PR 로만 한다.

## 착수를 막고 있는 것 하나

제어판이 도는 데몬의 Capabilities{Caps, At} 를 읽는 계약(ADR-068)이 아직
안 정해졌다. requirements/decisions.md 에 그 행이 없다. 이것은 진행자가
정할 일이지 네가 정할 일이 아니다.

panel Functional Design 첫머리에서 갈래를 정리해 질문 파일로 내고 사람의
답을 기다린다. 질문 형식은 AI-DLC 정본의 common/question-format-guide.md 를
따른다. 답이 오기 전까지는 겉면에 자리만 두고 나머지를 설계한다.

## 첫 손

브랜치는 이미 준비돼 있다. 지금 작업 디렉터리가 unit/panel 위에 있고 그것은
origin/main 의 끝(50af6cf)에서 딴 것이다. git branch --show-current 로 확인만
하고, 브랜치를 다시 따거나 옮기지 않는다. transcript 를 시작할 때가 되면
그때 origin/main 에서 unit/transcript 를 새로 딴다.

핸드오프 문서 3절이 가리키는 의존을 눈으로 확인한다 — internal/enode/policy.go
의 정책 파일 형식, internal/runctl/client.go 의 Nodes·Runs·Cancel·Record.
그다음 위의 질문 파일을 내고, panel Functional Design 계획을
aidlc-docs/nacl1119/construction/plans/panel-functional-design-plan.md 에
쓴다.

계획을 먼저 제출하고 승인을 기다린다. 코드부터 쓰지 않는다.
```
