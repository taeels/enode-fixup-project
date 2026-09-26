package enode

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/taeels/enode/internal/contract"
)

// 도는 동안의 원문을 Mediator 로 민다 (FR-5 의 노드 쪽).
//
// 링과 같은 tee 에 꽂힌다 — 규칙이 하나다: 링으로 가는 것은 Mediator 로도 간다.
// 그래서 중앙 화면이 제어판보다 적게 보이는 일이 없다.
//
// 이 겹의 첫째 규율은 "실행을 안 막는다" 이다. Write 가 네트워크를 기다리면
// io.MultiWriter 를 통해 하네스의 stdout 이 그 자리에서 멈추고, cmd.Run 까지
// 멈춘다. 관측하는 겹이 실행을 멈추면 그것은 관측이 아니다.

const (
	// uploadPeriod 는 새 바이트가 있을 때 미는 주기다 (decisions §2).
	uploadPeriod = 2 * time.Second
	// uploadNow 는 이만큼 쌓이면 주기를 안 기다린다 (decisions §2).
	uploadNow = 64 << 10
	// uploadBufferMax 는 아직 못 보낸 것의 상한이다.
	//
	// 아래에서 민 것 - 장면 게이트가 "Mediator 를 10초 멈췄다 켠다" 를 재고,
	// 그 10초의 출력이 여기 안 들어가면 게이트가 이 값 때문에 빨개진다.
	// 위에서 누른 것 - 노드의 힙이다. 링의 512 KiB 에 맞추는 편이 숫자가
	// 하나 줄어 좋으나, 그 이득보다 10초의 여유가 크다고 봤다.
	//
	// 상수인 이유 - 조절 손잡이가 아니라 보호다. 설정으로 열면 누군가 이것을
	// 키워 노드의 메모리로 실행을 위협하게 된다.
	uploadBufferMax = 1 << 20
)

// LogChunkAck 는 진행 청크를 민 뒤 서버가 말해 주는 것이다.
//
// 셋을 다 읽는 것이 이 겹의 값이다. UploadLog(단계 끝의 선별본)는 상태 코드만
// 보고 몸통을 버리는데, 이쪽은 셋이 각각 규율을 진다.
type LogChunkAck struct {
	// Total 은 서버가 든 그 파일의 총 길이다. 노드가 자기 오프셋을 여기 맞춘다 -
	// 자기가 몇 바이트 보냈는지를 안 믿는다. 상한에 닿아 일부만 받았을 때,
	// 서버가 앞 시도를 걷었을 때, 재전송이 겹쳤을 때 셋 다 노드의 셈과
	// 서버의 파일이 갈린다.
	Total int64
	// Attempt 는 서버가 실제로 쓴 시도다. 내 것보다 크면 내 청크가 버려진
	// 것이다 - 늦게 온 앞 시도의 청크를 서버가 0 바이트 쓰고 200 으로
	// 돌려주기 때문에, 이 값이 없으면 버려진 것을 성공으로 읽는다.
	Attempt int
	// Capped 는 총 길이 상한에 닿았다는 뜻이다. 표시 줄은 서버가 박으므로
	// 노드가 할 일은 멈추는 것뿐이다.
	Capped bool
}

// PutLogChunk 는 도는 동안의 원문 청크를 진행 파일에 붙인다.
//
// UploadLog 와 라우트가 같고 쿼리가 다르다 (progress=1). 파일이 다르므로 두
// 벌이 안 생긴다 - 하나는 봉인에 들어가고 하나는 봉인 때 지워진다.
func (c *Client) PutLogChunk(ctx context.Context, runID string, seq int,
	name string, attempt int, body []byte) (LogChunkAck, error) {
	url := fmt.Sprintf("%s/v1/runs/%s/steps/%d/log?name=%s&progress=1&attempt=%d",
		c.Base, runID, seq, name, attempt)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return LogChunkAck{}, err
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-Enode-Principal", c.Principal)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return LogChunkAck{}, err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode/100 != 2 {
		return LogChunkAck{}, fmt.Errorf("progress chunk rejected: %s", resp.Status)
	}
	ack := LogChunkAck{Capped: resp.Header.Get("X-Enode-Log-Capped") == "1"}
	// 파싱 실패를 오류로 안 올린다. 청크는 실제로 붙었고, 헤더를 못 읽은 것은
	// 다음 왕복이 고쳐 준다 - 붙은 것을 실패로 세면 같은 바이트를 또 보낸다.
	if n, err := strconv.ParseInt(resp.Header.Get("X-Enode-Log-Bytes"), 10, 64); err == nil {
		ack.Total = n
	}
	if n, err := strconv.Atoi(resp.Header.Get("X-Enode-Log-Attempt")); err == nil {
		ack.Attempt = n
	}
	return ack, nil
}

// uploader 는 한 단계의 진행 청크를 민다.
//
// 수명이 단계 하나다 - 단계마다 새로 만든다. 링의 Reset 과 같은 자리이고,
// 그래서 리셋을 잊을 자리가 없다. Worker 에 하나 두고 (seq, attempt) 로
// 리셋하는 모양은 조건을 사람이 맞춰야 하고, 틀리면 앞 단계의 꼬리가 다음
// 단계의 파일에 붙는다.
//
// 오프셋이 이 구조체에 없는 것이 값이다. 보내는 고루틴의 지역 변수로만 살아서
// Write 가 그것을 볼 일이 0 이다 - 공유 상태가 하나 줄고, Write 가 못 막히는
// 것이 잠금 범위가 아니라 구조로 선다.
type uploader struct {
	cl   *Client
	step *Step
	log  *slog.Logger

	mu  sync.Mutex
	buf []byte
	// inflight 는 take 로 떼어 나가 보내는 중인 바이트 수다.
	//
	// 상한이 이것을 함께 세야 한다. 안 세면 보내는 동안 버퍼가 비어 보여서
	// 그 틈에 상한만큼이 또 쌓이고, 실제 메모리가 상한의 두 배가 된다.
	// 시험이 이것을 잡았다 - 넘칠 만큼 썼는데 안 넘쳤다.
	inflight int
	// done 이 서면 더 안 받고 더 안 보낸다. 서는 자리가 둘이다 - 서버가
	// 상한에 닿았다고 했을 때와, 버퍼가 넘쳤을 때.
	done bool
	// overflowed 는 버퍼가 넘쳐 멈춘 것이다. 상한과 갈라 적는 이유는
	// 중앙이 이 둘을 다르게 알기 때문이다 - 상한은 서버가 표시 줄을 박고,
	// 넘침은 아무 줄도 안 남는다 (못 닿아서 넘쳤으니 표시도 못 보낸다).
	overflowed bool

	wake    chan struct{}
	stop    chan struct{}
	stopped chan struct{}
}

func newUploader(cl *Client, step *Step, log *slog.Logger) *uploader {
	u := &uploader{
		cl: cl, step: step, log: log,
		wake:    make(chan struct{}, 1),
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go u.run()
	return u
}

// Write 는 버퍼에 넣고 바로 돌아온다. 네트워크를 0 번 기다린다.
//
// 언제나 len(p), nil 을 낸다. io.MultiWriter 는 한 갈래가 오류를 내면 거기서
// 멈추므로, 여기서 오류를 내면 링까지 끊기고 제어판의 카드도 함께 죽는다.
func (u *uploader) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	u.mu.Lock()
	if u.done {
		u.mu.Unlock()
		return len(p), nil
	}
	if len(u.buf)+u.inflight+len(p) > uploadBufferMax {
		// 넘쳤다. 오래된 것을 버리고 계속 올리는 길도 있으나 그러면 중앙의
		// 파일이 이어 붙은 것처럼 보이는데 가운데가 빠진다 - 읽는 사람도
		// 파서도 그것을 구별할 길이 0 이다. 올라간 데까지를 온전하게 둔다.
		//
		// 연결이 돌아와도 재개하지 않는다. 넘쳐서 안 받은 바이트가 이미
		// 없으므로 재개하면 구멍이 생기고, 그 구멍에 표시가 없다.
		u.done, u.overflowed = true, true
		u.mu.Unlock()
		u.logf("the transcript upload buffer is full; no more of this step will reach the mediator",
			"limit", uploadBufferMax, "step", u.step.StepID)
		return len(p), nil
	}
	u.buf = append(u.buf, p...)
	big := len(u.buf) >= uploadNow
	u.mu.Unlock()
	if big {
		select {
		case u.wake <- struct{}{}:
		default: // 이미 깨어 있다. 여기서 막히면 Write 가 막힌다
		}
	}
	return len(p), nil
}

// Close 는 마지막으로 비우고 고루틴을 멈춘다.
//
// error 를 안 낸다 - 부르는 쪽이 검사할 것이 없다. 실패는 전부 노드 로그로
// 가고 실행을 안 막는다. 언제나 nil 인 error 를 돌려주면 부르는 쪽이 없는
// 값을 검사한다.
func (u *uploader) Close() {
	close(u.stop)
	<-u.stopped
}

func (u *uploader) logf(msg string, args ...any) {
	if u.log != nil {
		u.log.Warn(msg, args...)
	}
}

// take 는 버퍼를 통째로 떼어 온다. 버퍼는 다시 빈다.
func (u *uploader) take() []byte {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.done || len(u.buf) == 0 {
		return nil
	}
	b := u.buf
	u.buf = nil
	u.inflight = len(b)
	return b
}

// settled 는 보낸 것이 서버에 붙었음을 셈에서 지운다.
func (u *uploader) settled() {
	u.mu.Lock()
	u.inflight = 0
	u.mu.Unlock()
}

// putBack 은 못 보낸 것을 버퍼 앞에 도로 붙인다. 다음 주기에 새 바이트와 함께
// 한 번에 간다 - 청크 경계가 달라져도 서버가 이어 붙이므로 상관없다.
func (u *uploader) putBack(b []byte) {
	u.mu.Lock()
	u.inflight = 0
	overflow := false
	if !u.done {
		if len(b)+len(u.buf) > uploadBufferMax {
			u.done, u.overflowed, overflow = true, true, true
			u.buf = nil
		} else {
			u.buf = append(b, u.buf...)
		}
	}
	u.mu.Unlock()
	// 로그를 잠금 밖에서 낸다. 안에서 내면 슬로그 핸들러가 쓰는 동안 Write 가
	// 막히고, 그것이 이 겹이 절대로 하면 안 되는 일이다.
	if overflow {
		u.logf("the transcript upload buffer is full while retrying; no more of this step will reach the mediator",
			"limit", uploadBufferMax, "step", u.step.StepID)
	}
}

func (u *uploader) halt() {
	u.mu.Lock()
	u.done = true
	u.buf = nil
	u.inflight = 0
	u.mu.Unlock()
}

// run 은 보내는 고루틴이다.
//
// 패닉을 잡는 이유는 다른 것과 같다 - 고루틴 하나가 패닉으로 죽으면 프로세스가
// 죽는다. 관측하는 겹이 노드를 죽이면 그것은 관측이 아니다.
func (u *uploader) run() {
	defer close(u.stopped)
	defer func() {
		if r := recover(); r != nil {
			u.logf("the transcript uploader panicked and stopped", "err", r)
			u.halt()
		}
	}()

	// 오프셋은 여기 산다. 구조체 필드가 아니라 이 고루틴의 지역 변수다.
	//
	// 무엇에 쓰나 - 서버가 든 총 길이와 내 셈을 견준다. 둘이 갈리면 서버가
	// 내가 보낸 것과 다른 양을 받은 것이고(상한에서 일부만 받는 자리가 그렇다),
	// 그때 진실은 서버 쪽이다. 내 셈을 서버 값으로 맞추고 한 번 적는다.
	var sent int64
	diverged := false
	fails := 0
	var firstFail time.Time

	tick := time.NewTicker(uploadPeriod)
	defer tick.Stop()

	flush := func(ctx context.Context) {
		b := u.take()
		if len(b) == 0 {
			return
		}
		ack, err := u.cl.PutLogChunk(ctx, u.step.RunID, u.step.Seq, u.step.Name, u.step.Attempt, b)
		if err != nil {
			u.putBack(b)
			fails++
			if fails == 1 {
				// 연속 실패의 첫 번째만 적는다. 매번 적으면 Mediator 가
				// 10분 죽었을 때 2초마다 한 줄씩 300줄이 쌓이고, 그 300줄이
				// 노드 로그에서 진짜 실패를 덮는다.
				firstFail = time.Now()
				u.logf("cannot push the transcript chunk", "err", err, "step", u.step.StepID)
			}
			return
		}
		u.settled()
		if fails > 0 {
			u.logf("the transcript upload recovered",
				"failures", fails, "after", time.Since(firstFail).Round(time.Second),
				"step", u.step.StepID)
			fails = 0
		}
		// Attempt 를 Total 보다 먼저 본다. 버려진 청크의 Total 은 앞 시도의
		// 총 길이라, 그것으로 오프셋을 맞추면 뒤로 간다.
		if ack.Attempt > u.step.Attempt {
			u.logf("a newer attempt is running; this uploader is stale and stops",
				"mine", u.step.Attempt, "server", ack.Attempt, "step", u.step.StepID)
			u.halt()
			return
		}
		if ack.Capped {
			u.logf("the progress file hit its size limit; no more of this step will be pushed",
				"bytes", ack.Total, "step", u.step.StepID)
			u.halt()
			return
		}
		if want := sent + int64(len(b)); ack.Total != want && !diverged {
			diverged = true
			u.logf("the mediator holds a different length than this node counted",
				"server", ack.Total, "node", want, "step", u.step.StepID)
		}
		sent = ack.Total
	}

	for {
		select {
		case <-u.stop:
			// 단계가 끝났다. 남은 꼬리를 마지막으로 비운다 - 이것이
			// UploadLog(선별본)보다 먼저라야 봉인 직전의 마지막 문장이
			// 화면에 뜬다.
			ctx, cancel := context.WithTimeout(context.Background(), uploadPeriod)
			flush(ctx)
			cancel()
			return
		case <-tick.C:
			flush(context.Background())
		case <-u.wake:
			flush(context.Background())
		}
	}
}

// 컴파일러에게 이 타입이 io.Writer 임을 말한다 - tee 에 꽂히는 것이 이 겹의
// 유일한 쓰임이라, 그 계약이 깨지면 배선하는 자리가 아니라 여기서 빨개져야 한다.
var _ io.Writer = (*uploader)(nil)

// ── 단계 끝의 업로드 (FR-3 · N3 · business-rules.md 8절) ─────────────────
//
// 단계 로그(선별본)와 $OUT 의 산출물을 올린다. 둘 다 업로드 client 를 쓰고 마감은
// 업로드 예산의 ctx 가 준다 — 요청마다의 30초가 큰 산출물을 끊던 자리다.

// upload 는 업로드 client 다. 없으면 일반 것을 쓴다 — 없다고 동작이 달라지면 안 된다.
func (c *Client) upload() *http.Client {
	if c.Upload != nil {
		return c.Upload
	}
	return c.HTTP
}

// BlobRejected 는 Mediator 가 받고서 거절한 업로드다 (4xx). 전송 실패가 아니다 —
// 스키마 위반(422)이나 크기 상한(413)은 Mediator 의 판정이고, upload 칸을 error 로
// 만들지 않는다. 다시 보내도 같은 답이다.
type BlobRejected struct{ Status, Body string }

func (e *BlobRejected) Error() string {
	if e.Body == "" {
		return e.Status
	}
	return e.Status + ": " + e.Body
}

// UploadLog 는 그 단계가 뱉은 것을 원문 그대로 올린다 (ADR-005 의 logs/).
// 산출물보다 먼저 올린다 — 단계가 실패해도 로그는 남아야 한다.
func (c *Client) UploadLog(ctx context.Context, runID string, seq int, name string, body []byte) error {
	url := fmt.Sprintf("%s/v1/runs/%s/steps/%d/log?name=%s", c.Base, runID, seq, name)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	return c.put(req)
}

// PutBlob 은 산출물을 Mediator 로 흘려 보낸다 (run-contract §4 별 모양).
//
// 통째로 읽지 않는다 — size 를 Content-Length 로 싣고 body 를 그대로 흘린다. 크기를
// 미리 재서 거절하지 않는다: Mediator 가 상한+1 바이트에서 413 으로 끊으므로 보내는
// 양이 상한을 안 넘고, 노드는 Mediator 의 상한 값을 모른다.
//
// 노드끼리 직접 전송하지 않는다 — 서로 다른 기계라 공유 작업공간이 없고,
// 직접 보내려면 enode 가 서로를 알아야 한다. Mediator 는 이미 전부와 말한다.
//
// 422 는 스키마 위반이다 (ADR-020) — 저장되지 않았으므로 그 이름을
// produced 에 넣으면 안 된다. 어긴 산출물은 산출물이 아니다.
func (c *Client) PutBlob(ctx context.Context, runID string, seq int, name string, body io.Reader, size int64) error {
	if size == 0 {
		body = http.NoBody // 0 과 본문이 함께면 크기를 모르는 것으로 친다
	}
	url := fmt.Sprintf("%s/v1/runs/%s/steps/%d/blob/%s", c.Base, runID, seq, name)
	req, err := http.NewRequestWithContext(ctx, "PUT", url, body)
	if err != nil {
		return err
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", "application/octet-stream")
	return c.put(req)
}

func (c *Client) put(req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-Enode-Principal", c.Principal)
	resp, err := c.upload().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode/100 == 2 {
		return nil
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if resp.StatusCode/100 == 4 {
		return &BlobRejected{Status: resp.Status, Body: strings.TrimSpace(string(b))}
	}
	return fmt.Errorf("upload failed: %s %s", resp.Status, strings.TrimSpace(string(b)))
}

// upload 는 단계 로그를 먼저, 그다음 $OUT 의 이름들을 올린다 (business-rules.md 8절).
//
// 올라간 것만 produced 다 — 거절된 것은 저장되지 않았고, 어긴 산출물은 산출물이
// 아니다 (ADR-020). blobs 가 거짓이면 산출물을 안 올린다 (완주하지 못한 agent 단계).
//
// stage 는 upload 칸의 값이다. 마감이 지나면 남은 이름을 안 올리고 timeout,
// 예산 안에서 전송이 한 건이라도 실패했으면 error, 거절은 영향이 없다.
func (w *Worker) upload(ctx context.Context, step *Step, out string, logBody []byte, blobs bool, log *slog.Logger) ([]string, contract.Stage) {
	stage := contract.StageOK
	// failed 는 한 건의 실패를 적고 멈출지를 돌려준다 — 마감이나 임대 끝이면 멈춘다.
	failed := func(kind, name string, err error) bool {
		var rej *BlobRejected
		switch {
		case errors.As(err, &rej):
			log.Warn(kind+" rejected; not listed in produced", "name", name, "err", err)
		case ctx.Err() != nil:
			return true
		default:
			log.Warn(kind+" upload failed", "name", name, "err", err)
			stage = contract.StageError
		}
		return false
	}
	if err := w.Client.UploadLog(ctx, step.RunID, step.Seq, step.Name, logBody); err != nil && failed("log", step.Name, err) {
		return nil, w.uploadStopped(ctx, append([]string{"(step log)"}, uploadNames(out, blobs)...), log)
	}
	if !blobs {
		return nil, stage
	}
	var produced []string
	names := uploadNames(out, true)
	for i, name := range names {
		if err := w.putFile(ctx, step, out, name); err != nil {
			if failed("blob", name, err) {
				return produced, w.uploadStopped(ctx, names[i:], log)
			}
			continue
		}
		produced = append(produced, name)
	}
	return produced, stage
}

// uploadStopped 는 마감이나 임대 끝으로 멈춘 업로드다. 못 올린 이름을 적는다.
func (w *Worker) uploadStopped(ctx context.Context, left []string, log *slog.Logger) contract.Stage {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		log.Warn("upload budget exceeded; not uploaded", "names", left)
		return contract.StageTimeout
	}
	log.Warn("upload stopped; not uploaded", "names", left)
	return contract.StageError
}

// uploadNames 는 $OUT 에서 올릴 이름이다 — 디렉터리를 읽은 순서(이름순).
// .enode- 로 시작하는 것은 어댑터가 남긴 것(프롬프트 등)이고, .part 로 끝나는 것은
// 마감에 멈춘 복사가 남긴 반쪽이다. 둘 다 산출물이 아니다.
func uploadNames(out string, blobs bool) []string {
	if !blobs {
		return nil
	}
	var names []string
	for _, name := range harvest(out) {
		if strings.HasPrefix(name, ".enode-") || strings.HasSuffix(name, ".part") {
			continue
		}
		names = append(names, name)
	}
	return names
}

// putFile 은 파일 하나를 열어 흘려 보낸다.
func (w *Worker) putFile(ctx context.Context, step *Step, out, name string) error {
	f, err := os.Open(filepath.Join(out, name))
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck
	fi, err := f.Stat()
	if err != nil {
		return err
	}
	return w.Client.PutBlob(ctx, step.RunID, step.Seq, name, f, fi.Size())
}
