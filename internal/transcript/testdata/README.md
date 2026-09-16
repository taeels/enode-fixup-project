# `testdata/lines` — 줄 하나에 파일 하나

**실측한 원문이 아니다.** 실물 하네스(`claude 2.1.271`)를 돌려 **모양을 보고
손으로 지은 것**이다. 원문은 저장소에 안 싣는다 — 경로 · 세션 id · 토큰 수가
그대로 들어 있고, 그것을 픽스처로 두면 기록이 아니라 유출이다.

**고치면 실측과 갈린다.** 이 파일들의 키 이름과 모양은 하네스가 실제로 낸
것이고, 값만 우리가 지었다. 키를 고치는 것은 시험을 고치는 것이 아니라
**하네스가 무엇을 내는가에 대한 이 저장소의 기록을 고치는 것**이다. 하네스가
바뀌어서 고치는 것이면 그 판을 커밋 메시지에 적는다.

파일이 한 벌이다 — `internal/transcript` 와 `internal/enode` 양쪽 시험이 같은
파일을 읽는다. 두 벌로 두면 한쪽만 고쳐도 둘 다 초록이라 갈린 것을 아무도
못 본다.

**이름을 바꾸면 컴파일이 아니라 실행 때 깨진다.** `internal/enode` 의 시험이
`../transcript/testdata/lines/` 를 상대 경로로 읽으므로 `go list` 가 그 결합을
못 본다. 실패 메시지가 경로를 싣는 것으로 그 대가를 갚는다.

## 무엇이 들어 있나

```text
   양쪽이 쓰는 다섯 — logs_test.go 의 오늘 상수 그대로다
     init.json              system/init.  게이트가 head -1 로 읽는 그 줄
     assistant-tool.json    도구 호출 하나 + usage.  본문이 블록 안에도 최상위에도 있다
     assistant-text.json    thinking 과 text 블록.  도구를 안 부른 사건이다
     user-result.json       실패한 도구 결과.  is_error 가 참이다
     result.json            마지막 봉투

   internal/transcript 만 쓰는 열
     elided.json            enode.elided.  걷은 양의 집계 줄
     capped.json            enode.capped.  bytes 는 닿은 상한이다 (10 MiB)
     unknown-enode.json     모르는 enode.*.  raw 로 떨어져야 한다
     rate-limit.json        사상표에 없는 하네스 type.  raw 다
     hook-response.json     system 이되 init 이 아니다.  Sub 가 system/hook_response
     type-not-string.json   type 이 42 다.  plain 으로 떨어진다
     plain.txt              JSON 이 아니다.  plain 으로 떨어진다
     shell-assistant.json   Shell() 이 지은 껍데기.  원문이 아니다
     assistant-tool-id.json tool_use 에 id 가 있다.  붙이기의 짝 절반
     user-result-id.json    같은 id 의 tool_result.  붙이기의 나머지 절반
     user-result-array.json content 가 배열이다.  아래를 본다
```

## `user-result-array.json` — 이것만 내력이 다르다

나머지는 앞 단계가 이미 본 모양이고 이것은 **이 단계가 처음 실측한 것**이다.

앞 문서는 `tool_result` 의 `content` 를 문자열 하나로만 기록했고, 배열 꼴을
저장소가 한 번도 안 적었다. 2026-09-16 에 실물 하네스로 **이미지를 읽는 턴**을
돌려 배열이 실제로 온다는 것을 봤다.

```text
   본 것    content 가 배열이고 그 안에 image 블록 하나뿐이다.  text 원소가 0 이다
   뜻       배열 꼴에서 건질 본문이 없다.  파서가 Text 를 비우는 것이 정확하다
```

`tool_use_id` 는 **모든 `tool_result` 블록에 있었다** (문자열 꼴도 배열 꼴도).
R12 가 그 키의 존재에 안 기대는 것은 그대로 두고, 실측이 그것을 뒤집지 않았다는
사실만 적는다.

## 안 담는 것

```text
   실제 세션의 원문      경로 · 세션 id · 커밋 해시가 들어 있다
   1 MiB 넘는 줄        퍼즈 코퍼스가 시드로 먹는다.  느려지는 자리다
   한국어 값            픽스처 문자열은 영어다 (CONVENTIONS 2.1)
```
