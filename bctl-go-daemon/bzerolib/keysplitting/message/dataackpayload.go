package message

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

func (d DataAckPayload) BuildReplyPayload(payload interface{}, pubKey string) (DataPayload, error) {
	return DataPayload{}, nil
}
