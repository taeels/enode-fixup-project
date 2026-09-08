# Services — 3.4 온보딩 카드뉴스 + Guest Login

## 서비스 계층이 왜 얇은가

이 축은 함대 상태를 읽지도 쓰지도 않는다 — 조회할 도메인이 없다.
그래서 오케스트레이션할 여러 서비스 간 협업이 존재하지 않는다. 서비스
계층은 하나뿐이고, 그것도 orchestration 이 아니라 파일 서빙이다.

## S-1 StaticAssetService (`internal/api/ui`)

- **책임**: `C-1 UIHandler` 가 이 서비스의 전부다. 요청 경로를
  `static/` 아래 파일에 매핑하고, 없으면 404, 있으면 그 파일과 함께
  보안 헤더를 낸다
- **오케스트레이션 없음**: 다른 내부 서비스(store · record · match)를
  하나도 부르지 않는다. `internal/api.Server` 가 이 서비스를
  `Handler()` 로 받아 자기 mux 에 등록만 한다 — 그 반대(이 서비스가
  `api.Server` 를 부르는 것)는 없다
- **트랜잭션 경계**: 없음. 상태를 안 바꾼다

## 명시적으로 안 두는 서비스

```text
   GuestSessionService   서버 쪽에 Guest 세션을 두지 않는다.  식별은
                         전부 클라이언트 로컬 저장소에만 있고 서버는
                         그 존재 자체를 모른다 (decisions.md 8.2)
   CardContentService    카드 문안은 정적 HTML/JS 에 박힌 상수다.
                         서버가 카드 목록을 내려주는 API 가 없다 —
                         있으면 새 라우트가 늘고 "새 백엔드 라우트를
                         최소화한다" 를 어긴다
```
