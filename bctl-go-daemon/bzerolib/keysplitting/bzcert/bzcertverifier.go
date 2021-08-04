package bzcert

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	ed "crypto/ed25519"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/crypto/sha3"
)

const (
	googleUrl    = "https://accounts.google.com"
	microsoftUrl = "https://login.microsoftonline.com"
	// this is the tenant id Microsoft uses when the account is a personal account (not a work/school account)
	// https://docs.microsoft.com/en-us/azure/active-directory/develop/id-tokens#payload-claims)
	microsoftPersonalAccountTenantId = "9188040d-6c67-4c5b-b112-36a304b66dad"

	bzCustomTokenLifetime = time.Hour * 24 * 365 * 5 // 5 years
)

type IBZCertVerifier interface {
	VerifyIdToken(idtoken string, skipExpiry bool, verifyNonce bool) (time.Time, error)
}

type BZCertVerifier struct {
	orgId       string
	orgProvider ProviderType
	iss         string
	cert        *BZCert
}

type ProviderType string

const (
	Google    ProviderType = "google"
	Microsoft ProviderType = "microsoft"
	Custom    ProviderType = "custom"
)

func NewBZCertVerifier(bzcert *BZCert) IBZCertVerifier {
	orgId := "placeholder"             // eventually load these from vault
	provider := ProviderType("google") // eventually load these from vault
	customIss := "https://test"        // add this variable and load it from vault

	iss := ""
	switch provider {
	case Google:
		iss = googleUrl
	case Microsoft:
		iss = getMicrosoftIssuerUrl(orgId)
	case Custom:
		iss = customIss // Any valid iss requires a discovery document
	default:
		return &BZCertVerifier{} // return error
	}

	return &BZCertVerifier{
		orgId:       orgId,
		orgProvider: provider,
		iss:         iss,
		cert:        bzcert,
	}
}

func getMicrosoftIssuerUrl(orgId string) string {
	// Handles personal accounts by using microsoftPersonalAccountTenantId as the tenantId
	// see https://github.com/coreos/go-oidc/issues/121
	tenantId := ""
	if orgId == "None" {
		tenantId = microsoftPersonalAccountTenantId
	} else {
		tenantId = orgId
	}

	return microsoftUrl + "/" + tenantId + "/v2.0"
}

// This function verifies id_tokens
func (u *BZCertVerifier) VerifyIdToken(idtoken string, skipExpiry bool, verifyNonce bool) (time.Time, error) {
	// Verify Token Signature

	ctx := context.TODO() // Gives us non-nil empty context
	config := &oidc.Config{
		SkipClientIDCheck: true,
		SkipExpiryCheck:   skipExpiry,
		// SupportedSigningAlgs: []string{RS256, ES512},
	}

	provider, err := oidc.NewProvider(ctx, u.iss)
	if err != nil {
		return time.Time{}, fmt.Errorf("Error establishing OIDC provider during validation: %v", err)
	}

	// This checks formatting and signature validity
	verifier := provider.Verifier(config)
	token, err := verifier.Verify(ctx, idtoken)
	if err != nil {
		return time.Time{}, fmt.Errorf("ID Token verification error: %v", err)
	}

	// Verify Claims

	// the claims we care about checking
	var claims struct {
		HD       string `json:"hd"`    // Google Org ID
		Nonce    string `json:"nonce"` // Bastion Zero issued nonce
		TID      string `json:"tid"`   // Microsoft Tenant ID
		IssuedAt int64  `json:"iat"`   // Unix datetime of issuance
		Death    int64  `json:"exp"`   // Unix datetime of token expiry
	}

	if err := token.Claims(&claims); err != nil {
		return time.Time{}, fmt.Errorf("Error parsing the ID Token: %v", err)
	} else {
		// k.log.Infof("ID Token claims: {HD: %s, Nonce: %s, Org: %s}", claims.HD, claims.Nonce, claims.TID)
		// k.log.Infof("Agent Org Info: {orgID: %s, orgProvider: %s}", k.orgId, k.provider)
	}

	// Manual check to see if InitialIdToken is expired
	if skipExpiry {
		now := time.Now()
		iat := time.Unix(claims.IssuedAt, 0) // Confirmed both Microsoft and Google use Unix
		if now.After(iat.Add(bzCustomTokenLifetime)) {
			return time.Time{}, fmt.Errorf("InitialIdToken Expired {Current Time = %v, Token iat = %v}", now, iat)
		}
	}

	// Check if Nonce in ID token is formatted correctly
	if verifyNonce {
		if err = u.verifyAuthNonce(claims.Nonce); err != nil {
			return time.Time{}, err
		}
	}

	// Only validate org claim if there is an orgId associated with this agent.
	// This will be empty for orgs associated with a personal gsuite/microsoft account
	switch u.orgProvider {
	case Google:
		if u.orgId != claims.HD {
			return time.Time{}, fmt.Errorf("User's OrgId does not match target's expected Google HD")
		}
	case Microsoft:
		if u.orgId != claims.TID {
			return time.Time{}, fmt.Errorf("User's OrgId does not match target's expected Microsoft tid")
		}
	}

	return time.Unix(claims.Death, 0), nil
}

// This function takes in the BZECert, extracts all fields for verifying the AuthNonce (sent as
//  part of the ID Token).  Returns nil if nonce is verified, else returns an error of type KeysplittingError
func (b *BZCertVerifier) verifyAuthNonce(authNonce string) error {
	nonce := b.cert.ClientPublicKey + b.cert.SignatureOnRand + b.cert.Rand
	hash := sha3.Sum256([]byte(nonce))
	nonceHash := base64.StdEncoding.EncodeToString(hash[:])

	// check nonce is equal to what is expected
	if authNonce != nonceHash {
		return fmt.Errorf("Nonce in ID token does not match calculated nonce hash")
	}

	decodedRand, err := base64.StdEncoding.DecodeString(b.cert.Rand)
	if err != nil {
		return fmt.Errorf("BZCert Rand is not base64 encoded")
	}

	randHashBits := sha3.Sum256([]byte(decodedRand))
	sigBits, _ := base64.StdEncoding.DecodeString(b.cert.SignatureOnRand)

	pubKeyBits, _ := base64.StdEncoding.DecodeString(b.cert.ClientPublicKey)
	if len(pubKeyBits) != 32 {
		return fmt.Errorf("Public Key has invalid length %v", len(pubKeyBits))
	}
	pubkey := ed.PublicKey(pubKeyBits)

	if ok := ed.Verify(pubkey, randHashBits[:], sigBits); ok {
		return nil
	} else {
		return fmt.Errorf("Failed to verify signature on rand")
	}
}
