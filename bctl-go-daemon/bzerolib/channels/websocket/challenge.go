package websocket

import (
	"bytes"
	ed "crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/crypto/sha3"
)

func newChallenge(orgId string, clusterName string, serviceUrl string, privateKey string) (string, error) {
	// Get challenge
	challengeRequest := GetChallengeMessage{
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
		return "", fmt.Errorf("Error making post request to challenge agent. Error: %s. Response: %s", err, response)
	}
	defer response.Body.Close()

	// Extract the challenge
	responseDecoded := GetChallengeResponse{}
	json.NewDecoder(response.Body).Decode(&responseDecoded)

	// Solve Challenge
	return SignChallenge(privateKey, responseDecoded.Challenge)
}

func SignChallenge(privateKey string, challenge string) (string, error) {
	keyBytes, _ := base64.StdEncoding.DecodeString(privateKey)
	if len(keyBytes) != 64 {
		return "", fmt.Errorf("invalid private key length: %v", len(keyBytes))
	}
	privkey := ed.PrivateKey(keyBytes)

	hashBits := sha3.Sum256([]byte(challenge))

	sig := ed.Sign(privkey, hashBits[:])

	// Convert the signature to base64 string
	sigBase64 := base64.StdEncoding.EncodeToString(sig)

	return sigBase64, nil
}
