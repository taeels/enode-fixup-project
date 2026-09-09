# 영상 출력과 검증 기록

2026-09-09 · runixs · [문서 인덱스](../README.md).

## 출력 결과

아래 값은 완성된 로컬 파일에서 확인했다. 문서 정리 시 MP4 3개와 실제 UI 소스
복사본 5개의 SHA-256을 다시 대조했다. 모든 경로의 기준은 저장소 루트다.

최종 파일 디렉터리는 `output/enode-video/continuation-45s-v1/remotion/out/`다.

| 파일 | 길이 | 영상 프레임 | 파일 크기 |
|---|---:|---:|---:|
| `enode-full-60s-720p.mp4` | 60.000000초 | 1440 | 7,155,725바이트 |
| `enode-approval-15s-720p.mp4` | 15.000000초 | 360 | 2,893,743바이트 |
| `enode-after-approval-30s-720p.mp4` | 30.000000초 | 720 | 1,232,630바이트 |

세 파일 모두 1280×720, 24fps, H.264 영상과 AAC 음성이다. 각 파일의 전체 영상·
오디오 디코딩을 제작 완료 시 확인했다. 60초 원시 렌더의 AAC 패딩 약 53ms는
영상 프레임을 유지하고 오디오 끝만 정리했다. 원시 파일은 최종 전달 파일과 구분한다.

| 파일 | SHA-256 |
|---|---|
| `enode-full-60s-720p.mp4` | `1889475d2527b5f29cd54febbef8645447fd81d6e757ef108df77120e641afa4` |
| `enode-approval-15s-720p.mp4` | `16db5092042eae7bb85265550c9e3e413858f928352fd92b63c037aba7681011` |
| `enode-after-approval-30s-720p.mp4` | `b6fa577188e46c8ce6a5c3bd7c48a68059c309bfb66954cbd97334d2d2574f83` |

## 검사와 판정 범위

| 검사 | 결과·근거 |
|---|---|
| 제작 소스 | Remotion 프로젝트 TypeScript·ESLint 통과 |
| 실제 UI 캡처 | DashboardView의 8개 상태, 브라우저 페이지 오류 0 |
| 폰 화면 | 168프레임 화면 영역의 강한 chroma-green 잔여 최대 0픽셀 |
| 컷 연결 | 승인 opening 마지막 프레임을 후속 시작으로 사용. 개발실 64프레임부터 UI 합성 |
| 한국어 음성 | 앞 30초 Kling 대화와 뒤 30초 내레이션 문장이 자동 전사에 모두 나타남 |
| 오디오 | 전 구간 음성 스트림 디코딩, 최대 진폭 1 미만. 무음 트랙으로 대체하지 않음 |
| 자산 보존 | 채택 시트와 원본 영상 보존, 파생·합성·최종 파일 구분 |
| 문서 정리 | 파일 해시 재대조, Markdown 내부 링크·코드블록·표기 및 커밋 범위 검사 |

감사 로그에 보존된 사용자 원문의 두 줄에는 끝 공백이 있다. 원문 보존 지시에
따라 이 두 공백을 지우지 않았다. 영상 문서와 state에는 일반 `git diff --check`를
적용하고, audit에는 이 두 줄만 대조한 뒤 끝 공백 항목을 제외해 검사한다.

자동 전사는 바이너리·퓨징·셋업 등 일부 단어를 다르게 인식했다. 네이티브 발음과
음색을 사람이 청취해 모두 통과한 것으로 기록하지 않는다. 첫 영상의 금색 투구
윗부분 프레이밍과 컷 길이는 설계 목표와 차이가 있다. 폰의 손가락·버튼 위치 차이는
사용자가 사소하다고 수용했으며 정확히 일치하는 탭으로 기술하지 않는다.

실제 S2 기동·퓨징·격리 실행과 공동 하드웨어·방송 장면은 이 영상 검사의 대상이
아니다. `requirements/scene-gates.md`의 CP0~CP11 판정을 영상 성공으로 갱신하지
않았다. 이번 문서 커밋에서는 제품 코드 변경이 없어 Go·DB 회귀 검사를 다시
실행하지 않는다. DB를 생략한 테스트를 장면 게이트 통과로 기록하지 않는다.

## 실제 UI 소스 출처

복사 당시 HEAD: `c3f5ed9075c896fc5d00a3d062b17b2aceed89ad`.
아래 파일을 `internal/api/ui/static/shared/fleet/`에서 그대로 복사했다.
원본 제품 UI를 수정한 커밋이 아니라 이 소스에 가상 관측 데이터를 넣은 영상이다.

| 파일 | SHA-256 |
|---|---|
| `view.mjs` | `47fc5d7ed3bc772c7288ca326a6db47716126515b264a1e1ad48600c5558c1ce` |
| `fleet.css` | `229f013bd043da7f9afb478710b6e46c259fccc999f281a9349a27cadc7ca012` |
| `scene.mjs` | `58b1f09b2857403cda56f21ed6fec815a7db60cf3e2005b8621dea0d214698fd` |
| `client.mjs` | `84ec9810ba7e61a80696f98ceb6e90e6d5a50927897836773a971c7e009a9825` |
| `model.mjs` | `394c52af05a12228e8f87bb3340ccd6ec0df3ec7bdd6f74746c19645821450e9` |

## 로컬 재현 명령

이미 제작한 폴더와 로컬 도구가 있는 환경에서 실행한다. 영상 소스·PNG·MP4·
폰트·Kling 원본·합성 도구는 이 문서 커밋에 포함하지 않으므로 Git clone만으로
재현되지는 않는다. 유료 영상 생성도 아래 명령에 포함하지 않는다.

`remotion/node_modules`는 기존 `output/enode-video/remotion-ui-sample/node_modules`를
가리키는 로컬 심볼릭 링크다. Python 환경은 `local/video-asr/`이며 PyAV·OpenCV·
Pillow를 사용한다. UI 캡처는 로컬 Chrome·Playwright, 내레이션은 macOS Yuna에
의존한다. 파일 전달 시 이 환경과 입력 자산을 함께 준비한다.

저장소 루트에서 영상 소스 검사·렌더·최종 출력 검사를 수행한다.

```sh
cd output/enode-video/continuation-45s-v1/remotion
npm run lint
node node_modules/@remotion/cli/remotion-cli.js render EnodeFullFilm out/enode-full-60s-render-raw.mp4 --codec=h264 --crf=18 --concurrency=3
cd ../../../..
local/video-asr/bin/python output/enode-video/continuation-45s-v1/finalize.py
```

이미 완성된 파일의 길이·디코딩·해시만 다시 확인할 때는 다음 명령을 쓴다.

```sh
local/video-asr/bin/python output/enode-video/continuation-45s-v1/verify_export.py --verify-only
```

로컬 근거 파일은 `continuation-45s-v1/verification.json`, `final-asr.json`,
`ui-source/manifest.json`, `ui-capture/capture-records.json`,
`clip02-phone-7s/compositing-report.json`이다. 이 문단의 상대 경로 기준은
`output/enode-video/`다. ASR 원문은 의미·누락 확인의 보조 자료로만 사용한다.

## 문서 변경의 확장 규칙 적용

기존 선택인 security-baseline 활성, resiliency-baseline·property-based-testing
비활성을 상속했다. 다시 선택하거나 애플리케이션의 예외 범위를 바꾸지 않았다.

SECURITY-01~15의 애플리케이션·인프라 구현 항목은 이번 문서 변경에 N/A다.
배포·저장소·인증·로그·네트워크·의존성·실행 코드가 변경되지 않는다. 문서에
API 키·토큰·업로드 티켓·서명된 자산 URL을 옮기지 않고 로컬 증거의 해시와
비밀이 아닌 경로·설정만 기록한다. 기존 제품의 보안 준수나 공동 게이트 완료를
이 문서 검사로 인증하지 않는다.
