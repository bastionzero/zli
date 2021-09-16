package exec

import (
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
	"k8s.io/client-go/tools/remotecommand"
)

type TerminalSizeQueue struct {
	StreamType        smsg.StreamType
	execResizeChannel chan KubeExecResizeActionPayload // pretty sure this needs to be buffered
	RequestId         string
}

func NewTerminalSizeQueue(requestId string, execResizeChannel chan KubeExecResizeActionPayload) *TerminalSizeQueue {
	return &TerminalSizeQueue{
		execResizeChannel: execResizeChannel,
		RequestId:         requestId,
	}
}

func (t *TerminalSizeQueue) Next() *remotecommand.TerminalSize {
	tsMessage := <-t.execResizeChannel

	return &remotecommand.TerminalSize{
		Width:  tsMessage.Width,
		Height: tsMessage.Height,
	}
}
