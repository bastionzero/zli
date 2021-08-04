/*
This package defines all of the messages that are used at the AgentMessage level.
It defines the different types of messages (MessageType) and correlated payload
structs: the 4 types of keysplitting messages and agent output streams.
*/
package message

const (
	Schema = "zero"
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

	// Meta control message types that do not have corresponding
	// payload definitions
	Ready MessageType = "ready"
	Stop  MessageType = "stop"
)
