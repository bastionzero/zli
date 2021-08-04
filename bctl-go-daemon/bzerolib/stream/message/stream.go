package message

// Agent Output Streaming Messages

type StreamMessage struct {
	Type           string `json:"type"` // either stdout or stderr, see "StreamType"
	RequestId      int    `json:"requestId"`
	SequenceNumber int    `json:"sequenceId"`
	Content        []byte `json:"content"`
}

// Type restriction on our different kinds of agent
// output streams.  StdIn will come in the form of a
// Keysplitting DataMessage
type StreamType string

const (
	StdErr StreamType = "stderr"
	StdOut StreamType = "stdout"
	StdIn  StreamType = "stdin"
)
