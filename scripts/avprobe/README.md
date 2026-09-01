# avprobe — 백신이 무엇을 보고 지웠는지 가른다

`v0.0.1-rc13` 의 `enodectl.exe` 가 윈도우에서 풀린 직후 삭제됐다. 진단명은
`Trojan/Win.Generic.C5874069` (AhnLab V3) 이고, 처리는 격리가 아니라 삭제였다.
`rc7` 과 `rc9` 는 같은 기계에서 멀쩡했다.

## 무엇이 달라졌나

`rc7` 과 `rc9` 의 `enodectl.exe` 는 **바이트 단위로 같다.** `rc13` 에서만 바뀐다.

```text
                  크기          crypto/tls 심볼
   rc7          6,223,360             24
   rc9          6,223,360             24
   rc13        10,242,560          1,068
```

늘어난 4MB 는 전부 `net/http` 와 `crypto/tls` 다. `cmd/enodectl/setup.go`
열아홉 줄이 `enode.SetupCLI` 를 부르고, 그것이 `internal/enode` 전체를 링크
대상으로 끌어들이면서 따라 들어왔다. 같은 구간에서 `runctl.exe` 는 바이트
단위로 변하지 않았고 `enode.exe` 는 0.2MB 만 늘었다.

## 그래서 무엇이 방아쇠인가

`setup` 코드 자체에는 의심받을 것이 없다. `http.Client` 를 만들어 GET 을 한 번
보내는 것이 전부다. 의심받는 쪽은 `cmd/enodectl/main.go` 가 `rc7` 부터 갖고
있던 프로세스 관리 코드이고, `setup` 이 한 일은 **그 코드에 「TLS 로 바깥과
통신한다」는 맥락을 붙인 것**이다.

```text
   파일을 쓴다                 설정을 만든다
   프로세스를 떼어내어 띄운다   DETACHED_PROCESS
   남의 프로세스를 죽인다       OpenProcess + TerminateProcess
   바깥과 암호화 통신한다       rc13 에서 새로 생겼다
```

넷이 한 실행 파일에 모이면 제네릭 규칙이 반응한다. 다만 이것은 아직 추정이다.
제네릭 규칙의 내부를 볼 수 없으므로 **실험으로 가른다.**

## 탐침 넷

능력을 하나씩만 갖는 최소 프로그램이다.

```text
   neither    아무것도 없다                    순수 대조군
   netonly    net/http · crypto/tls 만
   proconly   OpenProcess · TerminateProcess · DETACHED_PROCESS 만
   both       둘 다                            rc13 의 enodectl 과 같은 조합
```

`proconly` 와 `both` 의 `TerminateProcess` 는 `len(os.Args) > 99` 라는 참이 될
수 없는 조건 뒤에 있다. 심볼은 바이너리에 들어가지만 실행되지 않는다. 실제로
하는 일은 자기 자신을 조회 권한으로 한 번 여는 것뿐이다.

## 쓰는 법

```sh
bash scripts/avprobe/build.sh
```

`scripts/avprobe/dist/` 에 넷과 `SHA256SUMS` 가 나온다. 윈도우의 같은 폴더에
넷을 함께 두고 몇 분 지켜본다.

```powershell
while ($true) {
  "{0}  neither={1} netonly={2} proconly={3} both={4}" -f (Get-Date -Format "HH:mm:ss"),
    (Test-Path .\neither.exe), (Test-Path .\netonly.exe),
    (Test-Path .\proconly.exe), (Test-Path .\both.exe)
  Start-Sleep -Seconds 2
}
```

## 판정

| 지워지는 것 | 방아쇠 | 네트워크를 걷어내면 |
|---|---|---|
| `both` 만 | 능력의 조합이다 | 풀린다 |
| `netonly` · `both` | 네트워크 단독으로 걸린다 | 풀린다 |
| `proconly` · `both` | 프로세스 조작 단독으로 걸린다 | **소용없다** |
| `neither` 까지 전부 | 서명 없는 Go 실행 파일 자체가 대상이다 | **소용없다** |
| 아무것도 안 지워짐 | 방아쇠가 다른 곳에 있다 | 다시 좁혀야 한다 |

아래 두 줄이 나오면 `enodectl setup` 을 `enode.exe` 에 위임하는 수리는 헛일이
된다. **고치기 전에 그 수리가 통할지 여기서 먼저 안다.**

## 왜 모듈을 따로 뒀나

`scripts/avprobe/go.mod` 가 있으므로 이 디렉터리는 메인 모듈의 `./...` 에
잡히지 않는다. `go vet` · `golangci-lint` · `govulncheck` · `go test` 와
커버리지 분모, 그리고 세 플랫폼 크로스 빌드 검사가 모두 이 코드를 안 본다.

진단용 일회성 도구를 게이트에 참여시키면 두 가지가 나쁘다. 커버리지 분모가
테스트할 이유가 없는 문장으로 불어나고, 능력 조합을 일부러 모아 둔 코드가
정적 검사의 경고를 부른다.

## 옮길 때

`zip` 으로 묶으면 푸는 순간 판정이 시작되어 실험이 시작 전에 끝난다. 확장자를
바꿔 옮긴 뒤 목적지에서 되돌린다. 검사를 피하는 것이 아니라 검사 시점을 실험
시작 시점에 맞추는 것이고, 되돌린 뒤에는 그대로 검사받는다.

사내 기계에서 백신 동작을 관찰하는 실험이므로 보안팀에 미리 알린다. 격리
이력이 중앙 콘솔에 남으면 그쪽에서 먼저 문의가 올 수 있다.

## 오탐 신고에 쓴다

「능력 조합만으로 걸린다」가 확인되면 그 자체가 최소 재현 사례다. 진단명과 함께
내면 규칙을 좁히기가 쉬워진다. 다음도 함께 적어 둔다.

```text
   go mod verify        all modules verified
   재현 빌드            같은 소스가 같은 SHA256 을 낸다
   rc9 → rc13 소스 변경 cmd/enodectl 생산 코드 101줄 추가 · 39줄 삭제
   새 의존성            없다. x/sync 와 x/text 판 올림뿐이다
```
