package jsonrpctest

import (
	"encoding/json"
	"errors"
	"github.com/eclipse/che/agents/go-agents/core/jsonrpc"
	"sync"
)

// ConnRecorder is a fake connection which records or writes
// and provides functionality to push reads.
type ConnRecorder struct {
	mutex          *sync.Mutex
	cond           *sync.Cond
	capturedWrites [][]byte
	nextChan       chan []byte
	closed         bool
}

// NewConnRecorder returns a new conn recorder.
func NewConnRecorder() *ConnRecorder {
	mx := &sync.Mutex{}
	return &ConnRecorder{
		mutex:          mx,
		cond:           sync.NewCond(mx),
		capturedWrites: make([][]byte, 0),
		nextChan:       make(chan []byte),
	}
}

// NativeConnWaitPredicate is used to wait on recorder until the condition
// behind this predicate is met.
type NativeConnWaitPredicate func(recorder *ConnRecorder) bool

// WriteCalledAtLeast is a predicate that allows to wait until write is called at
// least given number of times.
func WriteCalledAtLeast(times int) NativeConnWaitPredicate {
	return func(recorder *ConnRecorder) bool {
		return len(recorder.capturedWrites) >= times
	}
}

// ReqSent is a predicate that waits until a given method is requested.
func ReqSent(method string) NativeConnWaitPredicate {
	return func(recorder *ConnRecorder) bool {
		wLen := len(recorder.capturedWrites)
		if wLen == 0 {
			return false
		}
		last := recorder.capturedWrites[wLen-1]
		req := &jsonrpc.Request{}
		if err := json.Unmarshal(last, req); err != nil {
			return false
		} else {
			return method == req.Method
		}
	}
}

func (recorder *ConnRecorder) GetCaptured(idx int) []byte {
	return recorder.capturedWrites[idx]
}

func (recorder *ConnRecorder) GetAllCaptured() [][]byte {
	recorder.mutex.Lock()
	defer recorder.mutex.Unlock()
	cp := make([][]byte, len(recorder.capturedWrites))
	copy(cp, recorder.capturedWrites)
	return cp
}

func (recorder *ConnRecorder) Unmarshal(idx int, v interface{}) error {
	return json.Unmarshal(recorder.GetCaptured(idx), v)
}

func (recorder *ConnRecorder) GetRequest(idx int) (*jsonrpc.Request, error) {
	req := &jsonrpc.Request{}
	err := recorder.Unmarshal(idx, req)
	return req, err
}

func (recorder *ConnRecorder) GetAllRequest() {

}

func (recorder *ConnRecorder) GetResponse(idx int) (*jsonrpc.Response, error) {
	resp := &jsonrpc.Response{}
	err := recorder.Unmarshal(idx, resp)
	if floatId, ok := resp.ID.(float64); ok {
		resp.ID = int(floatId)
	}
	return resp, err
}

func (recorder *ConnRecorder) UnmarshalRequestParams(idx int, v interface{}) error {
	req, err := recorder.GetRequest(idx)
	if err != nil {
		return err
	}
	err = json.Unmarshal(req.RawParams, &v)
	if err != nil {
		return err
	}
	return nil
}

func (recorder *ConnRecorder) WaitUntil(p NativeConnWaitPredicate) error {
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

func (recorder *ConnRecorder) PushNext(v interface{}) error {
	marshaled, err := json.Marshal(v)
	if err != nil {
		return err
	}
	recorder.PushNextRaw(marshaled)
	return nil
}

func (recorder *ConnRecorder) PushNextRaw(data []byte) {
	recorder.nextChan <- data
}

func (recorder *ConnRecorder) PushNextReq(method string, params interface{}) error {
	marshaledParams, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return recorder.PushNext(&jsonrpc.Request{
		ID:        "test",
		Method:    method,
		RawParams: marshaledParams,
	})
}

func (recorder *ConnRecorder) Write(body []byte) error {
	recorder.mutex.Lock()
	recorder.capturedWrites = append(recorder.capturedWrites, body)
	recorder.cond.Broadcast()
	recorder.mutex.Unlock()
	return nil
}

func (recorder *ConnRecorder) Next() ([]byte, error) {
	data, ok := <-recorder.nextChan
	if !ok {
		return nil, jsonrpc.NewCloseError(errors.New("Closed"))
	}
	return data, nil
}

func (recorder *ConnRecorder) Close() error {
	recorder.mutex.Lock()
	recorder.closed = true
	recorder.cond.Broadcast()
	recorder.mutex.Unlock()
	close(recorder.nextChan)
	return nil
}
