package message

import (
	"bastionzero.com/bctl/v1/bzerolib/keysplitting/bzcert"
)

// Repetition in Keysplitting messages is requires to maintain flat
// structure which is important for hashing
type SynPayload struct {
	Timestamp     int64  `json:"timestamp"` // Unix time
	SchemaVersion string `json:"schemaVersion"`
	Type          string `json:"type"`
	Action        string `json:"action"`

	// Unique to Syn
	TargetId string        `json:"targetId"`
	Nonce    string        `json:"nonce"`
	BZCert   bzcert.BZCert `json:"BZCert"`
}
