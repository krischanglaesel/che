package jsonrpc

import (
	"encoding/json"
	"log"
	"sync"
)

// DefaultRouter is a default package router.
// It might be used as channels request handler.
var DefaultRouter = NewRouter()

// RegRoute registers route using package DefaultRouter.
func RegRoute(r Route) { DefaultRouter.Register(r) }

// RegRoutesGroup registers routes group using package DefaultRouter.
func RegRoutesGroup(rg RoutesGroup) { DefaultRouter.RegisterGroup(rg) }

// RegRoutesGroups registers routes groups using package DefaultRouter.
func RegRoutesGroups(rgs []RoutesGroup) { DefaultRouter.RegisterGroups(rgs) }

// DecodeFunc used to decode route params for forth handling by HandleFunc.
type DecodeFunc func(body []byte) (interface{}, error)

// HandleFunc used to handle route request.
type HandleFunc func(params interface{}, t ResponseTransmitter)

// Route defines named operation and its handler.
type Route struct {

	// Method is the operation name like defined by Request.Method.
	Method string

	// Decode used for decoding raw request parameters
	// into the certain object. If decoding is okay, then
	// decoded value will be passed to the Handle
	// of this request route, so it is up to the route
	// - to define type safe couple of Decode & Handle.
	Decode DecodeFunc

	// Handle handler for decoded request parameters.
	// If handler function can't perform the operation then
	// handler function should either return an error, or
	// send it directly within transmitter#SendError func.
	// Params is a value returned from the Decode.
	Handle HandleFunc
}

// HandleNotification converts a function which receives parameters only
// to Route.Handle function.
func HandleNotification(f func(params interface{})) HandleFunc {
	return func(params interface{}, t ResponseTransmitter) { f(params) }
}

// HandleBareNotification converts a function which receives nothing
// to Route.Handle function.
func HandleBareNotification(f func()) HandleFunc {
	return func(params interface{}, t ResponseTransmitter) { f() }
}

// HandleBareRequest converts a function which receives nothing but transmitter
// to Route.Handle function.
func HandleBareRequest(f func(t ResponseTransmitter)) HandleFunc {
	return func(params interface{}, t ResponseTransmitter) { f(t) }
}

// FactoryDec uses given function to get an instance of object
// and then unmarshal params json into that object.
// The result of this function can be used as Route.Decode function.
func FactoryDec(create func() interface{}) func(body []byte) (interface{}, error) {
	return func(body []byte) (interface{}, error) {
		v := create()
		if err := json.Unmarshal(body, &v); err != nil {
			return nil, err
		}
		return v, nil
	}
}

func RetErrorHandle(f func(params interface{}, rt ResponseTransmitter) error) HandleFunc {
	return func(params interface{}, t ResponseTransmitter) {
		if err := f(params, t); err != nil {
			t.SendError(asJSONRPCErr(err))
		}
	}
}

func RetHandle(f func(params interface{}) (interface{}, error)) HandleFunc {
	return func(params interface{}, t ResponseTransmitter) {
		if res, err := f(params); err == nil {
			t.Send(res)
		} else {
			t.SendError(asJSONRPCErr(err))
		}
	}
}

func asJSONRPCErr(err error) *Error {
	if rpcerr, ok := err.(*Error); ok {
		return rpcerr
	} else {
		return NewError(InternalErrorCode, err)
	}
}

func (r Route) Unmarshal(params []byte) (interface{}, error) {
	if r.Decode == nil {
		return nil, nil
	} else {
		return r.Decode(params)
	}
}

func (r Route) Call(params interface{}, rt ResponseTransmitter) { r.Handle(params, rt) }

// RoutesGroup is named group of rpc routes
type RoutesGroup struct {
	// The name of this group e.g.: 'ProcessRPCRoutes'
	Name string

	// Rpc routes of this group
	Items []Route
}

type Router struct {
	mutex  sync.RWMutex
	routes map[string]Route
}

func NewRouter() *Router {
	return &Router{routes: make(map[string]Route)}
}

func (r *Router) Register(route Route) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.routes[route.Method] = route
}

func (r *Router) RegisterGroup(group RoutesGroup) {
	for _, route := range group.Items {
		r.Register(route)
	}
}

func (r *Router) RegisterGroups(groups []RoutesGroup) {
	for _, group := range groups {
		r.RegisterGroup(group)
	}
}

func (r *Router) GetMethodHandler(method string) (MethodHandler, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	route, ok := r.routes[method]
	return route, ok
}

// PrintRoutes prints provided rpc routes by group.
func PrintRoutes(rg []RoutesGroup) {
	log.Print("⇩ Registered RPCRoutes:\n\n")
	for _, group := range rg {
		log.Printf("%s:\n", group.Name)
		for _, route := range group.Items {
			log.Printf("✓ %s\n", route.Method)
		}
	}
}
