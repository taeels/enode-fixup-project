# 페르소나 넷

**셋은 잇는다.** 앞 두 회차가 세운 P1 노드 소유자 · P2 계약 작성자 · P3 진행자다. 같은
제품의 같은 사람이라 이름과 번호를 그대로 둔다.

**네 칸은 이 팩에 대고 다시 쓴다.** 무엇을 쓰고 · 보고 · 모르고 · 틀리면 어떻게 되는지는
팩마다 다르다. 앞 회차의 값을 그대로 두면 이 팩에서 거짓이 된다 — 앞 회차의 P1 은
「적을 것이 안 는다」였는데 이 팩에서는 `min_free_gb` 의 뜻과 checkpoint 정책이 그
사람 몫이다.

**하나를 더한다 — P4 굽기 담당** (`story-generation-plan.md` Q1 = B). 굽기는 기제로는
일반 Run 이지만(ADR-077 §2) 그 Run 이 형제를 drain 시켜 다른 사람의 Run 을 기다리게
한다. 기다리게 하는 쪽과 기다리는 쪽을 한 이름에 두면 그 갈등이 스토리에서 안 보인다.

---

## P1 노드 소유자

그 기계와 그 위의 lower 를 가진 사람이다. `enode.yaml` 과 정책 파일을 편집하는 유일한
사람이고(ADR-012 · ADR-015), **그 기계에 무엇이 남고 왜 함대에서 빠졌는지 알아야 하는
유일한 사람**이다.

```text
   무엇을 쓰나     enode.yaml 의 min_free_gb · scratch 자리 · checkpoint 정책 (TTL · 보존 용량)
                  정책 파일의 drain.  environment profile 이 가리키는 lower
   무엇을 보나     enode env check 의 ready 와 not ready 의 사유
                  GET /v1/nodes 의 draining.  제어판
                  <scratch>/trash · spool · lower 루트의 .enode-metadata.json
   무엇을 모르나   함대에서 빠진 자기 노드의 drain 을 누가 걸었는지 — 자기인지, 여유 부족인지,
                  형제의 굽기인지.  자기 것만 파일에서 지워야 풀린다 (ADR-063 §4)
                  디스크 중 얼마가 곧 지워질 trash 이고 얼마가 보존 중인 checkpoint 인지
                  켠 적 없이도 실패한 단계의 upper 가 48시간 남는다는 것 (decisions 2-10)
                  min_free_gb 의 뜻이 바뀌었다는 것.  예시 설정 넷이 아직 옛 뜻을 적는다
   틀리면         저절로 풀릴 drain 을 고장으로 읽고 디스크를 늘린다.
                  또는 upper 에 남은 도구의 자격증명 캐시를 모른 채 기계를 넘긴다
```

## P2 계약 작성자

Run 을 내는 사람이다. `runctl` 로 직접 내거나 오케스트레이터가 대신 낸다. **이 팩에서
이 사람은 계약을 안 고쳐도 결과가 바뀐다** — effect 의 기본값이 수확을 좁힌다.

```text
   무엇을 쓰나     단계의 effect · Finalize 예산 · 업로드 예산 · success_when.changed
                  bounded discovery 를 켜는 것 (FR-1)
   무엇을 보나     진행 조회의 state · phase · phase_since · exit
                  finalize_timeout · upload_timeout.  receipt 의 checkpoint_capture
                  workspace.changed 와 그 뒤를 잇는 진단 자리 (FR-1)
   무엇을 모르나   QUEUED 인 자기 Run 이 굽기 drain 뒤에 서 있는지 그냥 후보가 바쁜지
                  effect 를 안 적은 명령 단계가 어제 내던 workspace.diff 를 오늘 안 낸다는 것
                  captured 된 보존본이 그 노드에만 있고 inspect-only 이며 48시간 뒤 사라진다는 것
   틀리면         Mediator 가 멈춘 줄 알고 Run 을 취소하고 다시 낸다.
                  source 를 고치는 명령 단계가 아무 결과도 안 낸 것을 성공으로 읽는다
```

## P3 진행자

회차를 돌리고 게이트를 집행하는 사람이다. **이 회차에서 유닛은 에이전트가 구현하고 사람
조각은 이 사람이 돈다** — 그래서 `scene-gates.md` 머리의 집행자 자격(그 유닛을 구현하지
않은 사람)을 만족한다 (`requirements.md` 6절 끝).

```text
   무엇을 쓰나     없다.  게이트 명령을 돌리는 사람이지 계약을 짓는 사람이 아니다
   무엇을 보나     사람 조각 여덟 — 2 · 4 · 6 · 8 · 9 · 10 · 11 · 12
                  진행 조회 · 봉인된 Record · 합치기 전후의 lower 목록 · GET /v1/capabilities
   무엇을 모르나   조각 6 에서 merge 가 waiting 일 때 형제의 어느 Run 이 lower 를 쥐고 있는지
                  굽기 계약이 앉을 노드의 lower 가 버려도 되는 것인지
   틀리면         합치기 조각을 운영 lower 위에서 돌린다.  SunnyVM 의 /srv/yocto 는 되돌릴 수 없다.
                  또는 SunnyVM 이 꺼져 못 돈 조각을 초록으로 적는다
```

## P4 굽기 담당 — 이 회차가 더한다

한 lower 의 신선도를 책임지는 사람이다. 굽기 계약을 쓰고 그것을 주기로 내는 계기를
세운다. **노드 기계에 들어갈 수 있다고 가정하지 않는다** — 사내에서 굽기를 세우는
사람과 노드를 가진 사람이 같다는 보장이 없다. SunnyVM 에서는 한 사람이 넷을 다 한다.

```text
   무엇을 쓰나     굽기 계약 — build 단계의 sync · builds[] (name · command), merge 단계의 대기 상한
                  그 계약을 주기로 내는 계기.  계기는 enode 밖이다
   무엇을 보나     build 단계의 결과 manifest (항목마다 시작 · 끝 시각과 exit code)
                  merge 단계의 waiting · bake_in_progress · merge_wait_timeout
                  GET /v1/capabilities 의 ir 과 repo.built.<name>.  봉인된 Record
   무엇을 모르나   merge 가 누구의 Run 을 기다리는지, 대기 상한까지 얼마 남았는지
                  FAILED 로 봉인된 자기 굽기 Run 이 재개로 lower 에 합쳐졌는지 —
                  그 흔적은 노드 기계 안의 metadata 에만 있다 (ADR-077 §7)
                  굽기가 성공했는데 ir 이 광고되지 않는 이유
   틀리면         이미 새 IR 에 선 lower 를 옛 IR 로 믿고 같은 굽기를 다시 낸다.
                  merge_wait_timeout 을 빌드 실패로 읽고 멀쩡한 빌드를 고친다
```

---

**넷 말고는 없다.** 형제 노드는 사람이 아니라 기계다. 형제에 Run 을 낸 사람이 P2 다.
하네스도 페르소나가 아니다 — 우리가 준 것만 읽는 프로세스이고, 이 팩에서 그 프로세스가
읽는 `workspace.changed` 의 변화는 그 훅을 쓰는 P2 의 일로 적는다.
