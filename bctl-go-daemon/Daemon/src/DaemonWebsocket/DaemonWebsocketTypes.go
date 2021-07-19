package DaemonWebsocket

type ReadyFromServerSignalRMessage struct {
	Target    string                   `json:"target"`
	Arguments []ReadyFromServerMessage `json:"arguments"`
	Type      int                      `json:"type"`
}

type ReadyFromServerMessage struct {
	Ready bool `json:"ready"`
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

type SendStdinToBastionSignalRMessage struct {
	Target    string                      `json:"target"`
	Arguments []SendStdinToBastionMessage `json:"arguments"`
	Type      int                         `json:"type"`
}
type SendStdinToBastionMessage struct {
	Stdin             string `json:"stdin"`
	RequestIdentifier int    `json:"requestIdentifier"`
}

type SendStdoutToDaemonFromBastionSignalRMessage struct {
	Target    string                                 `json:"target"`
	Arguments []SendStdoutToDaemonFromBastionMessage `json:"arguments"`
	Type      int                                    `json:"type"`
}
type SendStdoutToDaemonFromBastionMessage struct {
	Stdout            string `json:"stdout"`
	RequestIdentifier int    `json:"requestIdentifier"`
}
