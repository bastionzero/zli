package restapi

// For "kube/restapi" actions

type KubeRestApiActionPayload struct {
	Endpoint        string              `json:"endpoint"`
	Headers         map[string][]string `json:"headers"`
	Method          string              `json:"method"`
	Body            string              `json:"body"`
	RequestId       string              `json:"requestId"`
	CommandBeingRun string              `json:"commandBeingRun"`
	LogId           string              `json:"logId"`
}

type KubeRestApiActionResponsePayload struct {
	StatusCode int                 `json:"statusCode"`
	RequestId  string              `json:"requestId"`
	Headers    map[string][]string `json:"headers"`
	Content    []byte              `json:"content"`
}
