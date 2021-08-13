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
	RequestId      string
	SequenceNumber int
	logId          string
}

func NewStdWriter(streamType smsg.StreamType, ch chan smsg.StreamMessage, requestId string, logId string) *StdWriter {
	return &StdWriter{
		StdType:        streamType,
		outputChannel:  ch,
		RequestId:      requestId,
		SequenceNumber: 0,
		logId:          logId,
	}
}

func (w *StdWriter) Write(p []byte) (int, error) {
	message := smsg.StreamMessage{
		Type:           string(w.StdType),
		RequestId:      w.RequestId,
		SequenceNumber: w.SequenceNumber,
		Content:        p,
		LogId:          w.logId,
	}
	w.outputChannel <- message
	w.SequenceNumber = w.SequenceNumber + 1

	return len(p), nil
}
