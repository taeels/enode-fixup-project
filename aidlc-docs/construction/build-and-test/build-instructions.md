# 빌드 지침 — demo-pen

이 회차의 산출물은 `design/enode-demo.pen` 이다. 컴파일할 것이 없으므로
**빌드 = 시안을 여는 것과 PNG 를 내보내는 것**이다.

## 전제

- **도구**: pen.dev 앱 + pencil MCP (`~/.pencil/mcp/kiro/`). `execute` 는
  **편집기에 열린 문서**에만 건다 — `filePath` 는 열린 문서가 있으면 무시된다
- **의존**: 없음. Go 툴체인 불필요
- **환경 변수**: 없음
- **주의**: `.pen` 은 Read/Grep 하지 않는다 (pencil 규칙). 디스크 상태는 md5 로만 본다

## 단계

### 1. 시안 열기

pen.dev 에서 `design/enode-demo.pen` 을 연다. `get_app_state` 의 활성 문서가
그 경로여야 한다.

### 2. 저장 상태 확인

pen.dev 는 자동 저장하지 않는다. 편집기에서 Cmd+S 뒤에:

```bash
git status --short design/
```

`enode-demo.pen` 만 `M` 이어야 한다. `enode-ux.pen` 이 잡히면 안 된다.

### 3. PNG 내보내기 (2x)

```js
Export(["CBJwz","FN7dp","Sm65l","bnV1N","JViqt","Dfo8G","C9Suc","Db9Rf"], "png", "<임시 폴더>", {scale:2})
```

파일은 `<nodeId>.png` 로 나온다. 다음 이름으로 `design/exports/` 에 옮긴다:

```text
   CBJwz   D1-guest-login.png
   FN7dp   D2-fleet-3d.png
   Sm65l   D3-tour-1-webcam.png
   bnV1N   D3b-tour-2-fleet.png
   JViqt   D3c-tour-3-runs.png
   Dfo8G   D3d-tour-4-new-task.png
   C9Suc   D4-new-task-modal.png
   Db9Rf   D5-run-3d.png
```

### 4. 성공 확인

- `design/exports/D*.png` 여덟 장, 각 3840 x 2160
- 기존 `S*.png` · `C*.png` 열한 장은 그대로 (git status 에 안 잡힘)

## 문제가 나면

### execute 가 "A file needs to be open in the editor"
pen.dev 에 파일이 안 열렸다. 열고 다시 한다.

### 삭제 · 수정이 엉뚱한 파일에 갔다
열린 문서가 다른 파일이다. `get_app_state` 로 활성 문서를 먼저 본다. 원본이
망가졌으면 저장하지 말고 닫거나 `git checkout design/enode-ux.pen`.
