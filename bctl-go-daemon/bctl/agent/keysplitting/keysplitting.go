package keysplitting

import (
	ed "crypto/ed25519"
	"encoding/base64"
	"fmt"
	"time"

	bzcrt "bastionzero.com/bctl/v1/bzerolib/keysplitting/bzcert"
	ksmsg "bastionzero.com/bctl/v1/bzerolib/keysplitting/message"
	"bastionzero.com/bctl/v1/bzerolib/keysplitting/util"
)

const (
	schemaVersion = "1.0"
)

type BZCertMetadata struct {
	Cert bzcrt.BZCert
	Exp  time.Time
}

type IKeysplitting interface {
	Validate(ksMessage *ksmsg.KeysplittingMessage) error
	BuildResponse(ksMessage *ksmsg.KeysplittingMessage, action string, actionPayload []byte) (ksmsg.KeysplittingMessage, error)
}

type Keysplitting struct {
	hPointer         string
	expectedHPointer string
	bzCerts          map[string]BZCertMetadata // only for agent
	publickey        string
	privatekey       string
}

func NewKeysplitting() (IKeysplitting, error) {
	// Generate public private key pair along ed25519 curve
	if publicKey, privateKey, err := ed.GenerateKey(nil); err != nil {
		return &Keysplitting{}, fmt.Errorf("error generating key pair: %v", err.Error())
	} else {
		pubkeyString := base64.StdEncoding.EncodeToString([]byte(publicKey))
		privkeyString := base64.StdEncoding.EncodeToString([]byte(privateKey))

		return &Keysplitting{
			hPointer:         "",
			expectedHPointer: "",
			bzCerts:          make(map[string]BZCertMetadata),
			publickey:        pubkeyString,
			privatekey:       privkeyString,
		}, nil
	}
}

func (k *Keysplitting) Validate(ksMessage *ksmsg.KeysplittingMessage) error {
	switch ksMessage.Type {
	case ksmsg.Syn:
		synPayload := ksMessage.KeysplittingPayload.(ksmsg.SynPayload)

		// Verify the BZCert
		if hash, exp, err := synPayload.BZCert.Verify(); err != nil {
			return err
		} else {
			k.bzCerts[hash] = BZCertMetadata{
				Cert: synPayload.BZCert,
				Exp:  exp,
			}
		}

		// Verify the Signature
		if err := ksMessage.VerifySignature(synPayload.BZCert.ClientPublicKey); err != nil {
			return err
		}

		// Make sure targetId matches
		// if synPayload.TargetId != k.publickey {
		// 	return fmt.Errorf("syn's TargetId did not match Target's actual ID")
		// }
	case ksmsg.Data:
		dataPayload := ksMessage.KeysplittingPayload.(ksmsg.DataPayload)

		// Check BZCert matches one we have stored
		if certMetadata, ok := k.bzCerts[dataPayload.BZCertHash]; !ok {
			return fmt.Errorf("could not match BZCert hash to one previously received")
		} else {

			// Verify the Signature
			if err := ksMessage.VerifySignature(certMetadata.Cert.ClientPublicKey); err != nil {
				return err
			}
		}

		// Verify recieved hash pointer matches expected
		if dataPayload.HPointer != k.expectedHPointer {
			return fmt.Errorf("data's hash pointer did not match expected")
		}

		// Make sure targetId matches
		// if dataPayload.TargetId != k.publickey {
		// 	return fmt.Errorf("data's TargetId did not match Target's actual ID")
		// }
	default:
		return fmt.Errorf("error validating unhandled Keysplitting type")
	}
	return nil
}

func (k *Keysplitting) BuildResponse(ksMessage *ksmsg.KeysplittingMessage, action string, actionPayload []byte) (ksmsg.KeysplittingMessage, error) {
	var responseMessage ksmsg.KeysplittingMessage

	switch ksMessage.Type {
	case ksmsg.Syn:
		synPayload := ksMessage.KeysplittingPayload.(ksmsg.SynPayload)
		if synAckPayload, hash, err := synPayload.BuildResponsePayload(actionPayload, k.publickey); err != nil {
			return ksmsg.KeysplittingMessage{}, err
		} else {
			k.hPointer = hash
			responseMessage = ksmsg.KeysplittingMessage{
				Type:                ksmsg.SynAck,
				KeysplittingPayload: synAckPayload,
			}
		}
	case ksmsg.Data:
		dataPayload := ksMessage.KeysplittingPayload.(ksmsg.DataPayload)
		if dataAckPayload, hash, err := dataPayload.BuildResponsePayload(actionPayload, k.publickey); err != nil {
			return ksmsg.KeysplittingMessage{}, err
		} else {
			k.hPointer = hash
			responseMessage = ksmsg.KeysplittingMessage{
				Type:                ksmsg.DataAck,
				KeysplittingPayload: dataAckPayload,
			}
		}
	}

	hashBytes, _ := util.HashPayload(responseMessage.KeysplittingPayload)
	k.expectedHPointer = base64.StdEncoding.EncodeToString(hashBytes)

	// Sign it and send it
	if err := responseMessage.Sign(k.privatekey); err != nil {
		return responseMessage, fmt.Errorf("could not sign payload: %v", err.Error())
	} else {
		return responseMessage, nil
	}
}
