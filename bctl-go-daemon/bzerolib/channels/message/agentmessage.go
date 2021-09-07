/*
This package defines all of the messages that are used at the AgentMessage level.
It defines the different types of messages (MessageType) and correlated payload
structs: the 4 types of keysplitting messages and agent output streams.
*/
package message

const (
	SchemaVersion = "1.0"
)

type AgentMessage struct {
	MessageType    string `json:"messageType"`
	SchemaVersion  string `json:"schemaVersion"`
	MessagePayload []byte `json:"messagePayload"`
}

// The different categories of messages we might send/receive
type MessageType string

const (
	// All keysplittings messages: Syn, SynAck, Data, DataAck
	Keysplitting MessageType = "keysplitting"

	// Agent output stream message types
	Stream MessageType = "stream"

	// Error message type for reporting all error messages
	Error MessageType = "error"

	// For the control channel
	NewDatachannel MessageType = "newDatachannel" // TODO: Can we make this into a single word?
	HealthCheck    MessageType = "healthcheck"
)
