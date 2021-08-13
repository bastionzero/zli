package exec

// Exec payload for the "kube/exec/start" action
type KubeExecStartActionPayload struct {
	RequestId       string   `json:"requestId"`
	Command         []string `json:"command"` // what does this look like? Does it contain flags?
	Endpoint        string   `json:"endpoint"`
	LogId           string   `json"logId"`
	CommandBeingRun string   `json"commandBeingRun"`
}

// Exec payload for the "kube/exec/input" action
type KubeStdinActionPayload struct {
	RequestId string `json:"requestId"`
	Stdin     []byte `json:"stdin"`
	End       bool   `json:"end"`
	LogId     string `json:"logId"`
}

// payload for "kube/exec/resize"
type KubeExecResizeActionPayload struct {
	RequestId string `json:"requestId"`
	LogId     string `json:"logId:`
	Width     uint16 `json:"width"`
	Height    uint16 `json:"height"`
}
