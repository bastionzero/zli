package DaemonServerWebsocket

// Our custom stdin reader so we can pass it into Stream Options
type StdinReader struct {
	wsClient          *DaemonServerWebsocket
	RequestIdentifier int
}

func NewStdinReader(wsClient *DaemonServerWebsocket, requestIdentifier int) *StdinReader {
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
	sendStdinToClusterSignalRMessage := SendStdinToClusterSignalRMessage{}
	sendStdinToClusterSignalRMessage = <-r.wsClient.ExecStdinChannel

	n := copy(p, []byte(sendStdinToClusterSignalRMessage.Arguments[0].Stdin))

	return n, nil

	// }()
}
