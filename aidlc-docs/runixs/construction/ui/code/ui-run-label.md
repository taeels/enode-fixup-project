# Run 식별자 표시 축약 결과

2026-09-09 `Runixs/UI-Update-2`에서 [후속 계획](../../plans/ui-run-label-plan.md)을
수행했다. 기존 UI Code Generation 보완이며 공동 장면 승인·배포와 구분한다.

## 변경

`꾸준한 단풍 · demo-dada8cd6 · FAILED`처럼 긴 16진수 식별자의 접두어와
앞 8자리를 표시한다. `demo-`, `gallery-`, `gallery-post-`와 접두어 없는
긴 해시에 공통 적용한다. 짧은 ID와 사람이 지정한 이름은 보존한다.

작업 그래프 제목, 작업 목록, 노드의 현재 작업, 질문 연결 버튼, 데모 접수
안내와 모달 상태를 수정했다. 전체 식별자는 title 속성으로 확인할 수 있다.
노드 기술 상세의 원문, CLI 명령, 선택 키, API 조회 URL, 제출 응답 상태의
run_id와 재시도는 원본을 사용한다. 축약 이름이 같은 두 Run도 독립 선택된다.

## 검증

| 검사 | 결과 |
|---|---|
| `node --test internal/api/ui/tests/*.test.mjs` | 59 통과, 실패·스킵 0 |
| 별도 표시 확인 | 접두어·짧은 이름·사용자 문자열·빈값 12개 통과 |
| Playwright Chromium | demo/fleet × 1440px/390px 4환경 통과, pageerror 0 |
| `go test -count=1 -cover ./internal/api/ui` | 통과, 98.4% |
| `go vet ./internal/api/ui` | 통과 |
| `go run ./scripts/glyphscan.go` | 102파일, 위반 없음 |
| `go build -o /tmp/enode-ui-run-label-mediator ./cmd/mediator` | 통과 |
| demo.js 모듈 구문·`git diff --check` | 통과 |

기본 PATH에 Go가 없어 설치된 `/tmp/enode-impl-tools/go/bin/go`로 실행했다.
브라우저의 실제 UI 모듈과 obs-contract fixture를 사용해 2D/3D 제목·목록,
전체 ID tooltip, 같은 앞 8자리의 서로 다른 Run 선택·조회 URL, 노드 상세,
질문 버튼·CLI 원문, 제출 승인 ID·모달 상태를 확인했다. 390px에서 가로 넘침이
없고 축약 제목이 한 줄로 보인다. 브라우저와 요청 모형은 검사 후 종료했다.

재현 스크립트와 이미지·결과는 로컬 `/tmp/enode-run-label-check.DjuuTe/`에 있다.
새 라이브러리나 영구 테스트 파일은 추가하지 않았다. DB·실제 장비 검사는
실행하지 않았으며 전체 CP0 또는 공동 CP6/CP10 통과로 기록하지 않는다.
이 체크아웃의 코드·빌드 검증까지 완료했고 운영 서버에는 배포하지 않았다.

## 보안 확장

SECURITY-04·05·08·11·12·13·15 준수: 텍스트·title 삽입을 사용하고 원본 ID의
검증·인증·API 경계를 보존한다. 새 요청·토큰·HTML 삽입이 없다.
SECURITY-01·03·07은 decisions §3 예외를 유지한다.
SECURITY-02·06·09·10·14는 해당 인프라·권한·배포·의존·운영 변경이 없어 N/A다.
비활성 확장 둘과 승인된 NFR/인프라 SKIP은 유지하며 차단 소견은 없다.
