package handleExec

import (
	"bastionzero.com/bctl/v1/Server/Websockets/daemonServerWebsocket/daemonServerWebsocketTypes"
)

// Our custom stdin reader so we can pass it into Stream Options
type StdinReader struct {
	wsClient          *daemonServerWebsocketTypes.DaemonServerWebsocket
	RequestIdentifier int
}

func NewStdinReader(wsClient *daemonServerWebsocketTypes.DaemonServerWebsocket, requestIdentifier int) *StdinReader {
	return &StdinReader{
		wsClient:          wsClient,
		RequestIdentifier: requestIdentifier,
	}
}

func (r *StdinReader) Read(p []byte) (int, error) {
	// time.Sleep(time.Second * 2)
	// if r.readIndex >= int64(len(r.data)) {
	// 	err = io.EOF
	// 	return
	// }

	// n = copy(p, r.data[r.readIndex:])
	// r.readIndex += int64(n)
	// return

	// I think we will have to manually check for \n or exit, and then return err = io.EOF and n = 0

	// First set up our listening for the webscoket
	// go func() {

	// TODO: We need a special message to send our EOF
	stdinToClusterFromBastionSignalRMessage := daemonServerWebsocketTypes.StdinToClusterFromBastionSignalRMessage{}
	stdinToClusterFromBastionSignalRMessage = <-r.wsClient.ExecStdinChannel
	stdinToClusterFromBastionMessage := daemonServerWebsocketTypes.StdinToClusterFromBastionMessage{}
	stdinToClusterFromBastionMessage = stdinToClusterFromBastionSignalRMessage.Arguments[0]
	if stdinToClusterFromBastionMessage.RequestIdentifier == r.RequestIdentifier {
		n := copy(p, stdinToClusterFromBastionMessage.Stdin)
		return n, nil
	} else {
		// Rebroadcast the message
		r.wsClient.AlertOnExecStdinChan(stdinToClusterFromBastionSignalRMessage)
	}

	// Stdin is not over yet, but no bytes were read
	return 0, nil

	// }()
}
