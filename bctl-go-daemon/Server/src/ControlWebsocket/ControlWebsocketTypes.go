package ControlWebsocket

type ProvisionNewWebsocketSignalRMessage struct {
	Target    string                         `json:"target"`
	Arguments []ProvisionNewWebsocketMessage `json:"arguments"`
	Type      int                            `json:"type"`
}

type ProvisionNewWebsocketMessage struct {
	ConnectionId string `json:"connectionId"`
}
