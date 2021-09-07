package bzcert

import (
	"encoding/base64"
	"fmt"
	"time"

	"bastionzero.com/bctl/v1/bzerolib/keysplitting/util"
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

// This function verifies the user's bzcert.  We pass in the user's SSO provider (idpProvider) and
// their org id (e.g. called 'org' in Google jwts and 'tenantId' for microsoft) which specifies a particular
// organization/company/group that hosts their SSO as part of a larger SSO.
// The function returns the hash the bzcert, the expiration time of the bzcert, and an error if there is one
func (b *BZCert) Verify(idpProvider string, idpOrgId string) (string, time.Time, error) {
	verifier := NewBZCertVerifier(b, idpProvider, idpOrgId)

	if _, err := verifier.VerifyIdToken(b.InitialIdToken, true, true); err != nil {
		return "", time.Time{}, err
	}
	if exp, err := verifier.VerifyIdToken(b.CurrentIdToken, false, false); err != nil {
		return "", time.Time{}, err
	} else {
		if hash, ok := b.Hash(); ok {
			return hash, exp, err
		} else {
			return "", time.Time{}, fmt.Errorf("failed to hash BZCert")
		}
	}
}

func (b *BZCert) Hash() (string, bool) {
	if hashBytes, ok := util.HashPayload((*b)); ok {
		return base64.StdEncoding.EncodeToString(hashBytes), ok
	} else {
		return "", ok
	}
}
