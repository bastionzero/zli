package websocketClientTypes

// SignalR Protocol
type SignalrNegotiateResponse struct {
	negotiateVersion int
	connectionId     string
}

// SignalR Hubs
type DataFromClientSignalRMessage struct {
	Target    string                  `json:"target"`
	Arguments []DataFromClientMessage `json:"arguments"`
	Type      int                     `json:"type"`
}
type DataFromClientMessage struct {
	LogId             string            `json:"logId"`
	KubeCommand       string            `json:"kubeCommand"`
	Endpoint          string            `json:"endpoint"`
	Headers           map[string]string `json:"Headers"`
	Method            string            `json:"Method"`
	Body              string            `json:"Body"`
	RequestIdentifier int               `json:"RequestIdentifier"`
}

type DataToClientSignalRMessage struct {
	Type      int                   `json:"type"`
	Target    string                `json:"target"`
	Arguments []DataToClientMessage `json:"arguments"`
}

type DataToClientMessage struct {
	StatusCode        int               `json:"statusCode"`
	Content           string            `json:"content"`
	RequestIdentifier int               `json:"requestIdentifier"`
	Headers           map[string]string `json:"headers"`
}
