package exec

// Exec payload for the "kube/exec/start" action
type KubeExecStartActionPayload struct {
	RequestId int      `json:"requestId"`
	Command   []string `json:"command"` // what does this look like? Does it contain flags?
	Endpoint  string   `json:"endpoint"`
}

// Exec payload for the "kube/exec/input" action
type KubeStdinActionPayload struct {
	RequestId int    `json:"requestId"`
	Stdin     []byte `json:"stdin"`
	End       bool   `json:"end"`
}

// payload for "kube/exec/resize"
type KubeExecResizeActionPayload struct {
	RequestId int    `json:"requestId"`
	Width     uint16 `json:"width"`
	Height    uint16 `json:"height"`
}
