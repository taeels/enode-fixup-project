# YouTube Live 웹캠 연결 보완 계획 — 2026-09-09

기존 승인된 UI Code Generation의 공개 데모 후속 변경이다. 사용자의 라이브
스트림 연결과 최종 링크 교체 지시를 기존 계획의 웹캠 설정 범위에서 수행했다.
구현과 운영 적용 뒤 요청받은 AI-DLC 추적 보완이므로, 승인이나 검사를 앞서
수행한 것처럼 소급하지 않고 실제 지시·변경·검증 순서를 기록한다.

## 수용 조건

- 최종 영상 ID `8y2ln1nCHbM`을 YouTube 공식 iframe URL로 제공한다.
- 웹캠 창이 열릴 때 무음 자동 재생을 요청하고 모바일에서는 인라인 재생한다.
- 기존 확대·이동·맞춤·크기 조절·닫기 동작과 공개 데모 URL을 유지한다.
- CSP는 `https://www.youtube.com`만 frame origin으로 추가한다.
- 토큰·비밀 값·DB·runctl·터널·노드 설정은 변경하지 않는다.

## 유닛 문맥

- 담당: roster의 `ui` 소유자 `runixs`
- 파일 경계: `internal/api/ui` 단독 소유, `internal/store` import 금지
- 의존: 기존 `WebcamWindow`, 정적 설정 검증, Mediator embed FS와 보안 헤더
- 기능·게이트: 공개 데모 웹캠, CP11 화면 조작과 CP10 방송 관측의 입력
- 데이터·API·DB: 변경 없음

## 실행

- [x] 1. 담당 상태·유닛 소유권·기존 웹캠 구현과 설정 검증 경계를 확인한다.
- [x] 2. 첫 요청의 영상 ID를 설정한 뒤 사용자 후속 지시에 따라 배포 전 최종
  영상 ID `8y2ln1nCHbM`으로 교체한다.
- [x] 3. `internal/api/ui/static/demo/settings.json`의 `embedUrl`만 수정한다.
- [x] 4. 공개 HTTPS URL, 금지된 자격 증명 query 부재와 CSP frame origin을 확인한다.
- [x] 5. macOS arm64 Mediator를 빌드하고 기존 설정·DB·터널을 유지해 정상 종료 후
  교체한다.
- [x] 6. 로컬·공개 경로의 settings JSON, CSP, 리슨 PID와 배포 버전을 확인한다.
- [ ] 7. 구현·배포 기록을 커밋하고 main 대상 PR을 제출한다.

사용자의 선행 테스트 생략 지시를 유지해 Node·Go 단위/통합/브라우저 테스트
스위트는 실행하지 않았다. 빌드와 운영 응답 확인만 성공했으며 이를 장면 게이트
통과로 기록하지 않는다.

## 보안 확장

| 규칙 | 판정 | 근거 |
|---|---|---|
| SECURITY-01 | N/A | 저장소와 전송 구성을 변경하지 않는다. |
| SECURITY-02 | N/A | 터널과 네트워크 중계 구성을 변경하지 않는다. |
| SECURITY-03 | N/A | 애플리케이션 로그 경로를 변경하지 않는다. |
| SECURITY-04 | 준수 | 기존 필수 헤더를 유지하고 frame origin 하나만 CSP에 추가한다. |
| SECURITY-05 | 준수 | Go와 브라우저 양쪽의 public HTTPS URL allowlist를 통과한다. |
| SECURITY-06 | N/A | 권한 정책 변경이 없다. |
| SECURITY-07 | N/A | 리슨 주소·방화벽·터널 설정 변경이 없다. |
| SECURITY-08 | 준수 | 기존에 public으로 승인된 데모 정적 표면만 사용한다. |
| SECURITY-09 | 준수 | 디렉터리 목록 금지와 일반 오류 처리를 보존한다. |
| SECURITY-10 | N/A | 새 의존성·이미지·빌드 도구가 없다. |
| SECURITY-11 | 준수 | 설정 검증과 제한된 CSP를 함께 적용한다. |
| SECURITY-12 | N/A | 인증·세션·자격 증명 변경이 없다. |
| SECURITY-13 | 준수 | 외부 스크립트를 추가하지 않고 검증한 iframe URL만 사용한다. |
| SECURITY-14 | N/A | 알림·로그 보존 구성 변경이 없다. |
| SECURITY-15 | 준수 | 기존 fetch 오류·잘못된 설정의 fail-closed 처리를 보존한다. |

차단 보안 소견은 없다. `resiliency-baseline`과 `property-based-testing`은 담당
상태에서 Disabled이므로 실행하지 않는다.
