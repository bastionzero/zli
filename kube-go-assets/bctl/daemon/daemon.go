package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	dc "bastionzero.com/bctl/v1/bctl/daemon/datachannel"
	wsmsg "bastionzero.com/bctl/v1/bzerolib/channels/message"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
)

// Declaring flags as package-accesible variables
var (
	sessionId, authHeader, assumeRole, assumeClusterId, serviceUrl           string
	daemonPort, localhostToken, environmentId, certPath, keyPath, configPath string
	logPath                                                                  string
)

const (
	hubEndpoint   = "/api/v1/hub/kube"
	autoReconnect = true
	version       = "1.0.0" // TODO: Change this?
)

func main() {
	parseFlags() // TODO: Output missing args error

	// Setup our loggers
	// TODO: Pass in debug level as flag
	// TODO: Pass in stdout output as flag?
	logger, err := lggr.NewLogger(lggr.Debug, getLogFilePath())
	if err != nil {
		os.Exit(1)
	}
	logger.AddDaemonVersion(version)
	dcLogger := logger.GetDatachannelLogger()

	logger.Info(fmt.Sprintf("Opening websocket to Bastion: %s", serviceUrl))
	startDatachannel(dcLogger)

	select {} // sleep forever?
}

func startDatachannel(logger *lggr.Logger) {
	// Create our headers and params
	headers := make(map[string]string)
	headers["Authorization"] = authHeader

	// Add our token to our params
	params := make(map[string]string)
	params["session_id"] = sessionId
	params["assume_role"] = assumeRole
	params["assume_cluster_id"] = assumeClusterId
	params["environment_id"] = environmentId

	dataChannel, _ := dc.NewDataChannel(logger, configPath, assumeRole, serviceUrl, hubEndpoint, params, headers, targetSelectHandler, autoReconnect)

	if err := dataChannel.StartKubeDaemonPlugin(localhostToken, daemonPort, certPath, keyPath); err != nil {
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
				return "RequestHttpStreamDaemonToBastion", nil
			case "kube/log/stop":
				return "StopHttpStreamDaemonToBastion", nil
			case "kube/watch/start":
				return "RequestHttpStreamDaemonToBastion", nil
			case "kube/watch/stop":
				return "StopHttpStreamDaemonToBastion", nil
			}
		} else {
			return "", fmt.Errorf("fail on expected payload: %v", payload["keysplittingPayload"])
		}
	}
	return "", fmt.Errorf("")
}

func parseFlags() error {
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
	flag.StringVar(&logPath, "logPath", "", "Path to log file for daemon")

	flag.Parse()

	// Check we have all required flags
	if sessionId == "" || authHeader == "" || assumeRole == "" || assumeClusterId == "" || serviceUrl == "" ||
		daemonPort == "" || localhostToken == "" || environmentId == "" || certPath == "" || keyPath == "" ||
		logPath == "" || configPath == "" {
		return fmt.Errorf("missing flags")
	} else {
		return nil
	}
}

func getLogFilePath() string {
	return logPath
}
