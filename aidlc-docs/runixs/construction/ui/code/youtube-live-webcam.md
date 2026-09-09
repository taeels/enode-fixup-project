# YouTube Live 웹캠 연결 결과

2026-09-09 `unit/runixs-youtube-live-webcam`에서
[보완 계획](../../plans/ui-youtube-live-webcam-plan.md)을 수행했다. 기존 UI
Code Generation의 정적 배포 설정 변경이며 새 컴포넌트·API·DB는 만들지 않았다.

## 변경

`internal/api/ui/static/demo/settings.json`의 비활성 웹캠 설정을 다음 최종
YouTube iframe URL로 바꿨다.

```text
https://www.youtube.com/embed/8y2ln1nCHbM?autoplay=1&mute=1&playsinline=1
```

`WebcamWindow`의 기존 iframe 로더, 확대·이동·맞춤·크기 조절·닫기 UI를 그대로
사용한다. 서버는 설정에서 `https://www.youtube.com`을 추출해 데모 문서의
`frame-src`에만 추가한다. 다른 페이지는 기존 `default-src 'self'`를 유지한다.

첫 요청의 영상 ID `zoJC1SPEbJU`는 커밋 `3973ed7`에 기록됐고, 배포 전에 받은
후속 지시에 따라 커밋 `ce0ef4c`에서 `8y2ln1nCHbM`으로 교체했다. 최종 diff에는
후속 영상 ID만 남는다.

## 빌드와 운영 확인

| 항목 | 결과 |
|---|---|
| macOS arm64 Mediator 빌드 | 성공, `ce0ef4c154c1c15d0c78ed3b788469f966f1a1cb` |
| YouTube embed HTTP 응답 | 200 |
| 로컬 settings 응답 | 최종 embed URL과 일치 |
| 공개 settings 응답 | 최종 embed URL과 일치 |
| 공개 데모 CSP | `frame-src https://www.youtube.com` |
| Mediator | PID `61883`, `127.0.0.1:8080` 리슨 |
| Cloudflare 터널 | 기존 PID `70718` 유지 |
| 이전 바이너리 백업 | `/Users/runixs/enode-public-demo/backup-youtube-webcam-20260909T065344Z` |

공개 경로는
`https://deutsche-football-tract-necklace.trycloudflare.com/ui/demo/`다. 설정·토큰·
DB·runctl·터널·워커는 변경하지 않았다.

## 검사 범위와 보류

사용자의 선행 테스트 생략 지시에 따라 Node·Go 단위/통합/브라우저 테스트
스위트는 실행하지 않았다. `go build`와 배포 후 settings·CSP·프로세스 응답만
확인했다. 따라서 테스트 상태는 **미실행**이며 CP10·CP11 또는 전체 UI 장면 게이트
통과로 기록하지 않는다. 방송 송출 상태와 YouTube 측 embed 허용은 외부 라이브
상태에 따른다.

## 보안 확장

SECURITY-04·05·08·09·11·13·15를 준수한다. 공개 HTTPS URL을 기존 Go·브라우저
allowlist로 검사하고, 자격 증명 query 없이 YouTube frame origin 하나만 CSP에
추가하며 기존 fail-closed 처리를 유지한다. SECURITY-01·02·03·06·07·10·12·14는
해당 저장·중계·로그·권한·네트워크·의존·인증·모니터링 변경이 없어 N/A다.
차단 보안 소견은 없다.
