package jsonrpctest

import (
	"errors"
	"sync"
	"time"

	"github.com/eclipse/che/agents/go-agents/core/jsonrpc"
)

// RequestRecorder helps to catch/record jsonrpc.Channel incoming requests.
type ReqRecorder struct {
	mutex    *sync.Mutex
	cond     *sync.Cond
	closed   bool
	requests []*reqPair
}

// NewReqRecorder creates a new recorder.
func NewReqRecorder() *ReqRecorder {
	mx := &sync.Mutex{}
	return &ReqRecorder{
		mutex:    mx,
		cond:     sync.NewCond(mx),
		closed:   false,
		requests: make([]*reqPair, 0),
	}
}

// NativeConnWaitPredicate is used to wait on recorder until the condition
// behind this predicate is met.
type ReqRecorderWaitPredicate func(req *ReqRecorder) bool

// WriteCalledAtLeast is a predicate that allows to wait until write is called at
// least given number of times.
func ResponseArrivedAtLeast(times int) ReqRecorderWaitPredicate {
	return func(recorder *ReqRecorder) bool {
		return len(recorder.requests) >= times
	}
}

// WaitUntil waits until either recorder is closed or predicate returned true,
// if closed before predicate returned true, error is returned.
func (recorder *ReqRecorder) WaitUntil(p ReqRecorderWaitPredicate) error {
	recorder.cond.L.Lock()
	for {
		if recorder.closed {
			recorder.cond.L.Unlock()
			return errors.New("Closed before condition is met")
		}
		if p(recorder) {
			recorder.cond.L.Unlock()
			return nil
		}
		recorder.cond.Wait()
	}
	recorder.cond.L.Unlock()
	return nil
}

// CloseAfter closes this request after specified timeout.
func (recorder *ReqRecorder) CloseAfter(dur time.Duration) {
	go func() {
		<-time.NewTimer(dur).C
		recorder.Close()
	}()
}

// GetMethodHandler returns this recorder.
func (recorder *ReqRecorder) GetMethodHandler(method string) (jsonrpc.MethodHandler, bool) {
	return recorder, true
}

// GetMethodHandler returns given params.
func (recorder *ReqRecorder) Unmarshal(params []byte) (interface{}, error) {
	return params, nil
}

// Call records a call of method handler.
func (recorder *ReqRecorder) Call(params interface{}, rt jsonrpc.ResponseTransmitter) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	if byteParams, ok := params.([]byte); ok {
		recorder.requests = append(recorder.requests, &reqPair{
			request: &jsonrpc.Request{
				RawParams: byteParams,
			},
			transmitter: rt,
		})
		recorder.cond.Broadcast()
	}
}

// Get returns request + response transmitter which were caught (idx+1)th.
func (recorder *ReqRecorder) Get(idx int) (*jsonrpc.Request, jsonrpc.ResponseTransmitter) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	pair := recorder.requests[idx]
	return pair.request, pair.transmitter
}

// Close closes this recorder all waiters will be notified about close
// and WaitUntil will return errors for them.
func (recorder *ReqRecorder) Close() {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	if !recorder.closed {
		recorder.closed = true
		recorder.cond.Broadcast()
	}
}

type reqPair struct {
	request     *jsonrpc.Request
	transmitter jsonrpc.ResponseTransmitter
}
