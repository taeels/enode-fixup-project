# `checkpoint` — 기술 선택

저장소가 이미 정했다. 이 유닛은 새로 고르지 않는다 (NFR 계획 3.4 · `requirements.md` 5.6). 작성 2026-09-30T14:26:33Z.

| 자리 | 선택 | 근거 |
|---|---|---|
| 언어 | Go 1.26 | `go.mod:8`. 새 언어 0 |
| 모듈 | 새 모듈 0 | `requirements.md` 5.6 「새 의존 0」 |
| `internal/scratch` | 표준 라이브러리와 `golang.org/x/sys` v0.47.0 만 | `go.mod:14` · 봉인 `internal/panel/boundary_test.go:102` |
| 잠금 | `unix.Flock` — spool 잠금 (`.lock`) · 항목 잠금 (`.enode-session.lock`) | 세션 잠금과 같은 모양 (`scratch.HoldSession`) |
| 옮기기 | `unix.Renameat2(RENAME_NOREPLACE)` — upper 를 spool 로 | 오늘의 keep 과 같다 (`runc_overlay_linux.go:589`) |
| 여유와 inode | statfs — 바이트는 `freeBytes` (`disk_unix.go:7`) 를 넘겨 쓴다 · inode 는 `f_files` · `f_ffree` | 도구를 두 벌로 두지 않는다. inode 의 기본 (NFR 1절) |
| 자리 열기 | `O_NOFOLLOW` · lstat · fchmod | lower-state 의 상태 자리와 같다 (NFR C7 · C8) |
| ID | `crypto/rand` 6 바이트를 hex 12 글자 | FD 답 8 |
| 기록 | `encoding/json` · `os.CreateTemp` 뒤 `Rename` · fsync 없음 | NFR P4 |
| 측정 | 오늘의 trash-helper (`unshare --user --map-root-user --map-auto`) 에 측정 동작 | FD 흐름 5절 · 새 helper 0 |
| 시험 | 표준 `testing` — 단언 · 모킹 라이브러리 없음 · 표 시험 · 실제 namespace 는 `integration` 태그 | `requirements.md` 5.6 |
| 크로스 빌드 | `_unix` (또는 `_linux`) 와 `_other` 짝 · windows/amd64 · linux/arm · darwin/arm64 | 유닛 정의 10절. linux/arm 은 32비트 — statfs 칸의 타입을 lower-state 가 확인한 방식으로 |

**안 쓰는 것**

| 무엇 | 까닭 |
|---|---|
| project quota | 조회에 root 와 관리자 준비가 든다 (결정 2-8) |
| 보존 중 암호화 | 같은 내용이 lower 와 scratch 에 평문으로 있다 (결정 2-9) |
| 압축 · tar · 해시 | 보고 전 창에 크기에 비례하는 일을 들이지 않는다 (결정 2-4). portable format 은 순연 (ADR-076 §6) |
| 새 데이터베이스 · 색인 파일 | 기록은 항목 폴더마다 JSON 하나다. 항목 수는 몫과 TTL 이 막는다 (NFR S1) |
