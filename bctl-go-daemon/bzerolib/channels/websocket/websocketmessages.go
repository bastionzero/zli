package websocket

type GetChallengeMessage struct {
	OrgId       string `json:"orgId"`
	ClusterName string `json:"clusterName"`
}

type GetChallengeResponse struct {
	Challenge string `json:"challenge"`
}
