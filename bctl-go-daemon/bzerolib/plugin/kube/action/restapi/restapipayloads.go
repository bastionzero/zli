package restapi

// For "kube/restapi" actions

type KubeRestApiActionPayload struct {
	Endpoint  string            `json:"endpoint"`
	Headers   map[string]string `json:"headers"`
	Method    string            `json:"method"`
	Body      string            `json:"body"`
	RequestId int               `json:"requestIdentifier"`
	Role      string            `json:"role"`
}

type KubeRestApiActionResponsePayload struct {
	StatusCode int               `json:"statusCode"`
	RequestId  int               `json:"requestId"`
	Headers    map[string]string `json:"headers"`
	Content    []byte            `json:"content"`
}
