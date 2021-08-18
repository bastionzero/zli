package controlchannel

type NewDatachannelMessage struct {
	ConnectionId string `json:"connectionId"`
	Role         string `json:"role"`
	Token        string `json:"token"`
}

type AliveCheckToBastionFromClusterMessage struct {
	Alive        bool     `json:"alive"`
	ClusterUsers []string `json:"clusterUsers"`
}
