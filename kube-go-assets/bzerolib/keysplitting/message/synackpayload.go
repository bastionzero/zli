package message

import (
	"encoding/base64"
	"fmt"
	"time"

	"bastionzero.com/bctl/v1/bzerolib/keysplitting/util"
)

// Repetition in Keysplitting messages is requires to maintain flat
// structure which is important for hashing
type SynAckPayload struct {
	Timestamp             string `json:"timestamp"` // Unix time
	SchemaVersion         string `json:"schemaVersion"`
	Type                  string `json:"type"`
	Action                string `json:"action"`
	ActionResponsePayload []byte `json:"actionResponsePayload"`

	// Unique to SynAck
	TargetPublicKey string `json:"targetPublicKey"`
	Nonce           string `json:"nonce"`
	HPointer        string `json:"hPointer"`
}

func (s SynAckPayload) BuildResponsePayload(action string, actionPayload []byte, bzCertHash string) (DataPayload, string, error) {
	hashBytes, _ := util.HashPayload(s)
	hash := base64.StdEncoding.EncodeToString(hashBytes)

	return DataPayload{
		Timestamp:     fmt.Sprint(time.Now().Unix()),
		SchemaVersion: s.SchemaVersion,
		Type:          string(Data),
		Action:        action,
		TargetId:      s.TargetPublicKey, // TODO: Make this come from storage
		HPointer:      hash,
		ActionPayload: actionPayload,
		BZCertHash:    bzCertHash,
	}, hash, nil
}
