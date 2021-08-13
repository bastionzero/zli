package logs

// For "kube/restapi" actions

type KubeLogsActionPayload struct {
	Endpoint  string            `json:"endpoint"`
	Headers   map[string]string `json:"headers"`
	Method    string            `json:"method"`
	Body      string            `json:"body"`
	RequestId string            `json:"requestIdentifier"`
	End       bool              `json:"end"`
	LogId     string            `json:"logId"`
}
