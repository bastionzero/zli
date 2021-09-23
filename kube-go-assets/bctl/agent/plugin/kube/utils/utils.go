package utils

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/http"
)

func ValidateRequestId(requestIdPassed string, requestIdSaved string) error {
	if requestIdPassed != requestIdSaved {
		rerr := fmt.Errorf("invalid request ID passed")
		return rerr
	}
	return nil
}

func BuildHttpRequest(kubeHost string, endpoint string, body string, method string, headers map[string][]string, serviceAccountToken string, impersonateUser string, impersonateGroup string) *http.Request {
	// Perform the api request
	kubeApiUrl := kubeHost + endpoint
	bodyBytesReader := bytes.NewReader([]byte(body))
	req, _ := http.NewRequest(method, kubeApiUrl, bodyBytesReader)

	// Add any headers
	for name, values := range headers {
		// Loop over all values for the name.
		for _, value := range values {
			req.Header.Set(name, value)
		}
	}

	// Add our impersonation and token headers
	req.Header.Set("Authorization", "Bearer "+serviceAccountToken)
	req.Header.Set("Impersonate-User", impersonateUser)
	req.Header.Set("Impersonate-Group", impersonateGroup)

	// TODO: Figure out a way around this
	// CA certs can be found here /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	return req
}
