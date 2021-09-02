package kube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	exec "bastionzero.com/bctl/v1/bctl/daemon/plugin/kube/actions/exec"
	logaction "bastionzero.com/bctl/v1/bctl/daemon/plugin/kube/actions/logs"
	rest "bastionzero.com/bctl/v1/bctl/daemon/plugin/kube/actions/restapi"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
	plgn "bastionzero.com/bctl/v1/bzerolib/plugin"
	smsg "bastionzero.com/bctl/v1/bzerolib/stream/message"

	"github.com/google/uuid"
)

const (
	securityToken = "++++"
)

type JustRequestId struct {
	RequestId string `json:"requestId"`
}

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
	PushStreamResponse(streamMessage smsg.StreamMessage)
}

type KubeDaemonPlugin struct {
	localhostToken string
	daemonPort     string
	certPath       string
	keyPath        string

	// Input and output streams
	streamResponseChannel chan smsg.StreamMessage
	RequestChannel        chan plgn.ActionWrapper

	// Done channel to bubble up error to the user
	DoneChannel chan string
	ExitMessage string

	actions map[string]IKubeDaemonAction

	mapLock sync.RWMutex
	logger  *lggr.Logger
}

func NewKubeDaemonPlugin(logger *lggr.Logger,
	localhostToken string,
	daemonPort string,
	certPath string,
	keyPath string,
	doneChannel chan string) (*KubeDaemonPlugin, error) {

	plugin := KubeDaemonPlugin{
		localhostToken:        localhostToken,
		daemonPort:            daemonPort,
		certPath:              certPath,
		keyPath:               keyPath,
		streamResponseChannel: make(chan smsg.StreamMessage, 100),
		RequestChannel:        make(chan plgn.ActionWrapper, 100),
		DoneChannel:           doneChannel,
		ExitMessage:           "",
		actions:               make(map[string]IKubeDaemonAction),
		mapLock:               sync.RWMutex{},
		logger:                logger,
	}

	// get this from parent
	ctx := context.TODO()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case streamMessage := <-plugin.streamResponseChannel:
				plugin.handleStreamMessage(streamMessage)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case doneMessage := <-plugin.DoneChannel:
				plugin.ExitMessage = doneMessage
			}
		}
	}()

	go func() {
		// Define our http handlers
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			plugin.rootCallback(w, r)
		})

		// TODO: Figure out what to do with this
		log.Fatal(http.ListenAndServeTLS(":"+plugin.daemonPort, plugin.certPath, plugin.keyPath, nil))
	}()

	return &plugin, nil
}

func (k *KubeDaemonPlugin) handleStreamMessage(smessage smsg.StreamMessage) error {
	if act, ok := k.actions[smessage.RequestId]; ok {
		act.PushStreamResponse(smessage)
		return nil
	} else {
		rerr := fmt.Errorf("unknown request ID: %v", smessage.RequestId)
		k.logger.Error(rerr)
		return rerr
	}
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
		// Get just the request ID so we can associate it with the previously started action object
		var d JustRequestId
		if err := json.Unmarshal(actionPayload, &d); err != nil {
			rerr := fmt.Errorf("could not unmarshal json: %s", err)
			k.logger.Error(rerr)
			return "", []byte{}, rerr
		} else {
			if act, ok := k.actions[d.RequestId]; ok {
				wrappedAction := plgn.ActionWrapper{
					Action:        action,
					ActionPayload: actionPayload,
				}
				act.PushKSResponse(wrappedAction)
			} else {
				rerr := fmt.Errorf("unknown request ID: %v", d.RequestId)
				k.logger.Error(rerr)
				return "", []byte{}, rerr
			}
		}
	}

	k.logger.Info("Waiting for input...")
	select {
	case actionMessage := <-k.RequestChannel:
		msg := fmt.Sprintf("Received input from action: %v", actionMessage.Action)
		k.logger.Info(msg)

		actionPayloadBytes, _ := json.Marshal(actionMessage.ActionPayload)
		return actionMessage.Action, actionPayloadBytes, nil
		// case <-time.After(time.Second * 10): // a better solution is to have a cancel channel
		// 	return "", "", fmt.Errorf("TIMEOUT!")
	}
}

func generateRequestId() string {
	return uuid.New().String()
}

func (k *KubeDaemonPlugin) rootCallback(w http.ResponseWriter, r *http.Request) {
	msg := fmt.Sprintf("Handling %s - %s\n", r.URL.Path, r.Method)
	k.logger.Info(msg)

	if k.ExitMessage != "" {
		// Return the exit message to the user
		w.WriteHeader(http.StatusInternalServerError)
		msg := fmt.Sprintf("Daemon connection has been closed by Bastion. Message: " + k.ExitMessage)
		k.logger.Info(msg)
		w.Write([]byte(msg))
		return
	}

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
		k.logger.Error(errors.New("http internal server error"))
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
		subLogger := k.logger.GetActionLogger(string(Exec))
		subLogger.AddRequestId(requestId)

		execAction, _ := exec.NewExecAction(subLogger, requestId, logId, k.RequestChannel, k.streamResponseChannel, commandBeingRun)

		k.updateActionsMap(execAction, requestId)

		k.logger.Info(fmt.Sprintf("Created Exec action with requestId %v", requestId))
		if err := execAction.InputMessageHandler(w, r); err != nil {
			k.logger.Error(fmt.Errorf("error handling Exec call: %s", err))
		}
	} else if strings.Contains(r.URL.Path, "log") { // TODO : maybe ends with?
		subLogger := k.logger.GetActionLogger(string(Log))
		subLogger.AddRequestId(requestId)

		logAction, _ := logaction.NewLogAction(subLogger, requestId, logId, k.RequestChannel)

		k.updateActionsMap(logAction, requestId)

		k.logger.Info(fmt.Sprintf("Created Log action with requestId %v", requestId))
		if err := logAction.InputMessageHandler(w, r); err != nil {
			k.logger.Error(fmt.Errorf("error handling Logs call: %s", err))
		}
	} else {
		subLogger := k.logger.GetActionLogger(string(RestApi))
		subLogger.AddRequestId(requestId)

		restAction, _ := rest.NewRestApiAction(subLogger, requestId, logId, k.RequestChannel, commandBeingRun)

		k.updateActionsMap(restAction, requestId)

		k.logger.Info(fmt.Sprintf("Created Rest API action with requestId %v", requestId))
		if err := restAction.InputMessageHandler(w, r); err != nil {
			k.logger.Error(fmt.Errorf("error handling REST API call: %s", err))
		}
	}
}

func (k *KubeDaemonPlugin) updateActionsMap(newAction IKubeDaemonAction, id string) {
	// Helper function so we avoid writing to this map at the same time
	k.mapLock.Lock()
	k.actions[id] = newAction
	k.mapLock.Unlock()
}
