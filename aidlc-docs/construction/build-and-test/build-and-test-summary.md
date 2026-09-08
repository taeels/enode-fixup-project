# Build and Test 요약 — demo-pen (2026-09-08)

## 빌드

- **도구**: pen.dev + pencil MCP `execute` / `Export`
- **상태**: 성공
- **산출물**:
  - `design/enode-demo.pen` — 아트보드 여덟 (D1 · D2 · D3 · D3b · D3c · D3d · D4 · D5) + 노트 셋 + 컴포넌트 다섯
  - `design/exports/D*.png` 여덟 장, 3840 x 2160 (2x)
- **불변 확인**: `design/enode-ux.pen` 디스크 md5 = HEAD md5 (`5f04…c62d`).
  기존 `exports/S*.png` · `C*.png` 열한 장 변경 없음 (`git status` 에 안 잡힘)

## 검사 결과

### 단위 검사 (아트보드별)

| 검사 | 결과 |
|---|---|
| placeholder 남은 노드 | 0 |
| 레이아웃 problems (모형 path · 간선 · 덮개 제외) | 8 장 모두 0 — D4 `Task Sub` 1 건이 나와 `fixed-width` 로 고친 뒤 0 |
| 사양의 버튼 전수 존재 | 8 장 모두 missing none |
| 장식 문자 (CONVENTIONS 1.2) | 0 |
| 저장되지 않는 상태 어휘 (SEALED 등) | 0 |
| D3d 마지막 버튼 | 시작하기 |

### 통합 검사

| 시나리오 | 결과 |
|---|---|
| 1. D2 복사본 여섯의 우 열 텍스트 = D2 | 6 / 6 true |
| 2. 웹캠 창 자리 (24, 576, 360 x 360) | 7 / 7 같음 |
| 3. 전환 표의 출발 버튼이 그림에 있다 | 통과 (단위 검사의 버튼 전수와 같은 근거) |
| 4. D5 에서 run 목록이 안 가린다 (수용 기준) | 통과 — Run 행 여섯 전부 clip 없음, export 로 눈 확인 |

### 성능 검사
해당 없음.

### 추가 검사
- **계약 검사**: N/A (API 없음)
- **보안 검사**: N/A — security-baseline 끔 (Q5)
- **E2E**: 시안이라 N/A. 흐름 F1~F5 는 전환 표로 대신 봤다

### 확장 준수
- security-baseline: 끔
- resiliency-baseline: 끔
- property-based-testing: Partial — 코드 없음. PBT-02 · 03 · 07 · 08 · 09 전부 N/A

## 사람이 본 것

여덟 장의 1x/2x export 를 눈으로 봤다 — 텍스트 잘림 없음 · 웹캠 창이 3D 위에
뜸 · D3 넷의 스포트라이트가 대상만 밝힘 · D4 모달 뒤로 목록 윤곽이 비침 ·
D5 의 flash.led 가 보드 칩 모형에 배정. 사용자가 pen.dev 에서 직접 고친 문구
(D3 넷 · D4)도 검수했다.

## 남은 것

- pen.dev 는 자동 저장하지 않는다. D4 `Task Sub` 줄바꿈 수정이 편집기에만
  있으므로 **Cmd+S 뒤 커밋**해야 디스크 `.pen` 과 export 가 일치한다

## 전체 상태

- **빌드**: 성공
- **검사**: 전부 통과
- **다음**: Operations 는 자리만 있는 단계다. 이 회차는 여기서 끝난다 —
  `v2-run-shin_pen_drawing` 을 push 하고 PR 을 열 수 있다 (CONVENTIONS.md 3.5)
