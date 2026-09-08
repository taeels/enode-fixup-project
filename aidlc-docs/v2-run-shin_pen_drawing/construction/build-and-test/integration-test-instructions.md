# 통합 검사 — 아트보드 사이의 일관성과 전환

유닛이 하나라 유닛 간 통합은 없다. 여기서 보는 것은 **한 틀을 공유하는
아트보드들이 서로 어긋나지 않는가**와 **전환 표가 그림에 있는가**다.

## 시나리오 1: D2 의 복사본 여섯이 D2 와 같은 우 열을 갖는다

- **설명**: D3 · D3b · D3c · D3d · D4 · D5 는 D2 를 복사해 만들었다. 우 열
  (현재 작업 목록)의 글이 한 글자라도 다르면 복사 뒤 고친 것이 흘러 들어간 것이다
- **실행**:

```js
runTexts = id => Get(id, n => n.name==="Run Column" ? n.id : undefined).map(c => Get(c, n => n.type==="text" ? n.content : undefined).join("|"))[0]
base = runTexts("FN7dp")
for (const id of ["Sm65l","bnV1N","JViqt","Dfo8G","C9Suc","Db9Rf"]) Print(id, runTexts(id)===base)
```

- **기대**: 여섯 다 `true`

## 시나리오 2: 웹캠 창이 어느 장에서도 같은 자리에 있다

- **실행**: 아트보드마다 `Webcam Window` 의 bounds 를 찍는다

```js
for (const id of ["FN7dp","Sm65l","bnV1N","JViqt","Dfo8G","C9Suc","Db9Rf"]) Get(id, (n,c) => n.name==="Webcam Window" && Print(id, Math.round(c.bounds.x), Math.round(c.bounds.y), Math.round(c.bounds.width), Math.round(c.bounds.height)))
```

- **기대**: 전부 `24 576 360 360` (좌 패널 좌표계)

## 시나리오 3: 전환 표의 출발 버튼이 그림에 있다

frontend-components.md 6절의 전환 표를 따라 눈으로 본다:

```text
   D1  Guest 로그인        -> D2     D1 에 Guest Login Button
   D2  새 작업             -> D4     D2 에 New Task Button
   D2  Run 행 클릭         -> D5     D2 에 Run Row 인스턴스
   D3  다음 · 건너뛰기      -> D2     D3 넷에 Next · Skip. D3d 는 시작하기
   D4  LED · 사운드 · 닫기  -> D5/D2  D4 에 버튼 셋
   D5  함대로              -> D2     D5 에 Back Chip
```

## 시나리오 4: run 목록이 안 가린다 (수용 기준)

D5 export 를 열어 우 열의 Run 행 여섯과 안내 상자가 온전히 보이는지 본다.
D4 의 덮개는 모달이지 가림이 아니다 (business-rules.md R3) — 닫으면 그대로
돌아오는 것이므로 통과다.

## 환경

pen.dev 에 `enode-demo.pen` 이 열려 있으면 된다. 서비스 · DB 없음.
