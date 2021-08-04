package stdreader

import (
	"io"

	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

type StdReader struct {
	StreamType   smsg.StreamType
	InputChannel chan []byte // alternatively, we can push the stream message and double check that the requestId matches
	RequestId    int
}

func NewStdReader(streamType smsg.StreamType, requestId int) *StdReader {
	stdin := &StdReader{
		StreamType:   streamType,
		InputChannel: make(chan []byte),
		RequestId:    requestId,
	}

	// the idea here is to create a trigger on the channel such that whenever the action
	// pushes to the channel, it will push to read
	go func() {
		for {
			select {
			case messageBytes := <-stdin.InputChannel:
				stdin.Read(messageBytes)
			}
		}
	}()

	return stdin
}

func (r *StdReader) Read(p []byte) (int, error) {
	if len(p) > 0 {
		return len(p), nil // no fucking clue
	} else {
		return 0, io.EOF
	}
}
