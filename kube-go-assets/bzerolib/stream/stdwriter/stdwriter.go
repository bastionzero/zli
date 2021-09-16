package stdwriter

import (
	"encoding/base64"

	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

type StdWriter struct {
	StdType        smsg.StreamType
	outputChannel  chan smsg.StreamMessage
	RequestId      string
	SequenceNumber int
	logId          string
}

// Stdout or Stderr
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
	str := base64.StdEncoding.EncodeToString(p)
	message := smsg.StreamMessage{
		Type:           string(w.StdType),
		RequestId:      w.RequestId,
		SequenceNumber: w.SequenceNumber,
		Content:        str,
		LogId:          w.logId,
	}
	w.outputChannel <- message
	w.SequenceNumber = w.SequenceNumber + 1

	return len(p), nil
}
