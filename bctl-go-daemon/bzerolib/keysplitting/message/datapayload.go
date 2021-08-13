package message

import (
	"encoding/base64"
	"fmt"
	"time"

	"bastionzero.com/bctl/v1/bzerolib/keysplitting/util"
)

type DataPayload struct {
	Timestamp     string `json:"timestamp"` // Unix time
	SchemaVersion string `json:"schemaVersion"`
	Type          string `json:"type"`
	Action        string `json:"action"`

	// Unique to Data Payload
	TargetId      string `json:"targetId"`
	HPointer      string `json:"hPointer"`
	BZCertHash    string `json:"bZCertHash"`
	ActionPayload []byte `json:"actionPayload"`
}

func (d DataPayload) BuildResponsePayload(actionPayload []byte, pubKey string) (DataAckPayload, string, error) {
	hashBytes, _ := util.HashPayload(d)
	hash := base64.StdEncoding.EncodeToString(hashBytes)

	return DataAckPayload{
		Timestamp:             fmt.Sprint(time.Now().Unix()),
		SchemaVersion:         d.SchemaVersion,
		Type:                  string(DataAck),
		Action:                d.Action,
		TargetPublicKey:       pubKey,
		HPointer:              hash,
		ActionResponsePayload: actionPayload,
	}, hash, nil
}
