// Package provides implementation of jsonrpc.NativeConn based on websocket.
//
// The example:
//
// Client:
//	conn, err := jsonrpcws.Dial("ws://host:port/path")
//	if err != nil {
//      	panic(err)
//      }
// 	channel := jsonrpc.NewChannel(conn, jsonrpc.DefaultRouter)
//	channel.Go()
//	channel.SayHello()
//
// Server:
//	conn, err := jsonrpcws.Upgrade(w, r)
//	if err != nil {
//      	panic(err)
//      }
//	channel := jsonrpc.NewChannel(conn, jsonrpc.DefaultRouter)
//	channel.Go()
//	channel.SayHello()
package jsonrpcws

import (
	"github.com/eclipse/che-lib/websocket"
	"github.com/eclipse/che/agents/go-agents/core/jsonrpc"
	"net/http"
)

var (
	defaultUpgrader = &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func NewDial(url string) (*NativeConnAdapter, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	return &NativeConnAdapter{conn: conn}, nil
}

func Upgrade(w http.ResponseWriter, r *http.Request) (*NativeConnAdapter, error) {
	conn, err := defaultUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &NativeConnAdapter{RequestURI: r.RequestURI, conn: conn}, nil
}

type NativeConnAdapter struct {
	RequestURI string
	conn       *websocket.Conn
}

// jsonrpc.NativeConn implementation
func (adapter *NativeConnAdapter) Write(data []byte) error {
	return adapter.conn.WriteMessage(websocket.TextMessage, data)
}

func (adapter *NativeConnAdapter) Next() ([]byte, error) {
	for {
		mType, data, err := adapter.conn.ReadMessage()
		if err != nil {
			if closeErr, ok := err.(*websocket.CloseError); ok {
				return nil, jsonrpc.NewCloseError(closeErr)
			}
			return nil, err
		}
		if mType == websocket.TextMessage {
			return data, nil
		}
	}
}

func (adapter *NativeConnAdapter) Close() error {
	err := adapter.conn.Close()
	if closeErr, ok := err.(*websocket.CloseError); ok && isNormallyClosed(closeErr.Code) {
		return nil
	} else {
		return err
	}
}

func isNormallyClosed(code int) bool {
	return code == websocket.CloseGoingAway ||
		code == websocket.CloseNormalClosure ||
		code == websocket.CloseNoStatusReceived
}
