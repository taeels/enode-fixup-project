package enode

import (
	"bytes"
	"sync"

	"github.com/taeels/enode/internal/transcript"
)

// 하네스 stdout 의 줄을 사건으로 흘린다 (FR-1).
//
// 이 파일이 있는 이유는 한 문장이다 — Decode 는 종료코드를 인자로 받으므로
// 프로세스가 끝나기 전에 못 돈다. 그런데 사람이 보려는 것은 단계가 끝나기
// 전의 문장이다. 흘리는 것과 판정하는 것이 다른 일이라 자리도 둘이다.
//
//	흘린다    lineEmitter   도는 중.  줄마다 종류를 낸다
//	판정한다   Decode        끝난 뒤.  봉투를 읽어 HarnessResult 를 낸다
//
// 그래서 runner 는 Decode 에 no-op 을 넘긴다. 둘 다 j.Emit 을 받으면 한 단계의
// 사건이 정확히 두 번 나고, 그 두 벌은 오늘 로그 두 줄로만 보이지만 이 자리에
// 업로더가 붙는 날 같은 바이트를 두 번 올린다.
const (
	// emitQueueDepth 는 줄 대기열의 깊이다.
	//
	// 크기로 막는 값이 아니다 — 차면 버리는 것이 설계다. claude 의
	// stream-json 이 한 턴에 내는 줄이 실측 수십이라, 읽는 쪽이 한 턴
	// 뒤처져도 안 차는 정도의 여유로 잡았다.
	emitQueueDepth = 256

	// maxEmitLineBytes 는 한 줄을 모으는 상한이다.
	//
	// 없으면 개행이 안 오는 동안 이 겹이 원문을 통째로 또 든다. stdout 버퍼가
	// 이미 전체를 들고 있으므로(봉투 파서가 뒤에서부터 전체를 훑는다) 여기까지
	// 들면 같은 바이트가 세 벌이다. 관측하는 겹이 메모리를 그렇게 쓰면 안 된다.
	//
	// 넘으면 그 줄을 버리고 다음 개행까지 건너뛴다. 버린 것은 노드 로그의
	// 사건 하나이고 링과 봉투는 안 다친다.
	maxEmitLineBytes = 1 << 20
)

// lineEmitter 는 tee 의 한 갈래다. 쓰기 경로에서는 줄만 가르고, 사건으로 읽는
// 것은 자기 고루틴이 한다.
//
// 가르는 이유가 지연이다 — transcript.Parse 는 줄마다 객체를 짓는다. 그것을
// 하네스의 쓰기 경로에서 하면 도구 결과 한 줄(수십 KB)마다 파싱이 끼어들고,
// 그 시간이 자식 프로세스의 stdout 에 되먹힌다. 관측이 실행을 느리게 하는
// 자리는 만들지 않는다.
type lineEmitter struct {
	mu      sync.Mutex
	tail    []byte // 개행을 아직 못 본 꼬리
	skip    bool   // 상한을 넘은 줄을 버리는 중이다
	dropped int64

	lines  chan []byte
	done   chan struct{}
	closed bool
}

func newLineEmitter(emit func(Event)) *lineEmitter {
	if emit == nil {
		emit = func(Event) {}
	}
	e := &lineEmitter{
		lines: make(chan []byte, emitQueueDepth),
		done:  make(chan struct{}),
	}
	go e.run(emit)
	return e
}

// run 은 줄을 사건으로 읽어 흘린다.
//
// emit 은 이 패키지의 코드가 아니다 — claim.go 가 넘긴 클로저다. 그것이
// 패닉을 내도 이 고루틴이 죽으면 안 된다: 고루틴 하나가 죽는 것은 프로세스가
// 죽는 것이고, 관측하는 겹이 노드를 죽이면 그것은 관측이 아니다.
func (e *lineEmitter) run(emit func(Event)) {
	defer close(e.done)
	for line := range e.lines {
		for _, ev := range transcript.Parse(line, false).Events {
			safeEmit(emit, Event{Kind: ev.Kind})
		}
	}
}

func safeEmit(emit func(Event), ev Event) {
	defer func() { _ = recover() }()
	emit(ev)
}

// Write 는 개행마다 줄을 잘라 대기열에 민다.
//
// 언제나 (len(p), nil) 이다. 오류를 내면 io.MultiWriter 가 거기서 멈추고,
// 멈추면 하네스의 stdout 이 막힌다 — 링이 같은 이유로 같은 규율을 쓴다
// (transcript.go 의 Ring.Write).
func (e *lineEmitter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.closed {
		return len(p), nil
	}
	rest := p
	for {
		i := bytes.IndexByte(rest, '\n')
		if i < 0 {
			e.appendTail(rest)
			return len(p), nil
		}
		e.appendTail(rest[:i])
		e.flushLine()
		rest = rest[i+1:]
	}
}

// appendTail 은 꼬리에 잇되 상한을 넘으면 그 줄을 버린다.
func (e *lineEmitter) appendTail(b []byte) {
	if e.skip {
		return
	}
	if len(e.tail)+len(b) > maxEmitLineBytes {
		e.tail = e.tail[:0]
		e.skip = true
		e.dropped++
		return
	}
	e.tail = append(e.tail, b...)
}

// flushLine 은 모인 줄을 대기열에 민다. 차면 버리고 센다.
//
// 대기열이 유계이고 버리는 것이 설계다 — 막으면 그 순간 실행이 막힌다.
func (e *lineEmitter) flushLine() {
	defer func() {
		e.tail = e.tail[:0]
		e.skip = false
	}()
	if e.skip || len(e.tail) == 0 {
		return
	}
	// 개행을 붙여서 넘긴다 — transcript.Parse 는 개행으로 안 끝나는 꼬리를
	// 안 읽고 Partial 에 센다 (parse.go 의 2번). 그 규율은 쓰는 중에 읽히는
	// 링과 진행 파일을 위한 것이고, 여기 오는 줄은 이미 개행을 본 완결된
	// 줄이다. 안 붙이면 사건이 0 이다.
	line := make([]byte, 0, len(e.tail)+1)
	line = append(line, e.tail...)
	line = append(line, '\n')
	select {
	case e.lines <- line:
	default:
		e.dropped++
	}
}

// Close 는 남은 꼬리를 마지막 줄로 밀고 대기열을 닫고 고루틴을 기다린다.
// 버린 줄 수를 낸다 — 부르는 쪽이 로그로 적는다.
//
// error 를 안 낸다. 이 겹에는 실패가 없다(R6 · R7)고 정해 두고 언제나 nil 인
// error 를 돌려주면, 부르는 쪽이 검사할 것이 없는 값을 검사하게 된다.
func (e *lineEmitter) Close() int64 {
	e.mu.Lock()
	if e.closed {
		n := e.dropped
		e.mu.Unlock()
		return n
	}
	e.flushLine()
	e.closed = true
	close(e.lines)
	e.mu.Unlock()

	<-e.done

	e.mu.Lock()
	defer e.mu.Unlock()
	return e.dropped
}
