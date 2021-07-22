package daemonWebsocketTypes

type ReadyToClientFromBastionSignalRMessage struct {
	Target    string                            `json:"target"`
	Arguments []ReadyToClientFromBastionMessage `json:"arguments"`
	Type      int                               `json:"type"`
}

type ReadyToClientFromBastionMessage struct {
	Ready bool `json:"ready"`
}

type ResponseToDaemonFromBastionSignalRMessage struct {
	Type      int                                  `json:"type"`
	Target    string                               `json:"target"`
	Arguments []ResponseToDaemonFromBastionMessage `json:"arguments"`
}

type ResponseToDaemonFromBastionMessage struct {
	StatusCode        int               `json:"statusCode"`
	Content           []byte            `json:"content"`
	RequestIdentifier int               `json:"requestIdentifier"`
	Headers           map[string]string `json:"headers"`
}

type RequestToBastionFromDaemonSignalRMessage struct {
	Target    string                              `json:"target"`
	Arguments []RequestToBastionFromDaemonMessage `json:"arguments"`
	Type      int                                 `json:"type"`
}
type RequestToBastionFromDaemonMessage struct {
	LogId             string            `json:"logId"`
	KubeCommand       string            `json:"kubeCommand"`
	Endpoint          string            `json:"endpoint"`
	Headers           map[string]string `json:"Headers"`
	Method            string            `json:"Method"`
	Body              []byte            `json:"Body"`
	RequestIdentifier int               `json:"RequestIdentifier"`
}

type StartExecToBastionFromDaemonSignalRMessage struct {
	Target    string                                `json:"target"`
	Arguments []StartExecToBastionFromDaemonMessage `json:"arguments"`
	Type      int                                   `json:"type"`
}
type StartExecToBastionFromDaemonMessage struct {
	Command           []string `json:"command"`
	Endpoint          string   `json:"endpoint"`
	RequestIdentifier int      `json:"requestIdentifier"`
}

type StdinToBastionFromDaemonSignalRMessage struct {
	Target    string                            `json:"target"`
	Arguments []StdinToBastionFromDaemonMessage `json:"arguments"`
	Type      int                               `json:"type"`
}
type StdinToBastionFromDaemonMessage struct {
	Stdin             []byte `json:"stdin"`
	RequestIdentifier int    `json:"requestIdentifier"`
}

type StdoutToDaemonFromBastionSignalRMessage struct {
	Target    string                             `json:"target"`
	Arguments []StdoutToDaemonFromBastionMessage `json:"arguments"`
	Type      int                                `json:"type"`
}
type StdoutToDaemonFromBastionMessage struct {
	Stdout            []byte `json:"stdout"`
	RequestIdentifier int    `json:"requestIdentifier"`
}

type StderrToDaemonFromBastionSignalRMessage struct {
	Target    string                             `json:"target"`
	Arguments []StderrToDaemonFromBastionMessage `json:"arguments"`
	Type      int                                `json:"type"`
}
type StderrToDaemonFromBastionMessage struct {
	Stderr            []byte `json:"stderr"`
	RequestIdentifier int    `json:"requestIdentifier"`
}

type ResizeTerminalToBastionFromDaemonSignalRMessage struct {
	Target    string                                     `json:"target"`
	Arguments []ResizeTerminalToBastionFromDaemonMessage `json:"arguments"`
	Type      int                                        `json:"type"`
}
type ResizeTerminalToBastionFromDaemonMessage struct {
	Width             uint16 `json:"width"`
	Height            uint16 `json:"height"`
	RequestIdentifier int    `json:"requestIdentifier"`
}
