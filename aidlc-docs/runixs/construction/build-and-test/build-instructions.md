# 갤러리 확장 빌드

범위는 runixs의 UI/demo-back 갤러리 확장과 최신 main 통합이다. 기존 공동 하드웨어
장면의 완료 판정은 포함하지 않는다. 운영은 Mac mini에서 실행하고 MacBook 서비스는
계속 중지한다. 적용 결과는 [요약](build-and-test-summary.md)에 기록한다.

## 준비와 명령

Go는 go.mod의 1.26.6, UI 검사는 Node 20 이상, 게시 worker는 Python 3 표준 라이브러리를
사용한다. DB 검사는 scripts/testdb.sh와 전용 PostgreSQL DB가 필요하다. 운영 DB를
테스트 DB로 지정하면 안 된다. 실제 VM 검증에는 기존 Lima·Bedrock 연결이 필요하다.

저장소 루트에서 실행한다.

```sh
go version
go vet ./...
go run ./scripts/glyphscan.go
test -z "$(gofmt -l cmd internal)"
go build ./...
GOOS=darwin GOARCH=arm64 go build -o /tmp/mediator-gallery ./cmd/mediator
GOOS=darwin GOARCH=arm64 go build -o /tmp/enode-gallery-macos ./cmd/enode
GOOS=linux GOARCH=arm64 go build -o /tmp/enode-gallery-linux ./cmd/enode
GOOS=windows GOARCH=amd64 go build -o /tmp/enode-gallery.exe ./cmd/enode
```

운영 바이너리는 internal/build.Commit·Date를 ldflags로 주입한다. Mac mini의 검증
작업 디렉터리는 `/Users/runixs/enode-gallery-build-20260909`, 결과는 `deploy/`다.
CI의 cross job이 Windows·Pi 2 빌드와 enodectl 네트워크 심볼 상한도 검사한다.

## 적용·복구

[설치 절차](../demo-back/code/gallery-implementation.md#운영-적용-순서)에 따라
별도 게시 계정을 설치한다. 자격 증명은 installer stdin으로만 전달하고 Git에 넣지
않는다. VM 노드를 graceful로 비운 후 바이너리·gallery 광고를 갱신하고, Mediator
설정의 demo_gallery를 켠다. 동일 DB·토큰·아티팩트·터널을 유지한다.

서버와 설정은 `/Users/runixs/enode-public-demo/backup-gallery-<UTC>/`에 보관한다.
새 라우트 상태 검사가 실패하면 새 Mediator가 종료된 것을 확인한 뒤 이전 바이너리와
설정을 복구하고 재시작한다. VM 원래 정책도 복원한다. 게시 시도 DB는 삭제하지 않는다.
