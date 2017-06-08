package jsonrpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultVersion                  = "2.0"
	ConnectedNotificationMethodName = "connected"

	DefaultAllowedResponseDelay  = time.Minute
	DefaultWatchQueuePeriod      = time.Minute
	DefaultMaxRequestHoldTimeout = time.Minute
)

var (
	prevChanID    uint64
	prevRequestID int64
)

// Channel is high level jsonrpc transport layer which
// uses native connection to access low level transport routines.
type Channel struct {

	// ID is the unique identifier of this channel.
	ID string

	// Created is the time when this channel was created.
	Created time.Time

	// Conn is a native connection on which this channel is based.
	Conn NativeConn

	jsonOutChan chan interface{}
	reqHandler  RequestHandler
	rq          *requestQ
	closer      *closer
}

// ResponseHandleFunc used to handle requests responses.
// The request is sent by one of the Request, RequestBare, RequestRaw methods.
// If the response doesn't arrive in time the handler func will be called
// with an error of type *TimeoutError.
// Note 'response.Error' has nothing to do with func err param.
type ResponseHandleFunc func(response *Response, err error)

// RequestHandler is a single handler for all the channel incoming requests.
// The goal of the handler is to dispatch each incoming request to the interested side
// e.g. jsonrpc.Router dispatches registered request to the routes.
type RequestHandler interface {

	// GetMethodHandler must return a handler for a given method and ok=true,
	// if there is no such handler func must return ok=false
	GetMethodHandler(method string) (MethodHandler, bool)
}

// MethodHandler handles a certain method.
// First raw request parameters are decoded using Unmarshal function
// and then if call returned no error handle is called.
type MethodHandler interface {

	// Unmarshal decodes request raw request parameters
	// e.g. parses json and returns instance of structured params.
	// If the handler does not need parameters nil, nil should be returned.
	// If error is not nil Call is not executed.
	Unmarshal(params []byte) (interface{}, error)

	// Call calls handler of this request.
	// If no Send method is called on response transmitter instance
	// timeout error reply will be sent, unless request is notification.
	Call(params interface{}, rt ResponseTransmitter)
}

// ResponseTransmitter provides interface which allows to respond to request.
// The implementation must guarantee that reply will be eventually sent.
// Functions Send & SendError MUST not be called both or twice on the same
// instance of transmitter.
// The request id MUST be included to the response.
type ResponseTransmitter interface {

	// Send sends jsonrpc response with a given result in body.
	// This function can be called only once on one transmitter instance.
	Send(result interface{})

	// SendError sends jsonrpc response with a given error in body.
	// This function can be called only once on one transmitter instance.
	SendError(err *Error)

	// Channel returns the channel this transmitter belongs to.
	Channel() Channel
}

// TimeoutError occurs when timeout is reached before normal handling completes.
type TimeoutError struct{ error }

func NewManagedChannel(conn NativeConn) *Channel {
	channel := NewChannel(conn, DefaultRouter)
	Save(*channel)
	return channel
}

// NewChannel creates a new channel.
// Use channel.Go() to start the channel.
func NewChannel(conn NativeConn, handler RequestHandler) *Channel {
	rq := &requestQ{
		pairs:                make(map[int64]*rqPair),
		allowedResponseDelay: DefaultAllowedResponseDelay,
		stop:                 make(chan bool, 1),
		ticker:               time.NewTicker(DefaultWatchQueuePeriod),
	}
	channel := &Channel{
		ID:          "channel-" + strconv.Itoa(int(atomic.AddUint64(&prevChanID, 1))),
		Created:     time.Now(),
		Conn:        conn,
		jsonOutChan: make(chan interface{}),
		reqHandler:  handler,
		rq:          rq,
	}
	channel.closer = &closer{channel: channel}
	return channel
}

// Go starts this channel, makes it functional.
func (c *Channel) Go() {
	go c.mainWriteLoop()
	go c.mainReadLoop()
	go c.rq.watch()
}

// Notify sends notification(request without id) using given params as its body.
func (c *Channel) Notify(method string, params interface{}) {
	if marshaledParams, err := json.Marshal(params); err != nil {
		log.Printf("Could not unmarshal non-nil notification params, it won't be send. Error %s", err.Error())
	} else {
		c.jsonOutChan <- &Request{
			Version:   DefaultVersion,
			Method:    method,
			RawParams: marshaledParams,
		}
	}
}

// NotifyBare sends notification like Notify does but
// sends no request parameters in it.
func (c *Channel) NotifyBare(method string) {
	c.jsonOutChan <- &Request{Version: DefaultVersion, Method: method}
}

// Request sends request marshalling a given params as its body.
// ResponseHandleFunc will be called as soon as the response arrives,
// or response arrival timeout reached, in that case error of type
// TimeoutError will be passed to the handler.
func (c *Channel) Request(method string, params interface{}, rhf ResponseHandleFunc) {
	if marshaledParams, err := json.Marshal(params); err != nil {
		log.Printf("Could not unmrashall non-nil request params, it won't be send. Error %s", err.Error())
	} else {
		id := atomic.AddInt64(&prevRequestID, 1)
		request := &Request{
			ID:        id,
			Method:    method,
			RawParams: marshaledParams,
		}
		c.rq.add(id, request, time.Now(), rhf)
		c.jsonOutChan <- request
	}
}

// RequestBare sends the request like Request func does
// but sends no params in it.
func (c *Channel) RequestBare(method string, rhf ResponseHandleFunc) {
	id := atomic.AddInt64(&prevRequestID, 1)
	request := &Request{ID: id, Method: method}
	c.rq.add(id, request, time.Now(), rhf)
	c.jsonOutChan <- request
}

// Close closes native connection and internal sources, so started
// go routines should be eventually stopped.
func (c *Channel) Close() {
	c.closer.closeOnce()
}

// SayHello sends hello notification.
func (c *Channel) SayHello() {
	c.Notify(ConnectedNotificationMethodName, &ChannelEvent{
		Time:      c.Created,
		ChannelID: c.ID,
		Text:      "Hello!",
	})
}

// ChannelEvent struct describing notification params sent by SayHello.
type ChannelEvent struct {
	Time      time.Time `json:"time"`
	ChannelID string    `json:"channel"`
	Text      string    `json:"text"`
}

func (c *Channel) mainWriteLoop() {
	for message := range c.jsonOutChan {
		if bytes, err := json.Marshal(message); err != nil {
			log.Printf("Couldn't marshal message: %T, %v to json. Error %s", message, message, err.Error())
		} else {
			if err := c.Conn.Write(bytes); err != nil {
				log.Printf("Couldn't write message to the channel. Message: %T, %v", message, message)
			}
		}
	}
}

func (c *Channel) mainReadLoop() {
	for {
		binMessage, err := c.Conn.Next()
		if err == nil {
			c.dispatchMessage(binMessage)
		} else {
			c.closer.closeOnce()
			return
		}
	}
}

func (c *Channel) dispatchResponse(r *Response) {
	if r.ID == nil {
		log.Print("Received response with empty identifier, response will be ignored")
		return
	}

	// float64 used for json numbers https://blog.golang.org/json-and-go
	floatID, ok := r.ID.(float64)
	if !ok {
		log.Printf("Received response with non-numeric identifier %T %v, "+
			"response will be ignored", r.ID, r.ID)
		return
	}

	id := int64(floatID)
	rqPair, ok := c.rq.remove(id)
	if ok {
		rqPair.respHandlerFunc(r, nil)
	} else {
		log.Printf("Response handler for request id '%v' is missing which means that response "+
			"arrived to late, or response provides a wrong id", id)
	}
}

func (c *Channel) dispatchMessage(binMessage []byte) {
	draft := &draft{}
	err := json.Unmarshal(binMessage, draft)

	// parse error indicated
	if err != nil {
		c.jsonOutChan <- &Response{
			Version: DefaultVersion,
			ID:      nil,
			Error:   NewError(ParseErrorCode, errors.New("Error while parsing request")),
		}
		return
	}

	// version check
	if draft.Version == "" {
		draft.Version = DefaultVersion
	} else if draft.Version != DefaultVersion {
		err := fmt.Errorf("Version %s is not supported, please use %s", draft.Version, DefaultVersion)
		c.jsonOutChan <- &Response{
			Version: DefaultVersion,
			ID:      nil,
			Error:   NewError(InvalidRequestErrorCode, err),
		}
		return
	}

	// dispatching
	if draft.Method == "" {
		c.dispatchResponse(&Response{
			Version: draft.Version,
			ID:      draft.ID,
			Result:  draft.Result,
			Error:   draft.Error,
		})
	} else {
		c.dispatchRequest(&Request{
			Version:   draft.Version,
			Method:    draft.Method,
			ID:        draft.ID,
			RawParams: draft.RawParams,
		})
	}
}

func (c *Channel) dispatchRequest(r *Request) {
	handler, ok := c.reqHandler.GetMethodHandler(r.Method)

	if !ok {
		if !r.IsNotification() {
			c.jsonOutChan <- &Response{
				ID:      r.ID,
				Version: DefaultVersion,
				Error:   NewErrorf(MethodNotFoundErrorCode, "No such method %s", r.Method),
			}
		}
		return
	}

	// handle params decoding
	decodedParams, err := handler.Unmarshal(r.RawParams)
	if err != nil {
		if !r.IsNotification() {
			c.jsonOutChan <- &Response{
				ID:      r.ID,
				Version: DefaultVersion,
				Error:   NewError(ParseErrorCode, errors.New("Couldn't parse params")),
			}
		}
		return
	}

	// pick the right transmitter
	var transmitter ResponseTransmitter
	if r.IsNotification() {
		transmitter = &doNothingTransmitter{r}
	} else {
		transmitter = newWatchingRespTransmitter(r.ID, c)
	}

	// make a method call
	handler.Call(decodedParams, transmitter)
}

// both request and response
type draft struct {
	Version   string          `json:"jsonrpc"`
	Method    string          `json:"method"`
	ID        interface{}     `json:"id"`
	RawParams json.RawMessage `json:"params"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *Error          `json:"error,omitempty"`
}

type closer struct {
	once    sync.Once
	channel *Channel
}

func (closer *closer) closeOnce() {
	closer.once.Do(func() {
		close(closer.channel.jsonOutChan)
		closer.channel.rq.stopWatching()
		if err := closer.channel.Conn.Close(); err != nil {
			log.Printf("Error while closing connection, %s", err.Error())
		}
	})
}

type rqPair struct {
	request         *Request
	respHandlerFunc ResponseHandleFunc
	saved           time.Time
}

// Request queue is used for internal request + response handler storage,
// which allows to handle future responses.
// The q does not support any request identifiers except int64.
type requestQ struct {
	sync.RWMutex
	pairs                map[int64]*rqPair
	ticker               *time.Ticker
	stop                 chan bool
	allowedResponseDelay time.Duration
}

func (q *requestQ) watch() {
	for {
		select {
		case <-q.ticker.C:
			q.dropOutdatedOnce()
		case <-q.stop:
			q.ticker.Stop()
			return
		}
	}
}

func (q *requestQ) dropOutdatedOnce() {
	dropTime := time.Now().Add(-q.allowedResponseDelay)
	dropout := make([]int64, 0)

	q.RLock()
	for id, pair := range q.pairs {
		if pair.saved.Before(dropTime) {
			dropout = append(dropout, id)
		}
	}
	q.RUnlock()

	for _, id := range dropout {
		if rqPair, ok := q.remove(id); ok {
			rqPair.respHandlerFunc(nil, &TimeoutError{errors.New("Response didn't arrive in time")})
		}
	}
}

func (q *requestQ) stopWatching() { q.stop <- true }

func (q *requestQ) add(id int64, r *Request, time time.Time, rhf ResponseHandleFunc) {
	q.Lock()
	defer q.Unlock()
	q.pairs[id] = &rqPair{
		request:         r,
		respHandlerFunc: rhf,
		saved:           time,
	}
}

func (q *requestQ) remove(id int64) (*rqPair, bool) {
	q.Lock()
	defer q.Unlock()
	pair, ok := q.pairs[id]
	if ok {
		delete(q.pairs, id)
	}
	return pair, ok
}

func newWatchingRespTransmitter(id interface{}, c *Channel) *respTransmitter {
	t := &respTransmitter{
		reqId:   id,
		channel: c,
		once:    &sync.Once{},
		done:    make(chan bool, 1),
	}
	go t.watch(DefaultMaxRequestHoldTimeout)
	return t
}

type respTransmitter struct {
	reqId   interface{}
	channel *Channel
	once    *sync.Once
	done    chan bool
}

func (drt *respTransmitter) watch(timeout time.Duration) {
	timer := time.NewTimer(timeout)
	select {
	case <-timer.C:
		drt.SendError(NewErrorf(InternalErrorCode, "Server didn't respond to the request %v in time ", drt.reqId))
	case <-drt.done:
		timer.Stop()
	}
}

func (drt *respTransmitter) Send(result interface{}) {
	drt.release(func() {
		marshaled, _ := json.Marshal(result)
		drt.channel.jsonOutChan <- &Response{
			Version: DefaultVersion,
			ID:      drt.reqId,
			Result:  marshaled,
		}
	})
}

func (drt *respTransmitter) SendError(err *Error) {
	drt.release(func() {
		drt.channel.jsonOutChan <- &Response{
			Version: DefaultVersion,
			ID:      drt.reqId,
			Error:   err,
		}
	})
}

func (drt *respTransmitter) release(f func()) {
	drt.once.Do(func() {
		f()
		drt.done <- true
	})
}

func (drt *respTransmitter) Channel() Channel { return *drt.channel }

type doNothingTransmitter struct {
	request *Request
}

func (nrt *doNothingTransmitter) logNoResponse(res interface{}) {
	log.Printf(
		"The response to the notification '%s' will not be send(jsonrpc2.0 spec). The response was %T, %v",
		nrt.request.Method,
		res,
		res,
	)
}

func (nrt *doNothingTransmitter) Send(result interface{}) { nrt.logNoResponse(result) }

func (nrt *doNothingTransmitter) SendError(err *Error) { nrt.logNoResponse(err) }

func (nrt *doNothingTransmitter) Channel() Channel { return Channel{} }
