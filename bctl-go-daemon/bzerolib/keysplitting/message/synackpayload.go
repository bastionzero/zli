package message

// Repetition in Keysplitting messages is requires to maintain flat
// structure which is important for hashing
type SynAckPayload struct {
	Timestamp     int64  `json:"timestamp"` // Unix time
	SchemaVersion string `json:"schemaVersion"`
	Type          string `json:"type"`
	Action        string `json:"action"`

	// Unique to SynAck
	TargetPublicKey string `json:"targetPublicKey"`
	Nonce           string `json:"nonce"`
	HPointer        string `json:"hPointer"`
}
