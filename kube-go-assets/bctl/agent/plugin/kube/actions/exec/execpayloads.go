package exec

// Exec payload for the "kube/exec/start" action
type KubeExecStartActionPayload struct {
	RequestId       string   `json:"requestId"`
	LogId           string   `json:"logId"`
	IsTty           bool     `json:"isTty"`
	Command         []string `json:"command"`
	Endpoint        string   `json:"endpoint"`
	CommandBeingRun string   `json:"commandBeingRun"`
}

// Exec payload for the "kube/exec/input" action
type KubeStdinActionPayload struct {
	RequestId string `json:"requestId"`
	LogId     string `json:"logId"`
	Stdin     []byte `json:"stdin"`
}

// payload for "kube/exec/resize"
type KubeExecResizeActionPayload struct {
	RequestId string `json:"requestId"`
	LogId     string `json:"logId"`
	Width     uint16 `json:"width"`
	Height    uint16 `json:"height"`
}
