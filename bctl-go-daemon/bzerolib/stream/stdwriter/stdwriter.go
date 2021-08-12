package stdwriter

import (
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

var (
	notreadybytes = [2]byte{64, 94}
)

type StdWriter struct {
	StdType        smsg.StreamType
	outputChannel  chan smsg.StreamMessage
	RequestId      int
	SequenceNumber int
	ready          bool
}

func NewStdWriter(streamType smsg.StreamType, ch chan smsg.StreamMessage, requestId int) *StdWriter {
	return &StdWriter{
		StdType:        streamType,
		outputChannel:  ch,
		RequestId:      requestId,
		SequenceNumber: 0,
		ready:          false,
	}
}

func (w *StdWriter) Write(p []byte) (int, error) {
	// TODO: Fix this bug, not sure why were are seeing so many of these random bytes, not ready bytes maybe?
	// if w.ready == false && p[0] != notreadybytes[0] && p[0] != notreadybytes[1] {
	// 	w.ready = true
	// }

	// if w.ready == true {
	message := smsg.StreamMessage{
		Type:           string(w.StdType),
		RequestId:      w.RequestId,
		SequenceNumber: w.SequenceNumber,
		Content:        p,
	}
	w.outputChannel <- message
	w.SequenceNumber = w.SequenceNumber + 1
	// }

	return len(p), nil
}
