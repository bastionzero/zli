package stdreader

import (
	"bytes"
	"io"

	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

var (
	EndStreamBytes = []byte{0x62, 0x61, 0x73, 0x74, 0x69, 0x6f, 0x6e, 0x7a, 0x65, 0x72, 0x6f}
)

// Stdin
type StdReader struct {
	StreamType   smsg.StreamType
	RequestId    string
	stdinChannel chan []byte
}

func NewStdReader(streamType smsg.StreamType, requestId string, stdinChannel chan []byte) *StdReader {
	stdin := &StdReader{
		StreamType:   streamType,
		RequestId:    requestId,
		stdinChannel: stdinChannel,
	}

	return stdin
}

func (r *StdReader) Close() {
	r.Read(EndStreamBytes)
}

func (r *StdReader) Read(p []byte) (int, error) {
	// Listen for data on our stdinChannel
	if bytes.Equal(p, EndStreamBytes) {
		return 1, io.EOF
	}
	stdin := <-r.stdinChannel
	n := copy(p, stdin)
	return n, nil
}
