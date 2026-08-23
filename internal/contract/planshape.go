package contract

// ★ 계획이 지을 단계의 모양 ★ (ADR-057)
//
// ★ 왜 필요한가 ★ — 문법(Grammar)은 규칙을 말하고 이것은 ★ 필드의 모양 ★ 을
// 말한다. 둘은 다른 층이다.
//
//	★ 실측 ★ (vm-scratch-6) 오케스트레이터가 과제 3,308자를 agent.task 에
//	적었다. 문법이 "agent 단계는 uses · agent · in · out · schema 를 쓴다" 라고
//	★ 이름만 나열했고 ★, 무엇을 담는지 안 말했다. 그래서 이름에서 유추했고
//	— "agent 단계니까 agent 필드에 할 일을 적는다" — ★ 자연스럽지만 틀렸다 ★.
//
// ★ 정답은 계약 안에 있었다 ★: plan_setup 자신이
// agent:{max_turns,…} · in:{prompt:…} 였다. 그런데 계획은 계약 전문을 못 본다
// (ADR-055 §3-C 가 판정 조건 누출 때문에 옳게 기각했다). ★ 그 기각과 함께
// 실례를 보여줄 통로도 잃었다 ★ — 이 파일이 그 자리를 메운다.
//
// ★ 스키마가 아니라 예시다 ★ — JSON Schema 를 생성해 심으면 계획의 스키마
// (계약 저자가 쓰는 schema.plan)와 두 벌이 되고, ADR-045 가 고친 병이 재발한다.
// 그래서 ★ 형태 하나를 실제 값으로 보여준다 ★.
const PlanShape = `### The shape of a step

A step carries the task in in.prompt. agent carries execution parameters only.

    { "id": "install_it", "uses": "mac", "needs": ["survey_it"],
      "agent": { "max_turns": 40, "ask": "never" },
      "in": { "prompt": "Install enode in the VM and start it.",
              "from": ["survey_it_report"] },
      "out": ["install_report"],
      "schema": { "install_report": { "type": "object",
                  "required": ["started"],
                  "properties": { "started": { "type": "boolean" },
                                  "caps_line": { "type": "string" } } } },
      "feedback": ["survey_it_report"] }

    { "id": "check_it", "uses": "mac", "needs": ["install_it"],
      "run": ["/bin/sh", "-c", "runctl capabilities > \"$OUT/caps\""],
      "out": ["caps"] }

### Where each thing goes

    in.prompt    the task, in words. ★ Without it the agent gets an empty request ★
    in.from      names of earlier outputs to place as files in $IN
    feedback     names of earlier outputs to paste into the prompt itself
    agent        max_turns, max_tokens, ask, model, harness — ★ nothing else ★
    see.ledger   "list" places a list of everything produced so far in $IN

Any other key under agent or in is rejected: the whole plan is refused with the
name of the offending field.

### in.from and feedback are not the same

    in.from    the agent opens the file if it wants to. Cheap for large outputs
    feedback   the text is already in the prompt. ★ The agent cannot miss it ★

Use feedback for what the step must read, in.from for what it may need.
`
