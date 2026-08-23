package contract

// Grammar 는 ★ 계획을 짓는 쪽이 알아야 할 계약 문법 ★ 이다 (ADR-045).
//
// ★ 왜 여기 있나 ★
//
// 이 규칙들은 전부 Validate() 가 이미 강제한다. 그런데 계획을 짓는 것은
// ★ 기계 ★ 이고(ADR-022 계획 위임), 그 기계에게 규칙을 알려주는 경로가 없었다.
// 그래서 사람이 매 판 Validate() 의 일부를 자연어로 번역해 프롬프트에 실었고,
// 번역은 ★ 축약되고 · 퇴행하고 · 모순됐다 ★:
//
//	실측  3차  예시에서 "$OUT/" 을 빼고 적었다 → 계획이 그대로 빼고 지었다
//	실측  4차  DOCKER_HOST 를 언급 안 했다     → 맞게 지어졌던 부분이 퇴행했다
//	실측  6차  ask 단계에 uses 를 요구했다      → Validate 가 금지한다. 422 로 계획 전체가 버려졌다
//	실측  8차  agent 단계에 exit_code 를 요구했다 → 승인 시점에 터졌고 회복 경로가 없었다
//
// ★ ADR-012 가 이미 푼 문제와 같다 ★ — capability 어휘가 광고로만 존재해서
// "읽는 경로가 없으면 계약을 쓰는 쪽이 문자열을 추측한다" 였고, 답이
// `runctl capabilities` 였다. 여기서는 ★ 계약 문법을 읽는 경로 ★ 가 없었다.
//
// ★ 아는 쪽이 적어준다 ★ — agent.go 의 outContract 가 배출 규약을 모든 에이전트
// 단계에 심는 것과 같은 자리다. 주석이 이미 답을 적어뒀다:
// "어댑터는 경로를 아는데 모델은 모른다. 아는 쪽이 적어준다."
//
// ★ stale 을 테스트로 막는다 ★ — grammar_test.go 가 이 문장 하나하나에 대해
// "그 규칙을 어긴 계약이 실제로 거절되는가" 를 잰다. Validate 가 늘었는데
// 여기가 안 늘면 ★ 그 테스트가 깨진다 ★. go.mod 의 toolchain 을 CI 가
// go-version-file 로 읽는 것과 같은 장치다 — ★ 두 곳에 안 적는다 ★.
const Grammar = `## 계약 문법 (★ 네가 짓는 단계는 이 규칙을 지켜야 채택된다 ★)

어긴 계획은 ★ 통째로 버려진다 ★ — 스키마 검증이 422 로 거절하거나
계약 검증이 400 으로 거절한다. 부분 채택은 없다.

### 단계는 네 종류이고 필드가 다르다

    명령(run)    uses ○   run ○(argv 배열)   out ○
    에이전트     uses ○   agent ○  in ○  out ○  schema ○
    재계획       uses ○   expands:true  agent ○  in ○  out ★ 정확히 하나 ★  schema ★ 필수 ★
    되묻기(ask)  ★ uses 를 적지 않는다 ★   ask ○   out ★ 정확히 하나 ★  schema ★ 필수 ★

★ ask 에 uses 를 적으면 거절된다 ★ — 사람이 수행하므로 노드가 없다.

### success_when 은 단계 종류마다 쓸 수 있는 조건이 다르다

    명령(run) 단계        exit_code ○   produced ○
    에이전트 · 재계획      exit_code ★ ✗ ★   produced ○
    되묻기(ask)           exit_code ★ ✗ ★   produced ○

★ 에이전트 단계에 exit_code 를 걸면 거절된다 ★ —
하네스는 헛소리를 하고도 종료코드 0 으로 끝난다. 그래서 성패는
★ 무엇을 냈는가 ★ 로만 잰다.

success_when 은 ★ 실존하는 단계 ★ 만 가리킬 수 있다.

### 산출물은 $OUT 에 그 이름 그대로 파일로 놓는다

    ○  cmd > "$OUT/<out 에 적은 이름>" 2>&1
    ✗  cmd > <이름> 2>&1        ← 워크스페이스에 떨어져 ★ 수확되지 않는다 ★

명령 단계의 작업 디렉터리는 워크스페이스다. $OUT 은 그 밖에 있다.

### run 은 argv 배열이다 — 셸을 안 거친다

'&&' · '|' · '>' · '$VAR' 가 안 풀린다. 셸이 필요하면 명시적으로 부른다:
    ["/bin/sh", "-c", "..."]

### 스키마는 형식만 제약한다

    쓸 수 있다  type · required · properties · enum · items ·
                additionalProperties · title · description
    ★ 거절된다 ★  minimum · maximum · minLength · maxLength ·
                pattern · format · minItems · maxItems …

값의 크기나 내용으로 판정하면 안 되기 때문이다 — 그건 계약이 할 일이다.

### 재계획이 success_when 을 제안하면 그것을 승인할 ask 가 있어야 한다

    { "id":"replan_1", "uses":"…", "expands":true, "out":["plan2"], "schema":{…} }
    { "id":"approve_replan_1", "ask":{ "adopts":"replan_1", "prompt":"…" },
      "out":["approval2"], "schema":{ "approval2":{ "type":"object",
        "required":["verdict"],
        "properties":{ "verdict":{"enum":["approve","reject"]} } } } }

★ 판정 기준의 저자는 기계일 수 있으나, 효력을 얻는 유일한 길은 사람의 답이다 ★.
승인할 ask 가 없으면 그 계획은 거절된다.

### ★ 계약이 「반드시 지어라」고 약속한 단계가 있을 수 있다 ★

프롬프트에 「계약이 약속했는데 아직 안 지어진 단계」 목록이 실려 있으면,
★ 그 이름을 가진 단계를 반드시 지어야 한다 ★. success_when 이 이미 그 이름을
가리키고 있고, 안 지으면 ★ 계획 전체가 거절된다 ★.

    { "id": "<약속된 이름>", "uses": "…", … }

★ 그것이 남아 있는 한 빈 계획을 낼 수 없다 ★ — 목표가 아직 안 섰다는 뜻이다.

### ★ 고칠 것이 없으면 빈 계획을 낸다 ★

    { "steps": [], "success_when": [] }

★ 이것은 오류가 아니라 판단이다 ★ — 계약이 안 늘고, 그것을 승인할 ask 는
건너뛰어진다. 억지로 단계를 지어내면 사슬이 끝나지 않는다.
`
