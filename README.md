# enode

AI-Native SDLC 실행 모델의 구현. `Mediator` · `enode` · `runctl` · `enodectl` 이
돈다. **그리고 이 저장소는 구현만이 아니다** — 그 위에 이번 회차를 도는 층이
얹혀 있고 3절이 그것을 편다.

---

# 1. 여기는 어디인가

**설계 정본은 여기가 아니다.** 서브모듈 [`enode-design`](enode-design) 의
`adr/` · `protocol/` 이 정본이고, **어긋나면
`enode-design/protocol/INVARIANTS.md` 가 이긴다.**

```bash
git submodule status                    # 어느 판에 고정돼 있나
ls enode-design/adr/*.md | wc -l        # ADR 이 몇 건인가
```

올리는 것은 진행자만 한다 (`git submodule update --remote`). 올리면
`requirements/canon.md` 의 표를 다시 본다.

**정본 구현 저장소는 `upstream` 원격이다.** 이 저장소는 거기서 갈라져 나왔다.
거리는 문서가 아니라 git 이 답한다 — 문서에 적으면 그 줄이 조용히 낡는다.

```bash
git remote -v
git log --oneline HEAD..upstream/main
```

---

# 2. 첫 10분 — 클론하고 무엇을 치나

```bash
git submodule update --init      # enode-design/ 을 받는다
eval "$(scripts/testdb.sh)"      # 시험용 Postgres 하나
go test ./...
```

**`scripts/testdb.sh` 는 Docker 로 띄우되, 이미 잡힌
`ENODE_TEST_DATABASE_URL` 이 있으면 그것을 그대로 되돌려 준다.** Docker 를 못
까는 기계는 Postgres 를 직접 깔고 그 DSN 을 셸에 박으면 된다 — 위 명령은 그대로
선다. 까는 절차는 `docs/testdb-setup.md` 에 WSL2 와 macOS 로 나눠 적었다.

**서브모듈을 안 받으면 `enode-design/` 이 빈 디렉터리다** — 1절이 「이긴다」고
한 `INVARIANTS.md` 가 통째로 없다.

**`ENODE_TEST_DATABASE_URL` 이 없으면 DB 테스트가 안 돈다. 다만 안 도는
방식이 갈린다** — `internal/api` 는 `t.Skip` 이고 `internal/store` ·
`cmd/mediator` 는 `t.Fatal` 이다. 뒤쪽은 재현하지 못하는 실패를 남기느니
그 자리에서 죽는 쪽을 골랐다. 초록인데 안 돈 것이 가장 나쁘므로 CI 에 스킵을
잡는 스텝이 따로 있고, 면제 목록은 `.ci-allowed-skips` 다 — 그 파일 머리말이
지금 비어 있는 이유를 적는다.

**실행파일은 각자 다른 사람을 위한 것이다.** 명령 목록의 정본은 이 문서가
아니라 각 도구의 usage 다.

```bash
ls cmd/
go run ./cmd/runctl              # 인자 없이 부르면 명령과 종료코드를 낸다
go run ./cmd/enodectl -h
```

```text
   cmd/mediator     매칭 · 임대 · 시퀀싱 · Record 봉인.  실행은 하지 않는다
   cmd/enode        노드 데몬.  광고 · claim 롱폴 · 하네스 실행.  밖으로 듣지 않는다
   cmd/runctl       계약을 내는 사람의 CLI.  무상태
   cmd/enodectl     노드를 가진 사람이 자기 기계에서 쓴다
   cmd/iapadapter   바깥 트래커와 함대를 잇는다.  바깥을 아는 부품은 이것뿐이다
```

설치 묶음은 `packaging/` 에 있고 (`ls packaging/`) `.github/workflows/` 의
package 와 release 가 그것을 돈다.

읽는 순서는 `CLAUDE.md` · `CONVENTIONS.md` · `requirements/` 다.

---

# 3. 이 저장소의 층

```text
   구현            cmd/  internal/  scripts/  packaging/  .github/
   AI-DLC          .aidlc/aidlc-rules/     판은 VERSION 이 진다
   요구와 화면      requirements/  design/
   산출물 자리      aidlc-docs/             지금 README.md 한 장뿐이다
   규약            CLAUDE.md  CONVENTIONS.md
   설계 정본        enode-design/           서브모듈
```

```bash
cat .aidlc/aidlc-rules/VERSION                  # 이 저장소가 도는 AI-DLC 판
ls internal/ cmd/                               # 패키지와 실행파일
grep -c 'mux.HandleFunc' internal/api/api.go    # 등록된 라우트. api.go 가 정본이다
```

**`aidlc-docs/` 가 비어 있는 것은 결함이 아니라 결론이다.** 이 저장소는
브라운필드이므로 AI-DLC 가 Reverse Engineering 부터 돌고, 그 산출물이 이
디렉터리의 첫 내용이 된다. 낡은 산출물을 미리 실으면 그 단계를 실행한 것이 아니라
물려받은 것이 되고, 그것은 코드가 움직인 만큼 조용히 거짓이 된다. 근거와 실제로
밟은 사고는 `aidlc-docs/README.md` 에 있다.

**`requirements/` 는 이번 회차의 요구이고 AI-DLC 가 그것을 입력으로 읽는다.**
`design/` 은 그 화면이다 — `enode-ux.pen` 이 원본, `exports/` 가 거기서 내보낸
그림, `index.html` 이 모아 보는 페이지, `design/README.md` 가 그리는 규칙이다.

---

# 4. 이번 회차가 만드는 것

**축은 셋이다** — 관측 · 제어 · 접수.

```text
   관측    중앙 현황판 · 호스트 제어판 · 하네스 트랜스크립트
   제어    대기열(QUEUED) · 소유자 자원 제어권(drain)
   접수    Mediator MCP
```

**값은 이 문서가 지지 않는다.** `requirements/enode-features.md` 가 요구를 지고
`requirements/decisions.md` 가 이미 정해진 값을 진다. 두 벌로 두면 갈린다.

가치의 고정점은 `requirements/scene-gates.md` 의 장면 하나다 — 노드 소유자가
퇴근한 동안 함대가 그 보드를 쓰고, 소유자가 돌아와 drain 으로 되찾고, 기다리던
Run 이 이어받는다. **완결성은 그 장면이 끝까지 도는가로 판단한다.**

**안 만드는 것은 `requirements/constraints.md` 가 범주로 적는다** — 선점 ·
공정성 정책 · 격리 강화 · 인증 등. 요구 문서가 말하지 않은 것을 「빠뜨린 것」이
아니라 「뺀 것」으로 읽게 하는 장치다.

---

# 5. 규약과 게이트

**`CONVENTIONS.md` 가 표기 · 언어 · 커밋을 진다.**

```text
   표기    강조는 마크다운 굵게로만.  코드에 장식 문자를 넣지 않는다
           집행기는 enode-design/scripts/emphasis-check.py 와 scripts/glyphscan.go
   언어    밖으로 나가는 것은 영어 — 에러 · 로그 · CLI 출력 · 테스트 · 프롬프트
           안에 남는 것은 한국어 — 주석 · 커밋 메시지 · testdata 의 기록
   커밋    유닛 하나가 커밋 하나.  진행자만 커밋한다.  main 은 손대지 않는다
```

**언어를 가르는 기준은 「코드냐」가 아니라 「밖으로 나가느냐」다.** 이 저장소는
에이전트를 돌리는 제품이라, 프롬프트가 한국어면 실행 중에 그 언어가 에이전트에게
다시 주입되고 그 산출이 저장소에 쌓인다.

**`CLAUDE.md` 가 AI-DLC 워크플로를 진다.** 사본을 두 벌로 두지 않고
`.aidlc/aidlc-rules/aws-aidlc-rules/core-workflow.md` 를 그대로 끌어 쓴다.

게이트는 둘이다.

```text
   장면 게이트   requirements/scene-gates.md 의 CP0 부터.  조각마다 실행 명령이 있고
                 초록이 아니면 다음 유닛을 착수하지 않는다
                 커밋 지점은 그 게이트가 초록이 된 뒤다 (CONVENTIONS.md 3.3)
   CI           .github/workflows/
```

```bash
ls .github/workflows/
grep -nE '^      - name:' .github/workflows/ci.yml   # 이 워크플로가 실제로 세는 것
```

게이트가 유닛 밖의 이유로 빨가면 `scene-gates.md` 의 보류로 적고 넘어간다.
**보류는 통과가 아니므로 커밋 지점도 아니다.**

---

# 6. 설계가 코드에 어떻게 박혀 있나

**불변식을 애플리케이션 규약이 아니라 기계가 강제한다.** 예 셋과 그 확인 명령이다.

**노드 하나에 임대 하나 — 기본키가 강제한다.**

```bash
sed -n '55,64p' internal/store/schema.sql
```

두 번째 Run 이 같은 노드를 잡으려 하면 코드가 아니라 DB 가 막는다. 그 충돌에
롤백을 붙여 「전부 아니면 전무」를 얻으므로 손으로 해제할 것이 없다.

**봉인된 Record 는 불변이다 — 쓰기 비트를 내린다.**

```bash
grep -n 'os.Chmod' internal/record/record.go
```

「고치지 않기로 한다」가 아니라 권한이 막는다. Record 를 DB 행이 아니라 디렉터리에
둔 논거가 이것이다 — 봉인된 디렉터리는 권한으로 불변이 되지만 봉인된 행은
애플리케이션 규약일 뿐이다. 대가도 같은 파일이 적어 둔다: 봉인은 삭제까지 막는다.

**스키마는 형식만 말한다 — 허용 어휘를 좁혀 나머지를 거절한다.**

```bash
sed -n '26,45p' internal/schema/schema.go
```

「에이전트가 정직하게 답했을 때 통과하지 못할 수 있으면 그건 판정이다」는 선을
산문으로 두면 새어나간다. 누군가 값의 크기로 재는 키워드를 쓰는 순간 기계적
판정이라는 원칙이 스키마를 통해 무너지므로, 허용 목록에 없는 것은 계약 검증에서
거절한다.

---

**이 문서가 안 하는 일** — 설계 논거는 `enode-design/adr/`, 실측 서사는 코드 주석과
ADR 이 진다. 요구의 값은 `requirements/` 가 진다. **그리고 이 문서는 수치를 문장에
박지 않는다** — 세는 명령만 적는다. 지난 판이 낡은 이유가 세지 않고 적은 숫자였다.
