# Build Instructions — v1-run-dhseo-cardnews (유닛 cardnews-guest-login)

이 문서는 **이 실행의 스코프**(3.4.1 하나)만 다룬다. 저장소 전체의
빌드 게이트는 `requirements/scene-gates.md` §3 CP0 이 정본이다 — 여기는
그중 이 유닛이 실제로 건드는 부분만 반복한다.

## 사전 준비

```bash
cd enode-fixup-project
git switch unit/cardnews-guest-login   # 또는 병합 뒤 v1-run-dhseo-cardnews
```

Postgres 는 필요 없다 — 이 유닛은 `internal/store` 를 안 만진다.

## 빌드

```bash
go build ./...
```

**실행 결과 (2026-09-08)**: 통과. 새 패키지 `internal/api/ui` 를
포함해 전체가 빌드된다.

## 정적 검사

```bash
go vet ./...
gofmt -l .
go run ./scripts/glyphscan.go
```

**실행 결과**: 셋 다 통과 — `go vet` 지적 0, `gofmt -l .` 출력 없음,
`glyphscan` 83개 파일에서 장식 문자 0(이 유닛이 만든 13개 파일 포함).

## 크로스 빌드 (참고 — 이 유닛은 영향 없음)

`enodectl.exe` 심볼 상한 게이트(`constraints.md`)는 `cmd/enodectl` 을
빌드하는 게이트다. 이 유닛은 `cmd/enodectl` 을 안 건드리므로 상한 값에
변화가 없다 — 재측정하지 않는다(측정 대상이 안 바뀌었다).
