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
const Grammar = `## Contract grammar (the steps you write must follow these rules)

A plan that breaks a rule is rejected as a whole: schema validation returns 422,
or contract validation returns 400. There is no partial acceptance.

### There are four kinds of step, with different fields

    run       uses, run (argv array), out
    agent     uses, agent, in, out, schema
    ask       no uses, ask, out (exactly one), schema (required)
    acquire   no uses, acquire; performed by the mediator, not by a node

expands is not a kind. It is a flag on an agent step that makes it build a plan:

    uses, agent, expands:true, in, out (exactly one), schema (required)

An ask step must not set uses; it is performed by a person.

### Choosing between run and agent

Use a run step when the command is known before the run starts, and an agent
step when the decision has to be made on the machine that does the work.

    run     "west build -b nucleo_h753zi app"      the command is known
    agent   "install enode in the VM and start it" the path is not known yet

An agent step is not a weaker run step. It is judged by produced instead of
exit_code because a harness can exit 0 after doing nothing, not because the
step matters less. When a survey has to look at what it finds and decide what
to do next, an agent step does it in one step; a chain of run steps needs one
plan version per question asked.

The node running an agent step sees the machine. The plan does not.

### Two ways to branch

    dispatch   pick a named destination from a value in an output
    acquire    take a resource at run time; branch on whether it was taken

Both let one plan cover paths that are not known yet, so a survey and the work
that depends on it can live in the same plan.

    { "id":"survey", "uses":"n", "run":["/bin/sh","-c","..."], "out":["found"],
      "schema":{ "found":{ "type":"object", "required":["next"],
                 "properties":{ "next":{ "enum":["use_local","build_it"] } } } },
      "dispatch":{ "from":"found.next", "to":["use_local","build_it"] } }

    { "id":"get_board",
      "acquire":{ "want":{ "as":"board", "capability":"agent.reason",
                           "board":"stm32h753", "os":"linux" },
                  "acquired":"on_board", "unavailable":"qemu_only" } }

capability is a closed vocabulary: agent.reason and orchestration. Everything
else — board, os, arch, harness, repo — is an attribute, and the match is a
subset match on those attributes.

Rules for both:

    dispatch.from must name a field inside an output this step produces
    dispatch.to needs at least two targets, all distinct
    every branch target must exist and come after this step
    the value is also constrained by the schema enum

Branch targets are siblings, not a chain: the path not taken is skipped, and
conditions on a skipped step are vacuously true.

### Ask for the smallest set of resources up front

requires is taken all-or-nothing before the run starts, so a resource listed
there must be available before anything is known. List only what the survey
needs, and take the rest with acquire once the survey has answered.

    requires    the node that surveys, and the node that plans
    acquire     the board, the builder, whatever the survey turns out to need

acquire branches on availability, so a resource that is missing sends the run
down another path instead of killing it.

A role named in acquire.want.as becomes usable by the steps that follow it, so
a plan may introduce a role the contract never declared. Roles that come from
requires are fixed: uses may only name a role that requires declared or that an
earlier acquire in the same plan introduced.

### success_when conditions differ by step kind

    run                    exit_code and produced
    agent, ask, acquire    produced only

exit_code is allowed only on a run step. A harness can produce nonsense and
still exit 0, and a person's answer and a resource acquisition have no process
at all, so success is judged by what was produced.

success_when may only refer to steps that exist.

### Outputs go to $OUT under the exact declared name

    ok   cmd > "$OUT/<name declared in out>" 2>&1
    bad  cmd > <name> 2>&1        # lands in the workspace and is not collected

The working directory of a run step is the workspace; $OUT is outside it.

### run is an argv array; no shell is involved

'&&', '|', '>' and '$VAR' are not expanded. Invoke a shell explicitly if needed:

    ["/bin/sh", "-c", "..."]

### A schema constrains form only

    allowed   type, required, properties, enum, items,
              additionalProperties, title, description
    rejected  minimum, maximum, minLength, maxLength,
              pattern, format, minItems, maxItems, ...

Judging by magnitude or content is the contract's job, not the schema's.

### schema is keyed by output name

    "out": ["plan2"],
    "schema": { "plan2": { "type": "object", ... } }

Not the JSON Schema directly.

### A proposed success_when needs an ask that adopts it

    { "id":"replan_1", "uses":"...", "expands":true, "out":["plan2"],
      "schema":{ "plan2": { ... } } }
    { "id":"approve_replan_1", "ask":{ "adopts":"replan_1", "prompt":"..." },
      "out":["approval2"], "schema":{ "approval2":{ "type":"object",
        "required":["verdict"],
        "properties":{ "verdict":{"enum":["approve","reject"]} } } } }

Success criteria may be authored by a machine, but they take effect only
through a person's answer. Without an adopting ask the plan is rejected.

### The contract may promise steps you must build

If the prompt lists steps promised by the contract but not yet built, you must
build steps with those exact ids. success_when already refers to them, and a
plan that omits them is rejected. While any promise is outstanding you cannot
submit an empty plan.

### If there is nothing to fix, submit an empty plan

    { "steps": [], "success_when": [] }

This is a judgment, not an error. The contract is not extended and the ask that
would adopt it is skipped. Do not invent steps to fill the plan.

### If the goal was not reached, say so

An empty plan ends the run, and the run then succeeds if success_when holds.
success_when checks what a machine can check: an exit code, a file that exists,
a path that changed. It can hold while the goal is still not reached — a survey
that reports "nothing is installed" is a file that exists.

When that happens, write the file named in the prompt. A plan file is not
required alongside it; an empty plan is harmless but says the same thing twice.
A plan with steps in it is a contradiction and is rejected — either something
further will help, or nothing will. Failing to build a step the contract
promised is also a goal that was not reached.

Only a step that builds a plan may write it: judging the goal needs the whole
picture, and only that step is given it. From any other step the file is
recorded and ignored.

This can make a passing run fail. It can never make a failing run pass.
Success criteria stay where the contract put them.
`
