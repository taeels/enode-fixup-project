# 공개 Mediator의 노드 등록 경로 확인

2026-09-09 사용자가 제공한 공개 주소와 설정으로 확인했다. 토큰 값은 문서와
로컬 결과 파일에 저장하지 않는다. 로컬 Mediator·enode·DB는 중지 상태를 유지했다.

## 확인된 원인과 조치

설정의 `http://mediator.taeels.duckdns.org`는 openresty에서 HTTPS로 301 이동한다.
실제 `enode.Client.Advertise` 호출로 확인했을 때 `POST /v1/nodes`가 리다이렉트 후
`GET https://mediator.taeels.duckdns.org/v1/nodes`로 바뀐다. GET의 200 목록 응답은
AdvertResponse의 알 수 없는 필드로 무시돼, leases=null·renew_seconds=0과 오류 없음으로
반환된다. 등록 POST가 실행되지 않아 함대에 노드가 나타나지 않을 수 있다.

실패 PC의 기존 설정에서 주소를 아래처럼 바꾼 뒤 그 노드를 재시작한다.
토큰을 다시 발급할 필요는 없다. 전달받은 토큰으로 HTTPS 인증이 성공했다.

```yaml
mediator: https://mediator.taeels.duckdns.org
```

`local.yaml`인 경우 `enodectl stop local` 후 `enodectl start local`이다.
다른 이름의 설정은 그 이름을 사용한다. 이 PC의 노드를 대신 기동하지 않는다.

## 공개 서버 실측

| 요청 | 결과 |
|---|---|
| HTTP GET /v1/capabilities | 301, 같은 호스트 HTTPS로 이동 |
| HTTP POST /v1/nodes | 301, 같은 호스트 HTTPS로 이동 |
| HTTPS GET /v1/capabilities, 토큰 없음 | 401 |
| HTTPS GET /v1/capabilities, 사용자 토큰 | 200 |
| HTTPS POST /v1/nodes, 사용자 토큰과 빈 node_id | 400, node_id is missing |
| HTTPS GET /v1/nodes | 200, 조회 당시 노드 0개 |
| 실제 Go 클라이언트의 HTTP Advertise | GET으로 이동, 오류 없음·빈 광고 응답 |
| 같은 클라이언트의 HTTPS Advertise | 이동 없음, 빈 node_id에 대한 400 |

POST에는 의도적으로 유효한 node_id를 넣지 않았다. 등록 핸들러의 인증·유효성
검사까지 확인하고 DB 쓰기 전에 거부시켰으며 공개 함대에 가짜 노드를 추가하지 않았다.
자료는 ignored `local/public-registration-20260909/public-api-check.json`과
`public-client-check.jsonl`에 있다. HTTPS로 바꾼 실제 다른 PC의 등록 성공은
그 PC에서 재시작한 뒤 확인해야 한다.

설정의 `principal`이 없고 PC의 전역 git email도 없으면 주소 문제와 별도로
기동 시 신원 생성이 실패한다. 그 경우 로그에 `cannot derive node identity`가
있으며 설정에 PC 소유자의 `principal`을 넣는다. 현재 그 PC의 로그는 미수령이다.
빈 workspace 자체를 이번 등록 실패의 원인으로 단정하지 않는다.

제품 코드는 변경하지 않았다. 프록시의 301 대신 메서드 보존 리다이렉트를 쓰거나
클라이언트에서 메서드 변경을 거부하는 후속 보완은 서버·클라이언트 담당 검토 대상이다.
