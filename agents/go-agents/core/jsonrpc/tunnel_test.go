package jsonrpc_test

import (
	"bytes"
	"encoding/json"
	"github.com/eclipse/che/agents/go-agents/core/jsonrpc"
	"github.com/eclipse/che/agents/go-agents/core/jsonrpc/jsonrpctest"
	"testing"
	"time"
)

func TestChannelSaysHello(t *testing.T) {
	beforeConnected := time.Now()

	// initialization routine
	channel, connRecorder, reqRecorder := newTestChannel()
	channel.Go()
	defer channel.Close()
	defer reqRecorder.Close()

	// send hello notification
	channel.SayHello()

	// wait while this notification is received by connection
	if err := connRecorder.WaitUntil(jsonrpctest.WriteCalledAtLeast(1)); err != nil {
		t.Fatal(err)
	}

	// check the received notification is expected one
	helloNotification := &jsonrpc.TunnelNotification{}
	connRecorder.UnmarshalRequestParams(0, helloNotification)

	if helloNotification.ChannelID != channel.ID() {
		t.Fatalf("Tunnel ids are different %s != %s", helloNotification.ChannelID, channel.ID())
	}
	if helloNotification.Text != "Hello!" {
		t.Fatalf("Expected text to be 'Hello' but it is %s", helloNotification.Text)
	}
	now := time.Now()
	if !beforeConnected.Before(helloNotification.Time) || !helloNotification.Time.Before(now) {
		t.Fatalf("Expected event time to be between %v < x < %v", beforeConnected, now)
	}
}

// X Notification -> X'
func TestSendingNotification(t *testing.T) {
	channel, connRecorder, reqRecorder := newTestChannel()
	channel.Go()
	defer channel.Close()
	defer reqRecorder.Close()

	method := "event:my-event"
	channel.Notify(method, &testStruct{"Test"})

	if err := connRecorder.WaitUntil(jsonrpctest.WriteCalledAtLeast(1)); err != nil {
		t.Fatal(err)
	}

	// check request
	req, err := connRecorder.GetRequest(0)
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != method {
		t.Fatalf("Expected to send %s method but sent %s", method, req.Method)
	}
	if !req.IsNotification() {
		t.Fatalf("Expected request to be notification but it has id %v", req.ID)
	}

	// check params
	event := &testStruct{}
	json.Unmarshal(req.Params, event)
	if event.Data != "Test" {
		t.Fatal("Expected event data to be 'Test'")
	}
}

// X Request -> X'
func TestSendingRequest(t *testing.T) {
	channel, connRecorder, _ := newTestChannel()
	channel.Go()
	defer channel.Close()

	method := "domain.doSomething"
	channel.Request(method, &testStruct{"Test"}, func(response *jsonrpc.Response, err error) {
		// do nothing
	})

	if err := connRecorder.WaitUntil(jsonrpctest.WriteCalledAtLeast(1)); err != nil {
		t.Fatal(t)
	}

	// check request
	req, err := connRecorder.GetRequest(0)
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != method {
		t.Fatalf("Expected to send %s method but sent %s", method, req.Method)
	}
	if req.IsNotification() {
		t.Fatal("Expected request not to be notification but it does not have id")
	}

	// check params
	event := &testStruct{}
	json.Unmarshal(req.Params, event)
	if event.Data != "Test" {
		t.Fatal("Expected event data to be 'Test'")
	}
}

// X' Request -> X
func TestReceivingRequest(t *testing.T) {
	channel, connRecorder, reqRecorder := newTestChannel()
	channel.Go()
	defer channel.Close()
	defer reqRecorder.Close()

	// prepare a test request object and put it in native connection read stream
	reqBody, err := json.Marshal(testStruct{"Test"})
	if err != nil {
		t.Fatal(err)
	}
	sentReq := &jsonrpc.Request{
		ID:     "1",
		Method: "domain.doSomething",
		Params: reqBody,
	}
	connRecorder.PushNext(sentReq)

	// channel needs some time to call the handler
	if err := reqRecorder.WaitUntil(jsonrpctest.ResponseArrivedAtLeast(1)); err != nil {
		t.Fatal(err)
	}

	receivedReq, _ := reqRecorder.Get(0)
	if string(receivedReq.Params) != string(sentReq.Params) {
		t.Fatalf("Sent params %s but received %s", string(sentReq.Params), string(receivedReq.Params))
	}
}

// X' Request  -> X
// X' <- Response X
func TestSendingResponseBack(t *testing.T) {
	channel, connRecorder, reqRecorder := newTestChannel()
	channel.Go()
	defer channel.Close()
	defer reqRecorder.Close()

	// prepare a test request object and put it in native connection read stream
	reqBody, err := json.Marshal(testStruct{"Test"})
	if err != nil {
		t.Fatal(err)
	}
	req := &jsonrpc.Request{
		ID:     1,
		Method: "domain.doSomething",
		Params: reqBody,
	}
	connRecorder.PushNext(req)

	// wait for request to arrive
	if err := reqRecorder.WaitUntil(jsonrpctest.ResponseArrivedAtLeast(1)); err != nil {
		t.Fatal(t)
	}

	// respond back
	_, transmitter := reqRecorder.Get(0)
	sentBody := testStruct{"response test data"}
	transmitter.Send(sentBody)

	// wait for response to be written
	if err := connRecorder.WaitUntil(jsonrpctest.WriteCalledAtLeast(1)); err != nil {
		t.Fatal(err)
	}

	resp, err := connRecorder.GetResponse(0)
	if err != nil {
		t.Fatal(err)
	}

	// check the response is ok
	if resp.ID != req.ID {
		t.Fatalf("Expected ids to be the same but resp id %v != req id %v", resp.ID, req.ID)
	}
	if resp.Error != nil {
		t.Fatalf("Expected to get response without error, but got %d %s", resp.Error.Code, resp.Error.Message)
	}
	respBody := testStruct{}
	if err := json.Unmarshal(resp.Result, &respBody); err != nil {
		t.Fatal(err)
	}
	if respBody != sentBody {
		t.Fatalf("Expected to get the same body but got %v != %v", respBody, sentBody)
	}
}

// X Request  -> X'
// X <- Response X'
func TestRequestResponseHandling(t *testing.T) {
	channel, connRecorder, reqRecorder := newTestChannel()
	channel.Go()
	defer channel.Close()
	defer reqRecorder.Close()

	respChan := make(chan *jsonrpc.Response, 1)

	// X Request -> X'
	channel.Request("domain.doSomething", &testStruct{"req-params"}, func(response *jsonrpc.Response, err error) {
		respChan <- response
	})

	// wait for the response and catch its id
	if err := connRecorder.WaitUntil(jsonrpctest.WriteCalledAtLeast(1)); err != nil {
		t.Fatal(err)
	}
	req, err := connRecorder.GetRequest(0)
	if err != nil {
		t.Fatal(t)
	}

	// X' Response -> X
	repsBody := testStruct{"resp-body"}
	marshaledBody, err := json.Marshal(&repsBody)
	if err != nil {
		t.Fatal(err)
	}
	connRecorder.PushNext(&jsonrpc.Response{
		ID:     req.ID,
		Result: marshaledBody,
	})

	// wait for the response handler function to be called
	select {
	case resp := <-respChan:
		if bytes.Compare(resp.Result, marshaledBody) != 0 {
			t.Fatalf("Received different response body %s != %s", string(resp.Result), string(marshaledBody))
		}
	case <-time.After(time.Second * 2):
		t.Fatal("Didn't receieve the response in 2seconds")
	}
}

func TestSendingBrokenData(t *testing.T) {
	channel, connRecorder, reqRecorder := newTestChannel()
	channel.Go()
	defer channel.Close()
	defer reqRecorder.Close()

	connRecorder.PushNextRaw([]byte("{not-a-json}"))

	if err := connRecorder.WaitUntil(jsonrpctest.WriteCalledAtLeast(1)); err != nil {
		t.Fatal(err)
	}

	response, err := connRecorder.GetResponse(0)
	if err != nil {
		t.Fatal(err)
	}

	if response.ID != nil {
		t.Fatal("Response id must be nill")
	}
	if response.Version != jsonrpc.DefaultVersion {
		t.Fatalf("Exected response version to be %d but it is %d", jsonrpc.DefaultVersion, response.Version)
	}
	if response.Result != nil {
		t.Fatalf("Expected response result to be nil, but it is %v", string(response.Result))
	}
	if response.Error == nil {
		t.Fatal("Expected response to contain error")
	}
	if response.Error.Code != jsonrpc.ParseErrorCode {
		t.Fatalf("Expected error code to be %d but it is %d", jsonrpc.ParseErrorCode, response.Error.Code)
	}
}

type testStruct struct {
	Data string `json:"data"`
}

func newTestChannel() (*jsonrpc.Tunnel, *jsonrpctest.ConnRecorder, *jsonrpctest.ReqRecorder) {
	connRecorder := jsonrpctest.NewConnRecorder()
	reqRecorder := jsonrpctest.NewReqRecorder()
	channel := jsonrpc.NewTunnel(connRecorder, reqRecorder)
	connRecorder.CloseAfter(2 * time.Second)
	reqRecorder.CloseAfter(2 * time.Second)
	return channel, connRecorder, reqRecorder
}
