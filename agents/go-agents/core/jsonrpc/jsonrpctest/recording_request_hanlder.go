package jsonrpctest

import (
	"errors"
	"github.com/eclipse/che/agents/go-agents/core/jsonrpc"
	"sync"
)

type RequestsRecorder struct {
	mutex    *sync.Mutex
	cond     *sync.Cond
	closed   bool
	requests []*reqPair
}

// NativeConnWaitPredicate is used to wait on recorder until the condition
// behind this predicate is met.
type ReqRecorderWaitPredicate func(req *RequestsRecorder) bool

// WriteCalledAtLeast is a predicate that allows to wait until write is called at
// least given number of times.
func ResponseArrivedAtLeast(times int) ReqRecorderWaitPredicate {
	return func(recorder *RequestsRecorder) bool {
		return len(recorder.requests) >= times
	}
}

type reqPair struct {
	request     *jsonrpc.Request
	transmitter jsonrpc.ResponseTransmitter
}

func NewReqRecorder() *RequestsRecorder {
	mx := &sync.Mutex{}
	return &RequestsRecorder{
		mutex:    mx,
		cond:     sync.NewCond(mx),
		closed:   false,
		requests: make([]*reqPair, 0),
	}
}

func (recorder *RequestsRecorder) GetMethodHandler(method string) (jsonrpc.MethodHandler, bool) {
	return recorder, true
}

func (recorder *RequestsRecorder) Unmarshal(params []byte) (interface{}, error) {
	return params, nil
}

func (recorder *RequestsRecorder) Call(params interface{}, rt jsonrpc.ResponseTransmitter) {
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

func (recorder *RequestsRecorder) WaitUntil(p ReqRecorderWaitPredicate) error {
	recorder.cond.L.Lock()
	for {
		if recorder.closed {
			recorder.cond.L.Unlock()
			return errors.New("01")
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

func (recorder *RequestsRecorder) Get(idx int) (*jsonrpc.Request, jsonrpc.ResponseTransmitter) {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	pair := recorder.requests[idx]
	return pair.request, pair.transmitter
}

func (recorder *RequestsRecorder) Close() {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	recorder.closed = true
	recorder.cond.Broadcast()
}
