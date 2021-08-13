package kube

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	exec "bastionzero.com/bctl/v1/bctl/agent/plugin/kube/actions/exec"
	logaction "bastionzero.com/bctl/v1/bctl/agent/plugin/kube/actions/logs"
	rest "bastionzero.com/bctl/v1/bctl/agent/plugin/kube/actions/restapi"
	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"
	stdreader "bastionzero.com/bctl/v1/bzerolib/stream/stdreader"
)

const (
	impersonateGroup = "system:authenticated"
)

type IKubeAction interface {
	InputMessageHandler(action string, actionPayload string) (string, []byte, error)
}

type KubeAction string

const (
	Exec    KubeAction = "exec"
	Log     KubeAction = "log"
	RestApi KubeAction = "restapi"
)

type KubePlugin struct {
	role                string
	streamOutputChannel chan smsg.StreamMessage
	serviceAccountToken string
	kubeHost            string
	runningExecActions  map[string]*exec.ExecAction // need something like this for streams and multiple tabs when running exec & logs
}

func NewPlugin(ch chan smsg.StreamMessage, role string) plgn.IPlugin {
	// First load in our Kube variables
	// TODO: Where should we save this, in the class? is this the best way to do this?
	// TODO: Also we should be able to drop this req, and just load `IN CLUSTER CONFIG`
	serviceAccountTokenPath := os.Getenv("KUBERNETES_SERVICE_ACCOUNT_TOKEN_PATH")
	serviceAccountTokenBytes, _ := ioutil.ReadFile(serviceAccountTokenPath)
	// TODO: Check for error
	serviceAccountToken := string(serviceAccountTokenBytes)
	kubeHost := "https://" + os.Getenv("KUBERNETES_SERVICE_HOST")

	return &KubePlugin{
		role:                role,
		streamOutputChannel: ch,
		serviceAccountToken: serviceAccountToken,
		kubeHost:            kubeHost,
		runningExecActions:  make(map[string]*exec.ExecAction),
	}
}

func (k *KubePlugin) GetName() plgn.PluginName {
	return plgn.Kube
}

func (k *KubePlugin) PushStreamInput(smessage smsg.StreamMessage) error {
	return fmt.Errorf("")
}

func (k *KubePlugin) InputMessageHandler(action string, actionPayload []byte) (string, []byte, error) {
	log.Printf("Plugin received Data message with %v action", action)
	x := strings.Split(action, "/")
	if len(x) < 2 {
		return "", []byte{}, fmt.Errorf("Malformed action: %s", action)
	}
	kubeAction := x[1]

	// TODO: The below line removes the extra, surrounding quotation marks that get added at some point in the marshal/unmarshal
	// so it messes up the umarshalling into a valid action payload.  We need to figure out why this is happening
	// so that we can murder its family
	actionPayload = actionPayload[1 : len(actionPayload)-1]

	// Json unmarshalling encodes bytes in base64
	actionPayloadSafe, _ := base64.StdEncoding.DecodeString(string(actionPayload))

	switch KubeAction(kubeAction) {
	case RestApi:
		a, _ := rest.NewRestApiAction(k.serviceAccountToken, k.kubeHost, impersonateGroup, k.role)
		return a.InputMessageHandler(action, actionPayloadSafe)
	case Exec:
		// Determin what subcommand is it so we can get the request id
		switch exec.ExecSubAction(action) {
		case exec.StartExec:
			// Unmarshal the message to get the requestId
			var startExecRequest exec.KubeExecStartActionPayload
			if err := json.Unmarshal(actionPayloadSafe, &startExecRequest); err != nil {
				log.Printf("Error unmarshaling start: %v", err.Error())
				return "", []byte{}, fmt.Errorf("Unable to unmarshal start message")
			}

			// Create our new exec action
			a, _ := exec.NewExecAction(k.serviceAccountToken, k.kubeHost, impersonateGroup, k.role, k.streamOutputChannel, startExecRequest.LogId)

			// Add this to our running actions so we can search for this later
			k.runningExecActions[startExecRequest.RequestId] = a

			return a.StartExec(startExecRequest)
		case exec.ExecInput:
			// Unmarshal the message to get the requestId
			var execStdinAction exec.KubeStdinActionPayload
			if err := json.Unmarshal(actionPayloadSafe, &execStdinAction); err != nil {
				log.Printf("Error unmarshaling input: %v", err.Error())
				return "", []byte{}, fmt.Errorf("Unable to unmarshal stdin message")
			}

			// Check if we need to end the stream
			var toSend []byte = execStdinAction.Stdin
			if execStdinAction.End == true {
				toSend = stdreader.EndStreamBytes
			}

			// Send the message to our running action
			return k.runningExecActions[execStdinAction.RequestId].SendStdinToChannel(toSend)
		case exec.ExecResize:
			// Unmarshal the message to get the requestId
			var execResizeAction exec.KubeExecResizeActionPayload
			if err := json.Unmarshal(actionPayloadSafe, &execResizeAction); err != nil {
				log.Printf("Error unmarshaling resize: %v", err.Error())
				return "", []byte{}, fmt.Errorf("Unable to unmarshal resize message")
			}

			// Send the message to our running action
			return k.runningExecActions[execResizeAction.RequestId].SendResizeToChannel(execResizeAction)
		}
		break
	case Log:
		a, _ := logaction.NewLogAction(k.serviceAccountToken, k.kubeHost, impersonateGroup, k.role, k.streamOutputChannel)
		return a.InputMessageHandler(action, actionPayloadSafe)
		break
	default:
		return "", []byte{}, fmt.Errorf("Unhandled action")
	}

	return "", []byte{}, nil
}
