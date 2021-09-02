package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	cc "bastionzero.com/bctl/v1/bctl/agent/controlchannel"
	dc "bastionzero.com/bctl/v1/bctl/agent/datachannel"
	wsmsg "bastionzero.com/bctl/v1/bzerolib/channels/message"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
)

var (
	serviceUrl, orgId, clusterName string
	environmentId, activationToken string
)

const (
	hubEndpoint = "/api/v1/hub/kube-server"

	// Disable auto-reconnect
	autoReconnect = false
	logPath       = "/var/log/cwc"
)

func main() {
	// Get agent version
	version := getAgentVersion()

	// setup our loggers
	logger, err := lggr.NewLogger(lggr.Debug, logPath)
	if err != nil {
		return
	}
	logger.AddAgentVersion(version)

	ccLogger := logger.GetControlchannelLogger()
	dcLogger := logger.GetDatachannelLogger()

	if err := parseFlags(); err != nil {
		logger.Error(err)
		os.Exit(1)
	}

	// Connect to the control channel
	control, err := cc.NewControlChannel(ccLogger, serviceUrl, activationToken, orgId, clusterName, environmentId, version, controlchannelTargetSelectHandler)
	if err != nil {
		select {} // TODO: Should we be trying again here?
	}

	// Subscribe to control channel
	go func() {
		for {
			select {
			case message := <-control.NewDatachannelChan:
				// We have an incoming websocket request, attempt to make a new Daemon Websocket Client for the request
				startDatachannel(dcLogger, message)
			}
		}
	}()

	// Sleep forever because otherwise kube will endlessly try restarting
	// Ref: https://stackoverflow.com/questions/36419054/go-projects-main-goroutine-sleep-forever
	select {}
}

func startDatachannel(logger *lggr.Logger, message cc.NewDatachannelMessage) {
	// Create our headers and params, headers are empty
	// TODO: We need to drop this session id auth header req and move to a token based system
	headers := make(map[string]string)

	// Add our token to our params
	params := make(map[string]string)
	params["daemon_connection_id"] = message.ConnectionId
	params["token"] = message.Token

	// Create our response channels
	// TODO: WE NEED TO SEND AN INTERRUPT CHANNEL TO DATACHANNEL FROM CONTROL
	// or pass a context that we can cancel from the control channel??
	dc.NewDataChannel(logger, message.Role, serviceUrl, hubEndpoint, params, headers, datachannelTargetSelectHandler, autoReconnect)
}

func controlchannelTargetSelectHandler(agentMessage wsmsg.AgentMessage) (string, error) {
	switch wsmsg.MessageType(agentMessage.MessageType) {
	case wsmsg.HealthCheck:
		return "AliveCheckClusterToBastion", nil
	default:
		return "", fmt.Errorf("unsupported message type")
	}
}

func datachannelTargetSelectHandler(agentMessage wsmsg.AgentMessage) (string, error) {
	// First check if its a keysplitting message
	var keysplittingPayload map[string]interface{}
	if err := json.Unmarshal(agentMessage.MessagePayload, &keysplittingPayload); err == nil {
		if keysplittingPayloadVal, ok := keysplittingPayload["keysplittingPayload"].(map[string]interface{}); ok {
			switch keysplittingPayloadVal["action"] {
			case "kube/restapi":
				return "ResponseClusterToBastion", nil
			case "kube/exec/start":
				return "ResponseClusterToBastion", nil
			case "kube/exec/input":
				return "ResponseClusterToBastion", nil
			case "kube/exec/resize":
				return "ResponseClusterToBastion", nil
			case "kube/log/start":
				return "ResponseClusterToBastion", nil
			case "kube/log/stop":
				return "ResponseClusterToBastion", nil
			}
		}
	}

	// Else check if its a stream message
	var messagePayload smsg.StreamMessage
	if err := json.Unmarshal(agentMessage.MessagePayload, &messagePayload); err == nil {
		// p := payload["keysplittingPayload"].(map[string]interface{})
		switch messagePayload.Type {
		case "kube/exec/stdout":
			return "StdoutClusterToBastion", nil
		case "kube/exec/stderr":
			return "StderrClusterToBastion", nil
		case "kube/log/stdout":
			return "ResponseLogClusterToBastion", nil
		}
	}

	return "", fmt.Errorf("unable to determine SignalR endpoint")
}

func parseFlags() error {
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
		return fmt.Errorf("Missing flags! Missing: %v", missing)
	} else {
		return nil
	}
}

func getAgentVersion() string {
	if os.Getenv("DEV") == "true" {
		return "1.0"
	} else {
		return "$AGENT_VERSION"
	}
}
