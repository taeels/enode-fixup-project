# 단위 검사 — 아트보드별

pen.dev 편집기에 `enode-demo.pen` 을 연 뒤 `execute` 로 돌린다. 아트보드
여덟을 각각 검사한다.

## 1. 실행

```js
boards = {CBJwz:"D1", FN7dp:"D2", Sm65l:"D3", bnV1N:"D3b", JViqt:"D3c", Dfo8G:"D3d", C9Suc:"D4", Db9Rf:"D5"}
Print("placeholders:", Get(n => n.placeholder ? n.id : undefined).length)
ignore = /^(Tower|Stand|Monitor|Screen|Chip|Slot Marker|Edge|Dim|Spotlight|Balloon Tail|Floor)/
for (const [id, nm] of Object.entries(boards)) {
  const probs = Get(id, (n,c) => c.problems && !ignore.test(n.name||"") ? (n.name+":"+c.problems) : undefined)
  Print(nm, "problems:", probs.length, probs.join(" | "))
}
need = {D1:["Guest Login Button"], D2:["View Chip 격자","View Chip 그래프","View Chip 3D","Zoom In","Zoom Out","Fit","Resize Handle","New Task Button","Filter 전체","Filter RUNNING","Filter QUEUED","Filter SUCCEEDED","Filter FAILED","Run led-toggle-0417","Detail Panel","Tile ws-a","Tile board-01"], D3:["Next Button","Skip Button","Tour Balloon"], D3b:["Next Button","Skip Button"], D3c:["Next Button","Skip Button"], D3d:["Next Button","Skip Button"], D4:["Task LED Button","Task Sound Button","Close Button"], D5:["Back Chip","Step plan","Step flash.led","Step verify","Detail Panel","New Task Button","Zoom In","Resize Handle","Run led-toggle-0417"]}
for (const [id, nm] of Object.entries(boards)) {
  const names = new Set(Get(id, n => n.name))
  const missing = (need[nm]||[]).filter(x => !names.has(x))
  Print(nm, "buttons missing:", missing.length ? missing.join(",") : "none")
}
bad = /[*_~`#]{2}|[\u2605\u2606\u203B\u25C6\u25A0\u25B6\u25BA\u2714\u2713]/
badEnum = /\b(SEALED|RESOLVING|ALLOCATING|CREATED)\b/
for (const [id, nm] of Object.entries(boards)) {
  Print(nm, "decorative:", Get(id, n => n.type==="text" && bad.test(n.content||"") ? 1 : undefined).length,
            "bad enum:", Get(id, n => n.type==="text" && badEnum.test(n.content||"") ? 1 : undefined).length)
}
Print("D3d next label:", Get("Dfo8G", n => n.name==="Next Label" ? n.content : undefined)[0])
```

## 2. 기대값

```text
   placeholders            0
   problems (필터 뒤)       아트보드마다 0
   buttons missing         아트보드마다 none  — 사양의 버튼 전수 (frontend-components.md)
   decorative              0  — CONVENTIONS.md 1.2
   bad enum                0  — SEALED 등 저장되지 않는 상태 어휘 (design/README.md 6절)
   D3d next label          시작하기
```

`ignore` 에 든 이름은 원본 S6 · S7 부터 1px 넘치는 모형 path · 평면 밖으로
뻗는 간선 · 아트보드를 덮는 덮개다. 그것은 결함이 아니다.

## 3. 실패하면

- **problems**: 그 노드를 `Get(id,{depth:1})` 로 읽어 `textGrowth` · `width` 를
  본다. 한 줄 텍스트가 부모를 넘치면 `textGrowth:"fixed-width"` +
  `width:"fill_container"` 로 감싼다 (2026-09-08 D4 Task Sub 가 이 경우였다)
- **buttons missing**: 이름이 바뀌었는지 먼저 본다. 사양의 버튼이 정말 없으면
  Code Generation 으로 돌아간다
- **decorative / bad enum**: 텍스트를 고친다. 규약이 우선이다
