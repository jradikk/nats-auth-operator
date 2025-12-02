package jwt

import (
	"fmt"
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"

	natsv1alpha1 "github.com/jradikk/nats-auth-operator/api/v1alpha1"
)

// AccountManager manages NATS account JWT operations
type AccountManager struct {
	accountKP nkeys.KeyPair
}

// NewAccountManager creates a new account manager from an existing seed or generates a new one
func NewAccountManager(seed []byte) (*AccountManager, error) {
	var kp nkeys.KeyPair
	var err error

	if len(seed) > 0 {
		// Use existing seed
		kp, err = nkeys.FromSeed(seed)
		if err != nil {
			return nil, fmt.Errorf("failed to create keypair from seed: %w", err)
		}
	} else {
		// Generate new account keypair
		kp, err = nkeys.CreateAccount()
		if err != nil {
			return nil, fmt.Errorf("failed to create account keypair: %w", err)
		}
	}

	return &AccountManager{
		accountKP: kp,
	}, nil
}

// GetPublicKey returns the account's public key
func (am *AccountManager) GetPublicKey() (string, error) {
	return am.accountKP.PublicKey()
}

// GetSeed returns the account's seed (private key)
func (am *AccountManager) GetSeed() ([]byte, error) {
	return am.accountKP.Seed()
}

// GetKeyPair returns the account's keypair
func (am *AccountManager) GetKeyPair() nkeys.KeyPair {
	return am.accountKP
}

// CreateAccountClaims creates account claims from the spec
func (am *AccountManager) CreateAccountClaims(name, description string, limits *natsv1alpha1.AccountLimits) (*jwt.AccountClaims, error) {
	pubKey, err := am.accountKP.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	claims := jwt.NewAccountClaims(pubKey)
	claims.Name = name
	claims.Description = description
	claims.IssuedAt = time.Now().Unix()

	// Apply limits if specified
	if limits != nil {
		claims.Limits.Conn = limits.Conn
		claims.Limits.Subs = limits.Subs
		claims.Limits.Payload = limits.Payload
		claims.Limits.Data = limits.Data
		claims.Limits.Exports = limits.Exports
		claims.Limits.Imports = limits.Imports
		claims.Limits.WildcardExports = limits.WildcardExports

		// Apply JetStream limits if specified
		if limits.JetStream != nil {
			claims.Limits.MemoryStorage = limits.JetStream.MemoryStorage
			claims.Limits.DiskStorage = limits.JetStream.DiskStorage
			claims.Limits.Streams = limits.JetStream.Streams
			claims.Limits.Consumer = limits.JetStream.Consumer
			claims.Limits.MaxAckPending = limits.JetStream.MaxAckPending
			claims.Limits.MemoryMaxStreamBytes = limits.JetStream.MemoryMaxStreamBytes
			claims.Limits.DiskMaxStreamBytes = limits.JetStream.DiskMaxStreamBytes
			claims.Limits.MaxBytesRequired = limits.JetStream.MaxBytesRequired
		}
	}

	return claims, nil
}

// SignUserJWT signs a user JWT with the account key
func (am *AccountManager) SignUserJWT(userClaims *jwt.UserClaims) (string, error) {
	// Set the issuer to the account's public key
	pubKey, err := am.accountKP.PublicKey()
	if err != nil {
		return "", fmt.Errorf("failed to get account public key: %w", err)
	}
	userClaims.Issuer = pubKey

	// Sign the user JWT
	token, err := userClaims.Encode(am.accountKP)
	if err != nil {
		return "", fmt.Errorf("failed to encode user JWT: %w", err)
	}

	return token, nil
}
