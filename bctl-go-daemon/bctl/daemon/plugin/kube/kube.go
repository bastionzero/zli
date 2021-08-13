package kube

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	exec "bastionzero.com/bctl/v1/bctl/daemon/plugin/kube/actions/exec"
	logaction "bastionzero.com/bctl/v1/bctl/daemon/plugin/kube/actions/logs"
	rest "bastionzero.com/bctl/v1/bctl/daemon/plugin/kube/actions/restapi"
	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"

	"github.com/google/uuid"
)

const (
	securityToken = "++++"
)

type KubeDaemonAction string

const (
	Exec    KubeDaemonAction = "exec"
	Log     KubeDaemonAction = "log"
	RestApi KubeDaemonAction = "restapi"
)

// Perhaps unnecessary but it is nice to make sure that each action is implementing a common function set
type IKubeDaemonAction interface {
	InputMessageHandler(writer http.ResponseWriter, request *http.Request) error
	PushKSResponse(actionWrapper plgn.ActionWrapper)
}

type KubeDaemonPlugin struct {
	localhostToken string
	daemonPort     string
	certPath       string
	keyPath        string

	// Input and output streams
	streamResponseChannel chan smsg.StreamMessage
	RequestChannel        chan plgn.ActionWrapper

	// To keep track of all current, ongoing stream output channels for exec
	execStdoutMapping map[string]chan smsg.StreamMessage
	logChannelMapping map[string]chan smsg.StreamMessage

	actions map[string]IKubeDaemonAction

	mapLock sync.RWMutex
}

func NewKubeDaemonPlugin(localhostToken string, daemonPort string, certPath string, keyPath string) (*KubeDaemonPlugin, error) {
	plugin := KubeDaemonPlugin{
		localhostToken:        localhostToken,
		daemonPort:            daemonPort,
		certPath:              certPath,
		keyPath:               keyPath,
		streamResponseChannel: make(chan smsg.StreamMessage, 100),
		RequestChannel:        make(chan plgn.ActionWrapper, 100),
		execStdoutMapping:     make(map[string]chan smsg.StreamMessage),
		logChannelMapping:     make(map[string]chan smsg.StreamMessage),
		actions:               make(map[string]IKubeDaemonAction),
		mapLock:               sync.RWMutex{},
	}

	go func() {
		for {
			select {
			case streamMessage := <-plugin.streamResponseChannel:
				plugin.handleStreamMessage(streamMessage)
			}
		}
	}()

	go func() {
		// Define our http handlers
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			plugin.rootCallback(w, r)
		})

		log.Fatal(http.ListenAndServeTLS(":"+plugin.daemonPort, plugin.certPath, plugin.keyPath, nil))
	}()

	return &plugin, nil
}

func (k *KubeDaemonPlugin) handleStreamMessage(smessage smsg.StreamMessage) error {
	// Push stdout to the correct handler, first get the kube action
	switch smsg.StreamType(smessage.Type) {
	case smsg.StdOut:
		k.execStdoutMapping[smessage.RequestId] <- smessage
		break
	case smsg.LogOut:
		k.logChannelMapping[smessage.RequestId] <- smessage
		break
	}

	return nil
}

func (k *KubeDaemonPlugin) PushStreamInput(smessage smsg.StreamMessage) error {
	k.streamResponseChannel <- smessage // maybe we don't need a middleman channel? eh, probably even if it's just a buffer
	return nil
}

func (k *KubeDaemonPlugin) GetName() plgn.PluginName {
	return plgn.KubeDaemon
}

func (k *KubeDaemonPlugin) InputMessageHandler(action string, actionPayload []byte) (string, []byte, error) {
	if len(actionPayload) > 0 {
		if x := strings.Split(action, "/"); len(x) <= 1 {
			return "", []byte{}, fmt.Errorf("Malformed action: %v", action)
		} else {
			var payload map[string]interface{}
			if err := json.Unmarshal([]byte(actionPayload), &payload); err != nil {
				return "", []byte{}, fmt.Errorf("Could not unmarshal actionPayload: %v", string(actionPayload))
			} else {
				// Json always unmarshals numbers as float64
				if id, ok := payload["requestId"].(string); ok {
					log.Printf("Plugin received response for action with request ID: %v", id)
					if act, ok := k.actions[id]; ok {
						wrappedAction := plgn.ActionWrapper{
							Action:        action,
							ActionPayload: actionPayload,
						}
						act.PushKSResponse(wrappedAction)
					} else {
						log.Printf("%+v", k.actions)
						return "", []byte{}, fmt.Errorf("Unknown Request ID")
					}
				} else {
					return "", []byte{}, fmt.Errorf("Action payload must include request ID")
				}
			}
		}
	}
	// TODO: check that plugin name is "kube"

	log.Printf("Waiting for input...")
	select {
	case actionMessage := <-k.RequestChannel:
		log.Printf("Received input from action: %v", actionMessage.Action)
		actionPayloadBytes, _ := json.Marshal(actionMessage.ActionPayload)
		return actionMessage.Action, actionPayloadBytes, nil
		// case <-time.After(time.Second * 10): // a better solution is to have a cancel channel
		// 	return "", "", fmt.Errorf("TIMEOUT!")
	}
}

func generateRequestId() string {
	// gotta mix up the see otherwise it always gives us the same number
	// rand.Seed(time.Now().UnixNano())
	// return rand.Intn(10000) // We REALLY want to be using a uuid
	return uuid.New().String()
}

func (k *KubeDaemonPlugin) rootCallback(w http.ResponseWriter, r *http.Request) {
	log.Printf("Handling %s - %s\n", r.URL.Path, r.Method)

	// Trim off localhost token
	// TODO: Fix this
	k.localhostToken = strings.Replace(k.localhostToken, securityToken, "", -1) // ?

	// First verify our token and extract any commands if we can
	tokenToValidate := r.Header.Get("Authorization")

	// Remove the `Bearer `
	tokenToValidate = strings.Replace(tokenToValidate, "Bearer ", "", -1)

	// Validate the token
	tokensSplit := strings.Split(tokenToValidate, securityToken)
	if tokensSplit[0] != k.localhostToken {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Check if we have a command to extract
	// TODO: Maybe we can push this work to the bastion
	commandBeingRun := "N/A"
	logId := "N/A"
	if len(tokensSplit) == 3 {
		commandBeingRun = tokensSplit[1]
		logId = tokensSplit[2]
	} else {
		commandBeingRun = "N/A"
		logId = generateRequestId()
	}

	// Always generate requestId
	requestId := generateRequestId()

	if strings.Contains(r.URL.Path, "exec") {
		// Make a new channel for stdout
		stdoutChannel := make(chan smsg.StreamMessage)

		// Update our mapping
		k.execStdoutMapping[requestId] = stdoutChannel

		execAction, _ := exec.NewExecAction(requestId, logId, k.RequestChannel, k.streamResponseChannel, stdoutChannel, commandBeingRun)

		k.updateActionsMap(execAction, requestId)

		log.Printf("Created Exec action with requestId %v", requestId)
		if err := execAction.InputMessageHandler(w, r); err != nil {
			log.Printf("Error handling Exec call: %s", err.Error())
		}
	} else if strings.Contains(r.URL.Path, "log") { // TODO : maybe ends with?
		// Make a new channel for log output
		logChannel := make(chan smsg.StreamMessage)

		// Update our mapping
		k.logChannelMapping[requestId] = logChannel

		logAction, _ := logaction.NewLogAction(requestId, logId, k.RequestChannel, logChannel)

		k.updateActionsMap(logAction, requestId)

		log.Printf("Created Log action with requestId %v", requestId)
		if err := logAction.InputMessageHandler(w, r); err != nil {
			log.Printf("Error handling Logs call: %s", err.Error())
		}
	} else {
		restAction, _ := rest.NewRestApiAction(requestId, logId, k.RequestChannel, commandBeingRun)

		k.updateActionsMap(restAction, requestId)

		log.Printf("Created Rest API action with requestId %v", requestId)
		if err := restAction.InputMessageHandler(w, r); err != nil {
			log.Printf("Error handling REST API call: %s", err.Error())
		}
	}
}

func (k *KubeDaemonPlugin) updateActionsMap(newAction IKubeDaemonAction, id string) {
	// Helper function so we avoid writing to this map at the same time
	k.mapLock.Lock()
	k.actions[id] = newAction
	k.mapLock.Unlock()
}
