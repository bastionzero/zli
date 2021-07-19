package DaemonServerWebsocket

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

type SendStdinToClusterSignalRMessage struct {
	Target    string                      `json:"target"`
	Arguments []SendStdinToClusterMessage `json:"arguments"`
	Type      int                         `json:"type"`
}
type SendStdinToClusterMessage struct {
	Stdin             string `json:"stdin"`
	RequestIdentifier int    `json:"requestIdentifier"`
}
