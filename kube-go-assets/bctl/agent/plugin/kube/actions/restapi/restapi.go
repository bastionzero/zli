package restapi

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

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

	// Perform the api request
	httpClient := &http.Client{}
	kubeApiUrl := r.kubeHost + apiRequest.Endpoint
	bodyBytesReader := bytes.NewReader([]byte(apiRequest.Body))
	req, _ := http.NewRequest(apiRequest.Method, kubeApiUrl, bodyBytesReader)

	// Add any headers
	for name, values := range apiRequest.Headers {
		// Loop over all values for the name.
		req.Header.Set(name, values)
	}

	// Add our impersonation and token headers
	req.Header.Set("Authorization", "Bearer "+r.serviceAccountToken)
	req.Header.Set("Impersonate-User", r.role)
	req.Header.Set("Impersonate-Group", r.impersonateGroup)

	// Make the request and wait for the body to close
	r.logger.Info(fmt.Sprintf("Making request for %s", kubeApiUrl))

	// TODO: Figure out a way around this
	// CA certs can be found here /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	res, err := httpClient.Do(req)
	if err != nil {
		rerr := fmt.Errorf("bad response to API request: %s", err)
		r.logger.Error(rerr)
		return action, []byte{}, rerr
	}
	defer res.Body.Close()

	// Build the header response
	header := make(map[string]string)
	for key, value := range res.Header {
		// TODO: This does not seem correct, we should add all headers even if they are dups
		header[key] = value[0]
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
