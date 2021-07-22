package HandleExec

import (
	"bastionzero.com/bctl/v1/Server/src/DaemonServerWebsocket/DaemonServerWebsocketTypes"
)

// Our customer stdout writer so we can pass it into StreamOptions
type StdoutWriter struct {
	wsClient          *DaemonServerWebsocketTypes.DaemonServerWebsocket
	RequestIdentifier int
}

// Constructor
func NewStdoutWriter(wsClient *DaemonServerWebsocketTypes.DaemonServerWebsocket, requestIdentifier int) *StdoutWriter {
	return &StdoutWriter{
		wsClient:          wsClient,
		RequestIdentifier: requestIdentifier,
	}
}

// Our custom write function, this will send the data over the websocket
func (w *StdoutWriter) Write(p []byte) (int, error) {
	// Send this data over our websocket
	stdoutToBastionFromClusterMessage := &DaemonServerWebsocketTypes.StdoutToBastionFromClusterMessage{}
	stdoutToBastionFromClusterMessage.RequestIdentifier = w.RequestIdentifier
	stdoutToBastionFromClusterMessage.Stdout = p
	w.wsClient.SendStdoutToBastionFromClusterMessage(*stdoutToBastionFromClusterMessage)

	// Calculate what needs to be returned
	return len(p), nil
}

// Close the writer by closing the channel
func (w *StdoutWriter) Close() error {
	return nil
}
