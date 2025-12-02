package jwt

import (
	"testing"

	"github.com/nats-io/nkeys"
)

func TestNewOperatorManager(t *testing.T) {
	tests := []struct {
		name         string
		seed         []byte
		operatorName string
		wantErr      bool
	}{
		{
			name:         "Create new operator",
			seed:         nil,
			operatorName: "Test Operator",
			wantErr:      false,
		},
		{
			name:         "Create from existing seed",
			seed:         generateTestOperatorSeed(t),
			operatorName: "Test Operator",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			om, err := NewOperatorManager(tt.seed, tt.operatorName)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOperatorManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if om == nil {
					t.Error("NewOperatorManager() returned nil")
					return
				}

				// Verify we can get the public key
				pubKey, err := om.GetPublicKey()
				if err != nil {
					t.Errorf("GetPublicKey() error = %v", err)
				}
				if pubKey == "" {
					t.Error("GetPublicKey() returned empty string")
				}

				// Verify we can get the JWT
				jwt := om.GetJWT()
				if jwt == "" {
					t.Error("GetJWT() returned empty string")
				}

				// Verify we can get the seed
				seed, err := om.GetSeed()
				if err != nil {
					t.Errorf("GetSeed() error = %v", err)
				}
				if len(seed) == 0 {
					t.Error("GetSeed() returned empty seed")
				}
			}
		})
	}
}

func TestOperatorManager_SignAccountJWT(t *testing.T) {
	om, err := NewOperatorManager(nil, "Test Operator")
	if err != nil {
		t.Fatalf("Failed to create operator manager: %v", err)
	}

	// Create a test account manager
	am, err := NewAccountManager(nil)
	if err != nil {
		t.Fatalf("Failed to create account manager: %v", err)
	}

	// Create account claims
	claims, err := am.CreateAccountClaims("Test Account", "Test Description", nil)
	if err != nil {
		t.Fatalf("Failed to create account claims: %v", err)
	}

	// Sign the account JWT
	jwt, err := om.SignAccountJWT(claims)
	if err != nil {
		t.Errorf("SignAccountJWT() error = %v", err)
	}
	if jwt == "" {
		t.Error("SignAccountJWT() returned empty JWT")
	}
}

func generateTestOperatorSeed(t *testing.T) []byte {
	kp, err := nkeys.CreateOperator()
	if err != nil {
		t.Fatalf("Failed to create test operator keypair: %v", err)
	}
	seed, err := kp.Seed()
	if err != nil {
		t.Fatalf("Failed to get seed: %v", err)
	}
	return seed
}
