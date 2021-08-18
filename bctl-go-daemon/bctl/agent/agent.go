package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	controlws "bastionzero.com/bctl/v1/Server/Websockets/controlWebsocket"
	controlwsmsg "bastionzero.com/bctl/v1/Server/Websockets/controlWebsocket/controlWebsocketTypes"
	dc "bastionzero.com/bctl/v1/bctl/agent/datachannel"
	wsmsg "bastionzero.com/bctl/v1/bzerolib/channels/message"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

var (
	serviceUrl, orgId, clusterName string
	environmentId, activationToken string
)

const (
	hubEndpoint = "/api/v1/hub/kube-server"
	token       = "1234" // TODO: figure this out

	// Disable auto-reconnect
	autoReconnect = false
)

func main() {
	parseFlags()

	// Get agent version
	version := getAgentVersion()

	// Connect to the control channel
	control := controlws.NewControlWebsocketClient(serviceUrl, activationToken, orgId, clusterName, environmentId, version)

	// Subscribe to control channel
	go func() {
		for {
			select {
			case message := <-control.ProvisionWebsocketChan:
				// We have an incoming websocket request, attempt to make a new Daemon Websocket Client for the request
				startDatachannel(message)
			}
		}
	}()

	// Sleep forever
	// Ref: https://stackoverflow.com/questions/36419054/go-projects-main-goroutine-sleep-forever
	select {} // I don't think we need this?
}

func startDatachannel(message controlwsmsg.ProvisionNewWebsocketMessage) {
	// Create our headers and params, headers are empty
	// TODO: We need to drop this session id auth header req and move to a token based system
	headers := make(map[string]string)

	// Add our token to our params
	params := make(map[string]string)
	params["daemon_connection_id"] = message.ConnectionId
	params["token"] = token

	// Create our response channels
	// TODO: WE NEED TO SEND AN INTERRUPT CHANNEL TO DATACHANNEL FROM CONTROL
	// or pass a context that we can cancel from the control channel??
	dc.NewDataChannel(message.Role, serviceUrl, hubEndpoint, params, headers, targetSelectHandler, autoReconnect)
}

func targetSelectHandler(agentMessage wsmsg.AgentMessage) (string, error) {
	// First check if its a keysplitting message
	var keysplittingPayload map[string]interface{}
	if err := json.Unmarshal(agentMessage.MessagePayload, &keysplittingPayload); err == nil {
		if keysplittingPayloadVal, ok := keysplittingPayload["keysplittingPayload"].(map[string]interface{}); ok {
			switch keysplittingPayloadVal["action"] {
			case "kube/restapi":
				return "ResponseToBastionFromCluster", nil
			case "kube/exec/start":
				return "ResponseToBastionFromCluster", nil
			case "kube/exec/input":
				return "ResponseToBastionFromCluster", nil
			case "kube/exec/resize":
				return "ResponseToBastionFromCluster", nil
			}
		}
	}

	// Else check if its a stream message
	var messagePayload smsg.StreamMessage
	if err := json.Unmarshal(agentMessage.MessagePayload, &messagePayload); err == nil {
		// p := payload["keysplittingPayload"].(map[string]interface{})
		switch messagePayload.Type {
		case "kube/exec/stdout":
			return "StdoutToBastionFromCluster", nil
		case "kube/exec/stderr":
			return "StderrToBastionFromCluster", nil
		case "kube/log":
			return "ResponseLogToBastionFromCluster", nil
		}
	}

	return "", fmt.Errorf("unable to determine SignalR endpoint")
}

func parseFlags() {
	// Our expected flags we need to start
	flag.StringVar(&serviceUrl, "serviceUrl", "", "Service URL to use")
	flag.StringVar(&orgId, "orgId", "", "OrgId to use")
	flag.StringVar(&clusterName, "clusterName", "", "Cluster name to use")
	flag.StringVar(&environmentId, "environmentId", "", "Optional environmentId to specify")
	flag.StringVar(&activationToken, "activationToken", "", "Activation Token to use to register the cluster")

	// Parse any flag
	flag.Parse()

	// The environment will overwrite any flags passed
	serviceUrl = os.Getenv("SERVICE_URL")
	activationToken = os.Getenv("ACTIVATION_TOKEN")
	orgId = os.Getenv("ORG_ID")
	clusterName = os.Getenv("CLUSTER_NAME")
	environmentId = os.Getenv("ENVIRONMENT")

	// Ensure we have all needed vars
	missing := []string{}
	switch {
	case serviceUrl == "":
		missing = append(missing, "serviceUrl")
		fallthrough
	case orgId == "":
		missing = append(missing, "orgId")
		fallthrough
	case clusterName == "":
		missing = append(missing, "clusterName")
		fallthrough
	// case environmentId == "":
	// 	missing = append(missing, "environmentId")
	case activationToken == "":
		missing = append(missing, "activationToken")
	}
	if len(missing) > 0 {
		log.Printf("Missing flags! Missing: %v", missing)
		os.Exit(1)
	}
}

func getAgentVersion() string {
	if os.Getenv("DEV") == "true" {
		return "1.0"
	} else {
		return "$AGENT_VERSION"
	}
}
