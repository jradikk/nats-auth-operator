package jwt

import (
	"fmt"
	"time"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"

	natsv1alpha1 "github.com/jradikk/nats-auth-operator/api/v1alpha1"
)

// UserManager manages NATS user JWT operations
type UserManager struct {
	userKP nkeys.KeyPair
}

// NewUserManager creates a new user manager from an existing seed or generates a new one
func NewUserManager(seed []byte) (*UserManager, error) {
	var kp nkeys.KeyPair
	var err error

	if len(seed) > 0 {
		// Use existing seed
		kp, err = nkeys.FromSeed(seed)
		if err != nil {
			return nil, fmt.Errorf("failed to create keypair from seed: %w", err)
		}
	} else {
		// Generate new user keypair
		kp, err = nkeys.CreateUser()
		if err != nil {
			return nil, fmt.Errorf("failed to create user keypair: %w", err)
		}
	}

	return &UserManager{
		userKP: kp,
	}, nil
}

// GetPublicKey returns the user's public key
func (um *UserManager) GetPublicKey() (string, error) {
	return um.userKP.PublicKey()
}

// GetSeed returns the user's seed (private key)
func (um *UserManager) GetSeed() ([]byte, error) {
	return um.userKP.Seed()
}

// GetKeyPair returns the user's keypair
func (um *UserManager) GetKeyPair() nkeys.KeyPair {
	return um.userKP
}

// CreateUserClaims creates user claims from the spec
func (um *UserManager) CreateUserClaims(name string, permissions *natsv1alpha1.Permissions) (*jwt.UserClaims, error) {
	pubKey, err := um.userKP.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	claims := jwt.NewUserClaims(pubKey)
	claims.Name = name
	claims.IssuedAt = time.Now().Unix()

	// Apply permissions if specified
	if permissions != nil {
		if len(permissions.PublishAllow) > 0 {
			claims.Pub.Allow.Add(permissions.PublishAllow...)
		}
		if len(permissions.PublishDeny) > 0 {
			claims.Pub.Deny.Add(permissions.PublishDeny...)
		}
		if len(permissions.SubscribeAllow) > 0 {
			claims.Sub.Allow.Add(permissions.SubscribeAllow...)
		}
		if len(permissions.SubscribeDeny) > 0 {
			claims.Sub.Deny.Add(permissions.SubscribeDeny...)
		}
	}

	return claims, nil
}

// GenerateCredsFile generates a NATS credentials file content
func GenerateCredsFile(userJWT string, userSeed []byte) string {
	return fmt.Sprintf(`-----BEGIN NATS USER JWT-----
%s
------END NATS USER JWT------

************************* IMPORTANT *************************
NKEY Seed printed below can be used to sign and prove identity.
NKEYs are sensitive and should be treated as secrets.

-----BEGIN USER NKEY SEED-----
%s
------END USER NKEY SEED------

*************************************************************
`, userJWT, string(userSeed))
}
