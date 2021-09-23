package websocket

import (
	"bytes"
	ed "crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	wsmsg "bastionzero.com/bctl/v1/bzerolib/channels/message"
	"golang.org/x/crypto/sha3"
)

func newChallenge(orgId string, clusterName string, serviceUrl string, privateKey string) (string, error) {
	// Get challenge
	challengeRequest := wsmsg.GetChallengeMessage{
		OrgId:       orgId,
		ClusterName: clusterName,
	}

	challengeJson, err := json.Marshal(challengeRequest)
	if err != nil {
		return "", fmt.Errorf("Error marshalling register data: %s", err)
	}

	// Make our POST request
	response, err := http.Post("https://"+serviceUrl+challengeEndpoint, "application/json",
		bytes.NewBuffer(challengeJson))
	if err != nil || response.StatusCode != http.StatusOK {
		// If the status code is unauthorized, retrun an unauth error, error just a generic one
		if response.StatusCode == http.StatusInternalServerError {
			return "", fmt.Errorf("500")
		}
		return "", fmt.Errorf("Error making post request to challenge agent. Error: %s. Response: %d", err, response.StatusCode)
	}
	defer response.Body.Close()

	// Extract the challenge
	responseDecoded := wsmsg.GetChallengeResponse{}
	json.NewDecoder(response.Body).Decode(&responseDecoded)

	// Solve Challenge
	return signString(privateKey, responseDecoded.Challenge)
}

func signString(privateKey string, content string) (string, error) {
	keyBytes, _ := base64.StdEncoding.DecodeString(privateKey)
	if len(keyBytes) != 64 {
		return "", fmt.Errorf("invalid private key length: %v", len(keyBytes))
	}
	privkey := ed.PrivateKey(keyBytes)

	hashBits := sha3.Sum256([]byte(content))

	sig := ed.Sign(privkey, hashBits[:])

	// Convert the signature to base64 string
	sigBase64 := base64.StdEncoding.EncodeToString(sig)

	return sigBase64, nil
}
