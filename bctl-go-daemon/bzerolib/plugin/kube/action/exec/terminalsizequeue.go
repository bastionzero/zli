package exec

import (
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
	"k8s.io/client-go/tools/remotecommand"
)

type TerminalSizeQueue struct {
	StreamType   smsg.StreamType
	InputChannel chan KubeExecResizeActionPayload // pretty sure this needs to be buffered
	RequestId    int
}

func NewTerminalSizeQueue(requestId int) *TerminalSizeQueue {
	return &TerminalSizeQueue{
		InputChannel: make(chan KubeExecResizeActionPayload),
		RequestId:    requestId,
	}
}

func (t *TerminalSizeQueue) Next() *remotecommand.TerminalSize {
	tsMessage := <-t.InputChannel

	return &remotecommand.TerminalSize{
		Width:  tsMessage.Width,
		Height: tsMessage.Height,
	}
}
