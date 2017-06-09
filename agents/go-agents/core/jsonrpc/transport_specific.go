package jsonrpc

// NativeConn provides low level interface for jsonrpc.Tunnel
// to communicate with native connection such as websocket.
type NativeConn interface {

	// Write writes bytes to the connection.
	Write(body []byte) error

	// Next is blocking read of incoming messages.
	// If connection is closed an error of type *jsonrpc.CloseError
	// must be returned.
	Next() ([]byte, error)

	// Closes this connection.
	// After it is closed calls to Write and Next must fail with
	// *jsonrpc.CloseError error.
	Close() error
}

// CloseError is an error which MUST be
// published by NativeConn implementations and used to determine
// the cases when tunnel job should be stopped.
type CloseError struct{ error }

// NewCloseError creates a new close error based on a given error.
func NewCloseError(err error) *CloseError {
	return &CloseError{error: err}
}
