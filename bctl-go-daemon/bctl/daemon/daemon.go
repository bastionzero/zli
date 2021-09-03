package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	dc "bastionzero.com/bctl/v1/bctl/daemon/datachannel"
	wsmsg "bastionzero.com/bctl/v1/bzerolib/channels/message"
)

// Declaring flags as package-accesible variables
var (
	sessionId, authHeader, assumeRole, assumeClusterId, serviceUrl           string
	daemonPort, localhostToken, environmentId, certPath, keyPath, configPath string
)

const (
	hubEndpoint   = "/api/v1/hub/kube"
	autoReconnect = true
)

func main() {
	parseFlags()

	// Setup our loggers
	// TODO: Pass in debug level as flag
	// TODO: Pass in stdout output as flag?
	logger, err := lggr.NewLogger(lggr.Debug, getLogFilePath(), true)
	if err != nil {
		os.Exit(1)
	}
	logger.AddDaemonVersion(version)
	dcLogger := logger.GetDatachannelLogger()

	logger.Info(fmt.Sprintf("Opening websocket to Bastion: %s", serviceUrl))
	startDatachannel(dcLogger)

	select {} // sleep forever?
}

func startDatachannel() {
	// Create our headers and params
	headers := make(map[string]string)
	headers["Authorization"] = authHeader

	// Add our token to our params
	params := make(map[string]string)
	params["session_id"] = sessionId
	params["assume_role"] = assumeRole
	params["assume_cluster_id"] = assumeClusterId
	params["environment_id"] = environmentId

	dataChannel, _ := dc.NewDataChannel(configPath, assumeRole, serviceUrl, hubEndpoint, params, headers, targetSelectHandler, autoReconnect)
	// TODO: Integrate this with existing messaging
	time.Sleep(3 * time.Second)
	dataChannel.SendSyn()

	if err := dataChannel.StartKubeDaemonPlugin(localhostToken, daemonPort, certPath, keyPath); err != nil {
		log.Printf("Error starting Kube Daemon plugin: %s", err.Error())
		return
	}

}

func targetSelectHandler(agentMessage wsmsg.AgentMessage) (string, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(agentMessage.MessagePayload, &payload); err == nil {
		if p, ok := payload["keysplittingPayload"].(map[string]interface{}); ok {
			switch p["action"] {
			case "kube/restapi":
				return "RequestDaemonToBastion", nil
			case "kube/exec/start":
				return "StartExecDaemonToBastion", nil
			case "kube/exec/input":
				return "StdinDaemonToBastion", nil
			case "kube/exec/resize":
				return "ResizeTerminalDaemonToBastion", nil
			case "kube/log/start":
				return "RequestLogDaemonToBastion", nil
			case "kube/log/stop":
				return "RequestLogDaemonToBastion", nil
			}
		} else {
			return "", fmt.Errorf("Fail on expected payload: %v", payload["keysplittingPayload"])
		}
	}
	return "", fmt.Errorf("")
}

func parseFlags() {
	flag.StringVar(&sessionId, "sessionId", "", "Session ID From Zli")
	flag.StringVar(&authHeader, "authHeader", "", "Auth Header From Zli")

	// Our expected flags we need to start
	flag.StringVar(&serviceUrl, "serviceURL", "", "Service URL to use")
	flag.StringVar(&assumeRole, "assumeRole", "", "Kube Role to Assume")
	flag.StringVar(&assumeClusterId, "assumeClusterId", "", "Kube Cluster Id to Connect to")
	flag.StringVar(&environmentId, "environmentId", "", "Environment Id of cluster we are connecting too")

	// Plugin variables
	flag.StringVar(&localhostToken, "localhostToken", "", "Localhost Token to Validate Kubectl commands")
	flag.StringVar(&daemonPort, "daemonPort", "", "Daemon Port To Use")
	flag.StringVar(&certPath, "certPath", "", "Path to cert to use for our localhost server")
	flag.StringVar(&keyPath, "keyPath", "", "Path to key to use for our localhost server")
	flag.StringVar(&configPath, "configPath", "", "Local storage path to zli config")

	log.Printf("configPath: %v", configPath)

	flag.Parse()

	// Check we have all required flags
	if sessionId == "" || authHeader == "" || assumeRole == "" || assumeClusterId == "" || serviceUrl == "" ||
		daemonPort == "" || localhostToken == "" || environmentId == "" || certPath == "" || keyPath == "" {
		log.Printf("Missing flags!")
		os.Exit(1)
	}
}
