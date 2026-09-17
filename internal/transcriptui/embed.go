// Package transcriptui 는 트랜스크립트 카드를 그리는 .mjs 한 장을 들고만 있다.
//
// 제어판(internal/panel)과 현황판(internal/api/ui)이 둘 다 이 패키지를
// 임포트해 각자의 라우트로 같은 바이트를 낸다. 그래서 이 패키지는 잎이어야
// 한다 - 무언가를 임포트하면 두 화면이 그 의존을 통해 다시 붙고, 접은 두
// 벌이 뒤로 돌아온다. 금지 넷(panel · api · store · enode)을
// internal/panel/boundary_test.go 가 진다.
//
// 함수가 0 개인 것은 우연이 아니다. CI 의 커버리지 게이트가 패키지마다 80%
// 하한을 거는데, 문장이 하나라도 있으면 그 문장을 덮는 시험이 이 패키지에
// 따로 있어야 한다. 파일 하나를 내는 데 필요한 것은 변수뿐이므로 문장을
// 0 으로 두고, 그러면 이 패키지가 커버리지 표에 아예 안 나온다.
package transcriptui

import "embed"

// Files 는 card.mjs 하나를 든다. 부르는 쪽이 fs.Sub 나 http.FS 로 낸다.
//
// []byte 가 아니라 embed.FS 인 이유 - 두 서버가 각자 http.FileServerFS 로
// 내면 Content-Type 과 If-Modified-Since 를 표준 라이브러리가 지고, 그 둘을
// 두 자리에서 따로 지으면 같은 파일이 화면마다 다른 헤더로 나간다.
//
//go:embed card.mjs
var Files embed.FS
