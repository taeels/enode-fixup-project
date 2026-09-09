# enode 영상 제작 기록

2026-09-09 · 담당 runixs · AI-DLC v1.0.1 Construction 지원 문서.

Galaxy S2 개발환경을 찾는 요청에서 시작해, 서대현님의 메신저 승인 이후 해당
장치의 enode와 Claude가 일을 수행하는 60초 영상을 제작했다. 첫 15초와 폰
화면 수정본은 사용자가 채택했다. 후반 30초는 실제 웹 UI에 가상 노드와 작업
상태를 입력해 구성했다. 사용자는 영상 내용을 문서화하고 커밋하도록 요청했다.

## 문서와 소유 범위

| 문서 | 내용 |
|---|---|
| [전체 캐릭터 시트](ddthon-all-character-sheets.md) | DDTHON 9종, 원본 대응, 보완 디자인과 고정 기준 |
| [첫 15초 대본](opening-15s-rewrite-v3.md) | S2 수소문, 긴급 업데이트 대사, Kling 입력과 실제 검수 |
| [회사 메신저 자산](company-messenger-assets-v1.md) | ddthon-6 화면, ddthon-7 로고, 승인 화면 합성 |
| [후속 45초 계획](continuation-45s-plan.md) | 요청·승인 15초와 실행 설명 30초의 완료 체크 |
| [제작 요약](code/implementation-summary.md) | 장면별 책임, 캐릭터 고정, UI 데이터, 합성·내레이션·재개 절차 |
| [출력·검증 기록](code/build-and-test.md) | 규격·해시, 검사 결과, 적용 범위와 남은 시각·청취 한계 |
| [초기 캐릭터 계획](character-reference-plan.md) | 과거 보류 작업. 전체 9종 계획으로 대체한 경위 |
| [Remotion UI 샘플 계획](remotion-ui-sample-plan.md) | 후반 구성 전의 별도 실험. 최종 영상과 구분한 보류 기록 |

영상은 roster의 새 제품 유닛이 아니다. 담당 문서 루트와 계획·구현 요약·검증·
감사 기록의 구분을 따르며, 기존 ui/demo-back의 공식 Code Generation 단계와
공동 장면 게이트 상태를 유지한다. Inception과 이미 승인된 설계를 다시 시작하지
않는다. 영상 속 SUCCEEDED는 가상 데이터이며 실제 S2 성공이나 CP10 통과의
증거가 아니다.

## Git에 남기는 것과 로컬에 보존하는 것

Git에는 이 디렉터리의 제작 문서와 담당 state·audit의 영상 기록을 남긴다.
MP4·PNG·ZIP, 원본 ddthon 이미지, 제작 코드·의존성·임시 캡처·생성 응답 및
`dist/`는 사용자의 최종 지시에 따라 로컬에 그대로 둔다. 삭제하거나 이동하지
않는다. 기존 `.gitignore`의 `/dist/` 정책을 유지한다.

로컬 영상 루트는 `output/enode-video/`다. 아래 파일은 Git에 포함되지 않으므로
새 체크아웃에서는 별도로 전달받아야 한다. 문서의 로컬 경로는 저장 위치를
설명하는 텍스트이며 저장소에 포함된 다운로드 링크가 아니다.

| 산출물 | 로컬 경로 |
|---|---|
| 전체 60초 | `output/enode-video/continuation-45s-v1/remotion/out/enode-full-60s-720p.mp4` |
| 요청·승인 15초 | `output/enode-video/continuation-45s-v1/remotion/out/enode-approval-15s-720p.mp4` |
| 승인 이후 30초 | `output/enode-video/continuation-45s-v1/remotion/out/enode-after-approval-30s-720p.mp4` |
| DDTHON 시트 9종 | `output/enode-video/character-sheets/ddthon-all-v1/` |
| 메신저 자산 | `output/enode-video/messenger-assets-v1/` |
| 전체 로컬 인계 자료 | `output/enode-video/HANDOFF.md` |

## 최종 결정

- 파란 기사 02는 요청자, 금색 기사 06은 동료다. 서대현님은 휴가지의 청록색
  손과 현대형 휴대폰으로 등장한다.
- 파란 몸체는 균일한 파랑으로 보정하고 기존 시트와 두 Kling Element를 고정했다.
- Claude는 사용자가 지정한 Claude Code 주황 픽셀 캐릭터로 표현했다.
- 메신저의 녹색 원본과 임의 글자는 제거했다. 손가락과 승인 버튼의 정확한 위치가
  다른 점은 사용자가 사소한 것으로 수용했으므로 현재 수정본을 유지한다.
- 승인 후 요청 전달, 장치의 enode 수신, Claude 실행, 지정 sandbox 안 작업,
  결과 반환 순서를 지킨다. 화면의 승인 완료와 작업 성공은 별도 상태다.
- 이번 60초 판의 Kling 비용은 첫 15초 135크레딧, 후속 8초·7초 합계
  135크레딧이다. 과거 초안 비용과 이미지 생성 비용은 이 합계에 넣지 않는다.

사용자 원문과 결정 시각은 [담당 감사 로그](../../audit.md)에 보존한다.
