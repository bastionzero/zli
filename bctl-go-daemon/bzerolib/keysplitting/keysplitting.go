package keysplitting

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	bzcrt "bastionzero.com/bctl/v1/bzerolib/keysplitting/bzcert"
	ksmsg "bastionzero.com/bctl/v1/bzerolib/keysplitting/message"
	"bastionzero.com/bctl/v1/bzerolib/keysplitting/util"
)

const (
	schemaVersion = "1.0"

	// Config is in json
	// keysplittingConfigName  = "keySplitting"
	// tokenConfigName         = "tokenSet"
	// currentIdTokenFieldName = "id_token"
)

type Config struct {
	KSConfig KeysplittingConfig `json:"keySplitting"`
	TokenSet TokenSetConfig     `json:"tokenSet"`
}

type KeysplittingConfig struct {
	PrivateKey       string `json:"privateKey"`
	PublicKey        string `json:"publicKey"`
	CerRand          string `json:"cerRand"`
	CerRandSignature string `json:"cerRandSig"`
	InitialIdToken   string `json:"initialIdToken"`
}

type TokenSetConfig struct {
	CurrentIdToken string `json:"id_token"`
}

type BZCertMetadata struct {
	Cert bzcrt.BZCert
	Exp  time.Time
}

type IKeysplitting interface {
	BuildSyn(action string, payload []byte) (ksmsg.KeysplittingMessage, error)
	Validate(ksMessage *ksmsg.KeysplittingMessage) error
	BuildResponse(ksMessage *ksmsg.KeysplittingMessage, action string, actionPayload []byte) (ksmsg.KeysplittingMessage, error)
}

type Keysplitting struct {
	hPointer         string
	expectedHPointer string
	bzCerts          map[string]BZCertMetadata // only for agent
	publickey        string
	privatekey       string

	// daemon variables
	targetId   string
	configPath string
	bzcertHash string // Might not need this because we should be checking config everytime
}

func NewKeysplitting(targetId string, configPath string) (IKeysplitting, error) {
	// TODO: load keys from storage
	return &Keysplitting{
		hPointer:         "",
		expectedHPointer: "",
		bzCerts:          make(map[string]BZCertMetadata),
		publickey:        "legit",
		privatekey:       "superlegit",
		targetId:         targetId,
		configPath:       configPath,
	}, nil
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
		if synPayload.TargetId != k.publickey {
			return fmt.Errorf("syn's TargetId did not match Target's actual ID")
		}
	case ksmsg.SynAck:
		synAckPayload := ksMessage.KeysplittingPayload.(ksmsg.SynAckPayload)

		// Verify recieved hash pointer matches expected
		if synAckPayload.HPointer != k.expectedHPointer {
			return fmt.Errorf("SynAck's hash pointer did not match expected")
		}
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
		if dataPayload.TargetId != k.publickey {
			return fmt.Errorf("data's TargetId did not match Target's actual ID")
		}
	case ksmsg.DataAck:
		dataAckPayload := ksMessage.KeysplittingPayload.(ksmsg.SynAckPayload)

		// Verify recieved hash pointer matches expected
		if dataAckPayload.HPointer != k.expectedHPointer {
			return fmt.Errorf("SynAck's hash pointer did not match expected")
		}
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

	case ksmsg.SynAck:
		synAckPayload := ksMessage.KeysplittingPayload.(ksmsg.SynAckPayload)
		if dataPayload, hash, err := synAckPayload.BuildResponsePayload(action, actionPayload); err != nil {
			return ksmsg.KeysplittingMessage{}, err
		} else {
			k.hPointer = hash
			responseMessage = ksmsg.KeysplittingMessage{
				Type:                ksmsg.Data,
				KeysplittingPayload: dataPayload,
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
	case ksmsg.DataAck:
		dataAckPayload := ksMessage.KeysplittingPayload.(ksmsg.DataAckPayload)
		if dataPayload, hash, err := dataAckPayload.BuildResponsePayload(action, actionPayload); err != nil {
			return ksmsg.KeysplittingMessage{}, err
		} else {
			k.hPointer = hash
			responseMessage = ksmsg.KeysplittingMessage{
				Type:                ksmsg.Data,
				KeysplittingPayload: dataPayload,
			}
		}
	}

	hashBytes, _ := util.HashPayload(responseMessage.KeysplittingPayload)
	k.expectedHPointer = base64.StdEncoding.EncodeToString(hashBytes)

	// Sign and send message
	// signed := responseMessage.Sign(privatekey)
	return responseMessage, nil
}

func (k *Keysplitting) BuildSyn(action string, payload []byte) (ksmsg.KeysplittingMessage, error) {
	// If this is the beginning of the hash chain, then we create a nonce with a random value,
	// otherwise we use the hash of the previous value to maintain the hash chain and immutability
	var nonce string
	if k.expectedHPointer == "" {
		nonce = util.Nonce()
	} else {
		nonce = k.expectedHPointer
	}

	// Build the BZ Certificate then store hash for future messages
	bzCert, err := k.BuildBZCert()
	if err != nil {
		return ksmsg.KeysplittingMessage{}, err
	} else {
		if hashBytes, ok := util.HashPayload(bzCert); !ok {
			return ksmsg.KeysplittingMessage{}, fmt.Errorf("could not hash BZ Certificate")
		} else {
			k.bzcertHash = base64.StdEncoding.EncodeToString(hashBytes)
		}
	}

	// Build the keysplitting message
	synPayload := ksmsg.SynPayload{
		Timestamp:     fmt.Sprint(time.Now().Unix()),
		SchemaVersion: schemaVersion,
		Type:          string(ksmsg.Syn),
		Action:        action,
		ActionPayload: payload,
		TargetId:      k.targetId, // TODO
		Nonce:         nonce,
		BZCert:        bzCert,
	}

	ksMessage := ksmsg.KeysplittingMessage{
		Type:                ksmsg.Syn,
		KeysplittingPayload: synPayload,
	}

	// Sign it and send it
	if err := ksMessage.Sign(k.privatekey); err != nil {
		return ksMessage, fmt.Errorf("could not sign payload: %v", err.Error())
	} else {
		return ksMessage, nil
	}
}

func (k *Keysplitting) BuildBZCert() (bzcrt.BZCert, error) {
	if configFile, err := os.Open(k.configPath); err != nil {
		return bzcrt.BZCert{}, fmt.Errorf("could not open config file: %v", err.Error())
	} else {
		configFileBytes, _ := ioutil.ReadAll(configFile)

		var config Config
		err := json.Unmarshal(configFileBytes, &config)
		if err != nil {
			return bzcrt.BZCert{}, fmt.Errorf("could not unmarshal config file")
		}

		// Set public and private keys because someone maybe have logged out and logged back in again
		k.publickey = config.KSConfig.PublicKey

		// The golang ed25519 library uses a length 64 private key because the private key is the concatenated form
		// privatekey = privatekey + publickey.  So if it was generated as length 32, we can correct for that here
		if privatekeyBytes, _ := base64.StdEncoding.DecodeString(config.KSConfig.PrivateKey); len(privatekeyBytes) == 32 {
			publickeyBytes, _ := base64.StdEncoding.DecodeString(k.publickey)
			k.privatekey = base64.StdEncoding.EncodeToString(append(privatekeyBytes, publickeyBytes...))
		} else {
			k.privatekey = config.KSConfig.PrivateKey
		}

		return bzcrt.BZCert{
			InitialIdToken:  config.KSConfig.InitialIdToken,
			CurrentIdToken:  config.TokenSet.CurrentIdToken,
			ClientPublicKey: config.KSConfig.PublicKey,
			Rand:            config.KSConfig.CerRand,
			SignatureOnRand: config.KSConfig.CerRandSignature,
		}, nil
	}
}
