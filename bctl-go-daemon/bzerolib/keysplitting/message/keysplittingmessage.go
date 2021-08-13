package message

import (
	ed "crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"bastionzero.com/bctl/v1/bzerolib/keysplitting/util"
)

// Type restrictions for keysplitting messages
type KeysplittingPayloadType string

const (
	Syn     KeysplittingPayloadType = "Syn"
	SynAck  KeysplittingPayloadType = "SynAck"
	Data    KeysplittingPayloadType = "Data"
	DataAck KeysplittingPayloadType = "DataAck"
)

const (
	SchemaVersion = "1.0"
)

type IKeysplittingMessage interface {
	BuildResponse(actionPayload interface{}, publickey string) (KeysplittingMessage, error)
	VerifySignature(publicKey string) error
	Sign(privateKey string) error
}

type KeysplittingMessage struct {
	Type                KeysplittingPayloadType `json:"type"`
	KeysplittingPayload interface{}             `json:"keysplittingPayload"`
	Signature           string                  `json:"signature"`
}

func (k *KeysplittingMessage) VerifySignature(publicKey string) error {
	pubKeyBits, _ := base64.StdEncoding.DecodeString(publicKey)
	if len(pubKeyBits) != 32 {
		return fmt.Errorf("Public Key has invalid length %v", len(pubKeyBits))
	}
	pubkey := ed.PublicKey(pubKeyBits)

	hashBits, ok := util.HashPayload(k.KeysplittingPayload)
	if !ok {
		return fmt.Errorf("Could not hash the keysplitting payload")
	}

	sigBits, _ := base64.StdEncoding.DecodeString(k.Signature)

	if ok := ed.Verify(pubkey, hashBits, sigBits); ok {
		return nil
	} else {
		return fmt.Errorf("Failed to verify signature on rand")
	}
}

func (k *KeysplittingMessage) Sign(privateKey string) error {
	keyBytes, _ := base64.StdEncoding.DecodeString(privateKey)
	if len(keyBytes) != 64 {
		return fmt.Errorf("Invalid private key length: %v", len(keyBytes))
	}
	privkey := ed.PrivateKey(keyBytes)

	hashBits, _ := util.HashPayload(k.KeysplittingPayload)

	sig := ed.Sign(privkey, hashBits)
	k.Signature = base64.StdEncoding.EncodeToString(sig)
	return nil
}

func (k *KeysplittingMessage) UnmarshalJSON(data []byte) error {
	var objmap map[string]*json.RawMessage

	if err := json.Unmarshal(data, &objmap); err != nil {
		return err
	}

	var t, s string
	if err := json.Unmarshal(*objmap["type"], &t); err != nil {
		return err
	} else {
		k.Type = KeysplittingPayloadType(t)
	}

	if err := json.Unmarshal(*objmap["signature"], &s); err != nil {
		return err
	} else {
		k.Signature = s
	}

	kPayload := *objmap["keysplittingPayload"]
	switch k.Type {
	case Syn:
		var synPayload SynPayload
		if err := json.Unmarshal(kPayload, &synPayload); err != nil {
			return fmt.Errorf("Malformed Syn Payload")
		} else {
			k.KeysplittingPayload = synPayload
		}
	case SynAck:
		var synAckPayload SynAckPayload
		if err := json.Unmarshal(kPayload, &synAckPayload); err != nil {
			return fmt.Errorf("Malformed SynAck Payload")
		} else {
			k.KeysplittingPayload = synAckPayload
		}
	case Data:
		var dataPayload DataPayload
		if err := json.Unmarshal(kPayload, &dataPayload); err != nil {
			return fmt.Errorf("Malformed Data Payload")
		} else {
			k.KeysplittingPayload = dataPayload
		}
	case DataAck:
		var dataAckPayload DataAckPayload
		if err := json.Unmarshal(kPayload, &dataAckPayload); err != nil {
			return fmt.Errorf("Malformed DataAck Payload")
		} else {
			k.KeysplittingPayload = dataAckPayload
		}
	default:
		// TODO: explicitly check type of outer vs. inner payload
		return fmt.Errorf("Type mismatch in keysplitting message and actual message payload")
	}

	return nil
}
