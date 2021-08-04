package bzcert

import (
	"encoding/base64"
	"fmt"
	"time"

	"bastionzero.com/bctl/v1/bzerolib/keysplitting/hasher"
)

const (
	bzecertLifetime = time.Hour * 24 * 365 * 5 // 5 years
)

type IBZCert interface {
	Verify() (string, time.Time, error)
	Hash() (string, bool)
}

type BZCert struct {
	InitialIdToken  string `json:"initialIdToken"`
	CurrentIdToken  string `json:"currentIdToken"`
	ClientPublicKey string `json:"clientPublicKey"`
	Rand            string `json:"rand"`
	SignatureOnRand string `json:"signatureOnRand"`
}

type BZCertMetadata struct {
	Cert   BZCert
	Expiry time.Time
}

func (b *BZCert) Verify() (string, time.Time, error) {
	verifier := NewBZCertVerifier(b)
	//verifier.VerifyIdToken(b.InitialIdToken, b, true, true)

	if _, err := verifier.VerifyIdToken(b.InitialIdToken, true, true); err != nil {
		return "", time.Time{}, err
	}
	if exp, err := verifier.VerifyIdToken(b.CurrentIdToken, false, false); err != nil {
		return "", time.Time{}, err
	} else {
		if hash, ok := b.Hash(); ok {
			return hash, exp, err
		} else {
			return "", time.Time{}, fmt.Errorf("Failed to hash BZCert")
		}
	}
}

func (b *BZCert) Hash() (string, bool) {
	if hashBytes, ok := hasher.HashPayload((*b)); ok {
		return base64.StdEncoding.EncodeToString(hashBytes), ok
	} else {
		return "", ok
	}
}
