package controlWebsocketTypes

type ProvisionNewWebsocketSignalRMessage struct {
	Target    string                         `json:"target"`
	Arguments []ProvisionNewWebsocketMessage `json:"arguments"`
	Type      int                            `json:"type"`
}

type ProvisionNewWebsocketMessage struct {
	ConnectionId string `json:"connectionId"`
}

type AliveCheckToClusterFromBastionSignalRMessage struct {
	Target    string                                  `json:"target"`
	Arguments []AliveCheckToClusterFromBastionMessage `json:"arguments"`
	Type      int                                     `json:"type"`
}
type AliveCheckToClusterFromBastionMessage struct {
}

type AliveCheckToBastionFromClusterSignalRMessage struct {
	Target    string                                  `json:"target"`
	Arguments []AliveCheckToBastionFromClusterMessage `json:"arguments"`
	Type      int                                     `json:"type"`
}
type AliveCheckToBastionFromClusterMessage struct {
	Alive bool `json:"alive"`
}
