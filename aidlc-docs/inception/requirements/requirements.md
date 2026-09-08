# Requirements — enode 기능 추가 + 대회 데모 전환

AI-DLC Requirements Analysis 산출물이다. **정본은 `requirements/` 팩**
(`enode-features.md` · `decisions.md` · `scene-gates.md` · `canon.md` ·
`constraints.md`)이고 이 문서는 그것을 요약하고 의도 분석·추적을 얹는다. 값은
다시 적지 않고 팩을 가리킨다.

## 의도 분석
- **사용자 요청**: (1) enode 에 기능 일곱을 더한다 — 관측·제어·접수 (`requirements/` 팩). (2) 2026-09-08 현장 워크숍이 **대회 데모로 전환**하고 시연 다섯을 더했다.
- **요청 유형**: Enhancement + New Feature (브라운필드).
- **범위**: Multiple Components — `cmd/mediator` · `internal/api`(+`ui`) · `internal/store` · `internal/match` · `internal/enode` · `cmd/enodectl` · 새 패키지(`internal/panel` · `internal/api/ui` · `internal/mcp`) + 데모 표면.
- **복잡도**: Complex — 상태 어휘 추가(`QUEUED`) · 새 표면 여럿 · 차단 CI 게이트 다섯 · 정본 불변식 · 대회 데모의 공개 쓰기 표면.

## 기능 요구사항 (요약 — 정본은 `enode-features.md`)

### 일곱 — 데모가 딛는 기계
- 3.1.1 중앙 현황판 · 3.1.2 호스트 제어판 · 3.1.3 하네스 트랜스크립트
- 3.2.1 대기열(`QUEUED`) · 3.2.2 소유자 drain · 3.2.3 되묻기
- 3.3.1 Mediator MCP (`runctl mcp`)

### 시연 다섯 — 대회 데모 (2026-09-08 전환)
- 3.4.1 온보딩(첫 방문 카드 + 대시보드 가이드투어)
- 3.4.2 게스트 공개 데모 대시보드(Guest 로그인 · 랜덤 이름 · 3D 함대뷰 S6)
- 3.4.3 고정 시나리오 제출(서버측 allow-list · 시나리오 둘 · 작업그래프 S7 전환)
- 3.4.4 제출자 이름(`run.submitter` · 목록 노출 · 카드 표시 · 스텝 주입)
- 3.4.5 웹캠 플로팅(out-of-band 임베드 · 좌하단 resizable)

## 비기능 요구사항 (요약)
- **보안**: `security-baseline`(차단성) — 새 표면에 적용. 데모 공개 쓰기는 SECURITY-08 **수락 위험**(가두는 것 셋 — 일회용 Mediator · 서버측 allow-list · 브라우저에 실 토큰 없음). 기존 코드 사실(SECURITY-01·03·07)은 기록만.
- **CI 차단 게이트 다섯**: `crypto/tls` 심볼 상한 10 · `net/http` 상한 50(`enodectl.exe`) · 패키지 커버리지 하한 80% · 스킵 0 · U+2605 0.
- **정본 불변식**: `INVARIANTS` 우선 — `ALLOCATING -> QUEUED` 전이 · `202` · drain 취소 경로 · `GET /v1/nodes` 모양(draining 포함).
- **데모**: 일회용 Mediator · 웹캠 out-of-band · 3D 는 pen.dev(shonsin) S6·S7.

## 수용 기준
- `scene-gates.md`: 기계 게이트 **CP0~CP7**(전환 전 기록 CP0~CP4 포함) + 대회 데모 장면 **CP8~CP11**.
- **완결성 정본은 `CP10`(데모 완주)** — 옛 `CP4` 의 자리.

## 확장 재확인 (Step 5.1)
| Extension | Enabled | Decided At |
|---|---|---|
| security-baseline | Yes | decisions.md §1·§3 (2026-09-04) |
| resiliency-baseline | No | decisions.md §1 |
| property-based-testing | No | decisions.md §1 |

## 추적
- 질문/답: `requirement-verification-questions.md` (Q1~Q4 = A, veto 없음, Q1 시나리오 둘 명시).
- 결정 정본: `decisions.md` §1·§2·§5·§6·§7·**§8(대회 데모)**.
- RE 근거: `aidlc-docs/inception/reverse-engineering/` (라우트 15 · `QUEUED`/`GET /v1/nodes`/`serve` 부재 확인).
