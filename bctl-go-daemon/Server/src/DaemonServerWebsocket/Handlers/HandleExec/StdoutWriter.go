package HandleExec

import (
	"bastionzero.com/bctl/v1/Server/src/DaemonServerWebsocket/DaemonServerWebsocketTypes"
)

// Our customer stdout writer so we can pass it into StreamOptions
type StdoutWriter struct {
	ch                chan byte
	wsClient          *DaemonServerWebsocketTypes.DaemonServerWebsocket
	RequestIdentifier int
}

// Constructor
func NewStdoutWriter(wsClient *DaemonServerWebsocketTypes.DaemonServerWebsocket, requestIdentifier int) *StdoutWriter {
	return &StdoutWriter{
		ch:                make(chan byte, 1024),
		wsClient:          wsClient,
		RequestIdentifier: requestIdentifier,
	}
}

func (w *StdoutWriter) Chan() <-chan byte {
	return w.ch
}

// Our custom write function, this will send the data over the websocket
func (w *StdoutWriter) Write(p []byte) (int, error) {
	// Send this data over our websocket
	sendStdoutToBastionMessage := &DaemonServerWebsocketTypes.SendStdoutToBastionMessage{}
	sendStdoutToBastionMessage.RequestIdentifier = w.RequestIdentifier
	sendStdoutToBastionMessage.Stdout = string(p)
	w.wsClient.SendSendStdoutToBastionMessage(*sendStdoutToBastionMessage)

	// Calculate what needs to be returned
	n := 0
	for _, b := range p {
		w.ch <- b
		n++
	}
	return n, nil
}

// Close the writer by closing the channel
func (w *StdoutWriter) Close() error {
	close(w.ch)
	return nil
}
