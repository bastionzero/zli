package restapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	kubeutils "bastionzero.com/bctl/v1/bctl/agent/plugin/kube/utils"
	lggr "bastionzero.com/bctl/v1/bzerolib/logger"
)

type RestApiAction struct {
	serviceAccountToken string
	kubeHost            string
	impersonateGroup    string
	role                string
	closed              bool
	logger              *lggr.Logger
}

func NewRestApiAction(logger *lggr.Logger, serviceAccountToken string, kubeHost string, impersonateGroup string, role string) (*RestApiAction, error) {
	return &RestApiAction{
		serviceAccountToken: serviceAccountToken,
		kubeHost:            kubeHost,
		impersonateGroup:    impersonateGroup,
		role:                role,
		logger:              logger,
		closed:              false,
	}, nil
}

func (r *RestApiAction) Closed() bool {
	return r.closed
}

func (r *RestApiAction) InputMessageHandler(action string, actionPayload []byte) (string, []byte, error) {
	defer func() {
		r.closed = true
	}()

	var apiRequest KubeRestApiActionPayload
	if err := json.Unmarshal(actionPayload, &apiRequest); err != nil {
		rerr := fmt.Errorf("malformed Keysplitting Action payload %v", actionPayload)
		r.logger.Error(rerr)
		return action, []byte{}, rerr
	}

	// Build the request
	r.logger.Info(fmt.Sprintf("Making request for %s", apiRequest.Endpoint))
	req := r.buildHttpRequest(apiRequest.Endpoint, apiRequest.Body, apiRequest.Method, apiRequest.Headers)

	httpClient := &http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		rerr := fmt.Errorf("bad response to API request: %s", err)
		r.logger.Error(rerr)
		return action, []byte{}, rerr
	}
	defer res.Body.Close()

	// Build the header response
	header := make(map[string][]string)
	for key, value := range res.Header {
		header[key] = value
	}

	// Parse out the body
	bodyBytes, _ := ioutil.ReadAll(res.Body)

	// Now we need to send that data back to the client
	responsePayload := KubeRestApiActionResponsePayload{
		StatusCode: res.StatusCode,
		RequestId:  apiRequest.RequestId,
		Headers:    header,
		Content:    bodyBytes,
	}
	responsePayloadBytes, _ := json.Marshal(responsePayload)

	return action, responsePayloadBytes, nil
}

func (r *RestApiAction) buildHttpRequest(endpoint, body, method string, headers map[string][]string) *http.Request {
	return kubeutils.BuildHttpRequest(r.kubeHost, endpoint, body, method, headers, r.serviceAccountToken, r.role, r.impersonateGroup)
}
