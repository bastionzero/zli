package message

import (
	"encoding/base64"
	"fmt"
	"time"

	"bastionzero.com/bctl/v1/bzerolib/keysplitting/hasher"
)

type DataAckPayload struct {
	Timestamp     string `json:"timestamp"` // Unix time int64
	SchemaVersion string `json:"schemaVersion"`
	Type          string `json:"type"`
	Action        string `json:"action"`

	//Unique to DataAck
	TargetPublicKey       string      `json:"targetPublicKey"`
	HPointer              string      `json:"hPointer"`
	ActionResponsePayload interface{} `json:"actionResponsePayload"`
}

func (d DataAckPayload) BuildResponsePayload(action string, actionPayload []byte) (DataPayload, error) {
	hashBytes, _ := hasher.HashPayload((d))
	hash := base64.StdEncoding.EncodeToString(hashBytes)

	return DataPayload{
		Timestamp:     fmt.Sprint(time.Now().Unix()),
		SchemaVersion: d.SchemaVersion,
		Type:          string(Data),
		Action:        d.Action,
		TargetId:      d.TargetPublicKey, //TODO: Make this come from storage
		HPointer:      hash,
		ActionPayload: actionPayload,
		BZCertHash:    "", // TODO: Make this come from storage
	}, nil
}
