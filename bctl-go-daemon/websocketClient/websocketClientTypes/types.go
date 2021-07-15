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

type ReadyFromServerSignalRMessage struct {
	Target    string                   `json:"target"`
	Arguments []ReadyFromServerMessage `json:"arguments"`
	Type      int                      `json:"type"`
}

type ReadyFromServerMessage struct {
	Ready bool `json:"ready"`
}

type RequestForServerSignalRMessage struct {
	Target    string                    `json:"target"`
	Arguments []RequestForServerMessage `json:"arguments"`
	Type      int                       `json:"type"`
}
type RequestForServerMessage struct {
	Endpoint          string            `json:"endpoint"`
	Headers           map[string]string `json:"headers"`
	Method            string            `json:"method"`
	Body              string            `json:"body"`
	RequestIdentifier int               `json:"requestIdentifier"`
}

type ResponseToDaemonSignalRMessage struct {
	Target    string                    `json:"target"`
	Arguments []ResponseToDaemonMessage `json:"arguments"`
	Type      int                       `json:"type"`
}
type ResponseToDaemonMessage struct {
	StatusCode        int               `json:"statusCode"`
	Content           string            `json:"content"`
	RequestIdentifier int               `json:"requestIdentifier"`
	Headers           map[string]string `json:"headers"`
}

type StartExecToBastionSignalRMessage struct {
	Target    string                      `json:"target"`
	Arguments []StartExecToBastionMessage `json:"arguments"`
	Type      int                         `json:"type"`
}
type StartExecToBastionMessage struct {
	Command           []string `json:"command"`
	Endpoint          string   `json:"endpoint"`
	RequestIdentifier int      `json:"requestIdentifier"`
}

type RequestForStartExecToClusterSingalRMessage struct {
	Target    string                                `json:"target"`
	Arguments []RequestForStartExecToClusterMessage `json:"arguments"`
	Type      int                                   `json:"type"`
}
type RequestForStartExecToClusterMessage struct {
	Command           []string `json:"command"`
	Endpoint          string   `json:"endpoint"`
	RequestIdentifier int      `json:"requestIdentifier"`
}

type SendStdoutToBastionSignalRMessage struct {
	Target    string                       `json:"target"`
	Arguments []SendStdoutToBastionMessage `json:"arguments"`
	Type      int                          `json:"type"`
}
type SendStdoutToBastionMessage struct {
	Stdout            string `json:"stdout"`
	RequestIdentifier int    `json:"requestIdentifier"`
}

type SendStdoutToDaemonSignalRMessage struct {
	Target    string                      `json:"target"`
	Arguments []SendStdoutToDaemonMessage `json:"arguments"`
	Type      int                         `json:"type"`
}
type SendStdoutToDaemonMessage struct {
	Stdout            string `json:"stdout"`
	RequestIdentifier int    `json:"requestIdentifier"`
}
