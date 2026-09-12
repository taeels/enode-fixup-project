# 오케스트레이션 — 무엇이 어떤 순서로 돌고 어디서 죽나

이 저장소에 서비스 계층은 없다. 도는 것은 **데몬 하나 안의 세 흐름**이다 —
한 단계의 실행 · 탐지 · 팩의 운반. 여기서는 그 셋의 순서와 **실패 지점**을 적는다.

---

## 1. 한 단계의 실행 — 치명이 전부 exec 앞에 모인다

```text
   claim.go                                              runner.go
   ────────                                              ────────
   워크스페이스 준비 · 기준 시각                 
   $OUT · $IN 을 짓는다 (워크스페이스 밖)
   $IN 을 0444 · 0555 로 잠근다 (봉인)
   프롬프트를 짓는다 · 자격증명을 준비한다
   Job 을 조립한다 (NodeMCP 를 실어 넘긴다)  ──▶
                                                 ①  계장 디렉터리를 짓는다
                                                    실패 -> 단계 실패
                                                 ②  resolveComponents(j)
                                                    실패 -> 단계 실패.  하네스 안 뜬다
                                                 ③  Argv 를 조립한다 (stream-json --verbose 포함)
                                                 ④  Instrument(tmp, self, a, c)
                                                    errAux 만 삼킨다 -> 그 밖은 단계 실패
                                                    삼킬 때도 플래그는 붙인다
                                                 ⑤  Fixed(tmp) 를 합친다
                                                    ── 여기까지가 닫는 자리 ──
                                                 ⑥  exec (cwd = 워크스페이스)
                                                 ⑦  Decode · Version · 자백
                                                 ⑧  MCP · Pack 을 결과에 채운다
                                                 ⑨  defer 가 계장을 지운다 (오늘 그대로)
   ◀──  로그와 HarnessResult
   $OUT 을 수확한다 · 보고한다
```

**②와 ④가 이 회차가 더하는 전부다.** ⑥ 뒤로는 한 줄도 안 는다.

### 1.1 ② 해소의 안쪽

```text
   팩이 요청됐나        agent.pack 이 있으면 $IN/<이름> 을 연다
     readPack           검증하면서 메모리로 읽는다.  거부는 오류다 (SEC-A)
     팩의 mcp.json      서버 맵으로 들어온다.  agent.mcp 가 적은 이름만 실린다 —
                        팩도 필터를 탄다 (application-design.md 4.4)

   노드 선언 중 요청된 것  agent.mcp 가 적은 이름만 Local.MCP 에서 집는다
   워크스페이스 중 요청된 것 agent.mcp 가 적은 이름만 <cwd>/.mcp.json 에서 집는다

   덮었나               노드가 선언한 이름을 팩이 덮으면 오류다 (4.5) —
                        pack redefines node-declared mcp server <이름>
   남은 이름이 있나      셋 어디에도 없으면 오류다 —
                        mcp server <이름> is not available on this node
   비었나               요청이 없으면 빈 목록이다.  팩이 실었어도 그렇다.  0 은 의도다
```

**워크스페이스 `.mcp.json` 은 여기서만 읽힌다.** 광고에는 안 실린다 — 같은 rev 의
노드마다 똑같아서 후보를 못 가른다 (`ADR-035` §4.1).

`project` 설정 소스는 **안 켠다.** 켜면 저장소의 `.claude/settings.json` 의 훅과
권한까지 딸려온다 (실측 F2).

### 1.2 ④ 쓰기의 안쪽 — 겹이 몇인가

```text
   <dir>/home/                 가장 먼저 짓는다.  비어 있다 (4.3 ①)
   <dir>/home/settings.json    훅 + 게이트웨이 인증 필드.  features.md 3.1 의 요구대로 홈 안이다
   <dir>/home/skills/…         팩의 스킬
   <dir>/home/agents/…         팩의 서브에이전트
   <dir>/home/.credentials.json  실제 홈에 있으면 0600 으로 복사.  없으면 아무것도 안 한다
   <dir>/mcp.json              해소된 서버 맵
```

**겹은 둘이고 첫 겹은 재측정을 기다린다** (`application-design.md` 4.6).

```text
   가짜 홈        CLAUDE_CONFIG_DIR 이 빈 디렉터리를 가리킨다.
                  개인 서버와 자동 메모리는 이것으로 끊긴다.
                  계정 커넥터는 로그아웃 홈에서만 쟀다 (features.md 1.2 의 실측표).
                  FR-1 이 .credentials.json 을 복사하므로 로그인된 홈에서
                  다시 재야 한다 — CA1 의 몫이다
   strict 허용목록 --strict-mcp-config 가 우리가 준 파일만 읽게 한다.
                  홈에 무엇이 생겨도 그 목록 밖은 안 뜬다.  계정 상태와 무관하다
                  (ADR-067 §2.1 「항상 켠다」)
```

그래서 이 플래그는 **보조 실패에도 안 떨어져야 한다** (4.3 ②). 그리고 CA1 은
「로그인된 가짜 홈에서도 커넥터가 안 보인다」를 재야 한다 — 그 답이 겹의 수를
정하고, 하나로 나오면 SECURITY-11 판정이 다시 나온다.

---

## 2. 탐지 — 광고 루프는 이 흐름을 안 탄다

```text
   Detector (고루틴, 5분)                 광고 루프 (하트비트 주기)
   ──────────────────────                 ────────────────────────
   refresh()                              Capabilities()
     costlyAttrs(ctx, l, log)               cheapAttrs(...)        새로 본다
       fingerprinters 를 순회한다            마지막 costly 를 읽는다   안 기다린다
         harnessFP  Usable()                capabilities(...) 로 합친다
         repoFP     DetectRepo()          ──▶ 광고 봉투
         mcpFP      LookPath · LookupEnv
     잠금을 잡고 마지막 값을 갈아끼운다
```

**여기가 이 설계에서 가장 중요한 「안 바꾼다」다.** `mcpFP` 를 `cheapAttrs` 에
두거나 광고 루프에서 직접 부르면, 노드 선언 수만큼 파일시스템을 훑는 동안
하트비트가 함께 멈추고 노드가 프로세스도 로그도 정상인 채로 함대에서 사라진다.

**순회로 바꾸는 것은 이 갈래를 안 건드린다.** `Fingerprinter` 셋은 전부
`refresh()` 안에서 돌고 `Capabilities()` 는 마지막 값만 읽는다 — `ADR-035:246-248`
이 「오늘 동작을 안 바꾸고 된다」고 적은 그대로다.

탐지 주기가 5분이라는 것의 값 — 실행파일을 치우면 **최대 5분 동안** 광고에
남는다. 그 사이 잡힌 Run 은 `resolveComponents` 가 exec 전에 실패로 낸다.
조용히 틀리는 것이 아니라 늦게 아는 것이고, 늦게 아는 값으로 5분은 싸다.

---

## 3. 팩의 운반 — 새 전송이 0 인 이유

```text
   ①  계약의 첫 단계가 명령 단계다        run: ["curl", "-o", "$OUT/pack", "<주소>"]
                                          또는 git archive --remote
   ②  $OUT 이 수확된다                    이미 있는 blob 경로다
   ③  Mediator 가 blob 을 든다            새 라우트 0
   ④  다음 단계의 in.from 이 잇는다       ["fetch.pack"]
   ⑤  $IN 에 깔린다                       0444 · 0555 로 봉인된다
   ⑥  agent.pack 이 그 이름을 가리킨다    resolveComponents 가 $IN/<이름> 을 연다
   ⑦  검증을 통과한 것만 가짜 홈에 펴진다
```

**⑤ 의 봉인이 SEC-B 의 근거 하나다.** 받은 뒤에는 안 바뀐다. 그래서 실행 전
다이제스트 검증 없이 사후 기록(`HarnessResult.Pack`)으로 족하다.

오케스트레이터는 `executor.pack` 이 있을 때만 ① 과 ⑥ 을 템플릿에 더한다.
비면 오늘 그대로 돈다.

---

## 4. 계약이 흐르는 길 — 두 어휘가 어디서 갈리나

```text
   requires[].mcp.<이름>: "1"    매칭.  internal/match 의 매처가 광고와 대조한다
                                이 팩은 매처 코드를 안 건드린다 — 속성 키가 늘 뿐이다.
                                다만 정렬 기준이 attr 개수라 동작은 따로 잰다
                                (application-design.md 6.4)
                                없는 키를 적으면 422 (후보 0)

   agent.mcp: ["gerrit"]        실행.  그 노드 위에서 무엇을 열지
   agent.pack: "pack"           실행.  무엇을 실을지
                                모르는 키를 적으면 400 (contract.Validate)
```

`ADR-013` 결정 4 의 갈래 그대로다 — **매칭은 `requires` 에, 실행은 `agent` 에.**
둘을 한 자리에 두면 「그 기계가 가졌나」와 「이번에 쓰겠나」가 섞인다.

---

## 5. 실패했을 때 무엇이 남나

```text
   ① 에서 죽으면    아무것도 안 남는다.  계장 디렉터리조차 없다
   ② 에서 죽으면    계장 디렉터리만 있고 비어 있다.  defer 가 지운다
   ④ 에서 죽으면    반쯤 쓴 가짜 홈이 있고 defer 가 지운다.  하네스는 안 떴다
   ⑥ 뒤에 죽으면    오늘 그대로 — 봉투 · 자백 · 워크스페이스 diff 가 판정한다
```

**하네스가 뜬 뒤에는 이 회차의 어떤 실패 규칙도 발동하지 않는다.** 그것이
「치명을 exec 앞에 모은다」의 값이다 — 반쯤 일한 단계를 이 팩 때문에 죽이는 일이
없다.
