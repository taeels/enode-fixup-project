# Run 식별자 표시 축약 — 2026-09-09

기존 승인된 UI Code Generation의 가독성 후속 수정이다. 사용자의 “간단하게
표시될 수 있게” 지시를 기존 계획 1·2·5단계 보완으로 수행한다. 담당은 roster의
ui 소유자 runixs이며 작업 브랜치는 `Runixs/UI-Update-2`다.

## 수용 조건

- 긴 16진수 식별자의 접두어와 앞 8자리만 표시한다. 예: `demo-dada8cd6`.
- 작업 그래프·목록·노드 상세·접수 안내에 적용하고 전체 ID는 title로 제공한다.
- 서버 ID, 선택·조회·재시도, CLI 명령의 ID는 원문을 유지한다.
- 사람이 지정한 이름과 짧은 ID는 그대로 표시한다.

## 실행

- [x] 1. 담당 상태·승인·표시 지점·파일 경계·기존 게이트 증거를 확인한다.
- [x] 2. `internal/api/ui/static/shared/fleet/{format,view}.mjs`와
  `internal/api/ui/static/demo/{demo.js,submission.mjs}`의 표시만 수정한다.
- [x] 3. 기존 Node 검사, 데스크톱·모바일 브라우저의 축약 표시·원본 선택,
  Go UI 검사·vet·glyphscan·Mediator embed 빌드를 확인한다.
- [x] 4. 결과와 담당 state/audit를 갱신한다.

[검증 결과](../ui/code/ui-run-label.md): Node 59개·표시 확인 12개,
브라우저 4환경, Go UI 98.4%, vet·glyphscan·Mediator 빌드 통과.

obs·queue·demo-back 인수 증거는 기존 담당 상태와 UI 가독성 기록을 계승한다.
새 API·DB·의존·프로토콜 변경이 없어 FD 및 승인된 NFR/인프라 SKIP을 유지한다.
DB 전체·실제 장비 검사는 이번 정적 수정 범위 밖이며 공동 CP6/CP10 보류를
통과로 변경하지 않는다.

## 보안 확장

SECURITY-04·05·08·11·12·13·15: 기존 헤더·인증·검증을 보존하고 DOM textContent와
title 속성으로만 표시한다. 축약 ID를 요청 키로 사용하지 않는다.
SECURITY-01·03·07은 decisions §3의 예외를 상속한다.
SECURITY-02·06·09·10·14는 해당 인프라·권한·배포·의존·운영 변경이 없어 N/A다.
resiliency-baseline·property-based-testing은 Disabled로 실행하지 않는다.
