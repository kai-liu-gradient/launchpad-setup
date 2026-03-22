// Package secrets generates and manages cryptographic secrets for the deployment.
package secrets

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
)

// Secrets holds all cryptographic secrets needed by the deployment.
type Secrets struct {
	JWTSecret                      string `yaml:"jwt_secret"`
	JWTRefreshSecret               string `yaml:"jwt_refresh_secret"`
	SessionSecret                  string `yaml:"session_secret"`
	EncryptionKey                  string `yaml:"encryption_key"`
	ClaudeCredentialsEncryptionKey string `yaml:"claude_credentials_encryption_key"`
	SSHKeyEncryptionSecret         string `yaml:"ssh_key_encryption_secret"`
	RSAPublicKey                   string `yaml:"rsa_public_key"`
	RSAPrivateKey                  string `yaml:"rsa_private_key"`
	DBPasswordMain                 string `yaml:"db_password_main"`
	DBPasswordMonitoring           string `yaml:"db_password_monitoring"`
	DBPasswordEvents               string `yaml:"db_password_events"`
	DBPasswordBilling              string `yaml:"db_password_billing"`
	DBPasswordStats                string `yaml:"db_password_stats"`
	DBPasswordGateway              string `yaml:"db_password_gateway"`
	DBPasswordGitea                string `yaml:"db_password_gitea"`
	PostgresSuperuserPassword      string `yaml:"postgres_superuser_password"`
	RedisPassword                  string `yaml:"redis_password"`
	GatewayAPIKey                  string `yaml:"gateway_api_key"`
	InternalSecret                 string `yaml:"internal_secret"`
	AdminPassword                  string `yaml:"admin_password"`
}

// Generate creates a new set of cryptographic secrets using crypto/rand.
func Generate() (*Secrets, error) {
	rsaPub, rsaPriv, err := generateRSAKeyPair()
	if err != nil {
		return nil, fmt.Errorf("generating RSA keypair: %w", err)
	}

	s := &Secrets{}

	// 64-char hex secrets (32 random bytes -> 64 hex chars)
	hexFields := []*string{
		&s.JWTSecret,
		&s.JWTRefreshSecret,
		&s.SessionSecret,
		&s.InternalSecret,
	}
	for _, f := range hexFields {
		v, err := randHex(32)
		if err != nil {
			return nil, err
		}
		*f = v
	}

	// 32-char hex secrets (16 random bytes -> 32 hex chars)
	hex16Fields := []*string{
		&s.EncryptionKey,
		&s.ClaudeCredentialsEncryptionKey,
		&s.SSHKeyEncryptionSecret,
	}
	for _, f := range hex16Fields {
		v, err := randHex(16)
		if err != nil {
			return nil, err
		}
		*f = v
	}

	// RSA keys
	s.RSAPublicKey = rsaPub
	s.RSAPrivateKey = rsaPriv

	// 24-char URL-safe base64 passwords
	pwFields := []*string{
		&s.DBPasswordMain,
		&s.DBPasswordMonitoring,
		&s.DBPasswordEvents,
		&s.DBPasswordBilling,
		&s.DBPasswordStats,
		&s.DBPasswordGateway,
		&s.DBPasswordGitea,
		&s.PostgresSuperuserPassword,
		&s.RedisPassword,
		&s.GatewayAPIKey,
		&s.AdminPassword,
	}
	for _, f := range pwFields {
		v, err := randBase64Safe(24)
		if err != nil {
			return nil, err
		}
		*f = v
	}

	return s, nil
}

// randHex returns a hex-encoded string from n random bytes (length = 2*n).
func randHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

// randBase64Safe returns a URL-safe base64 string of exactly length chars
// (no padding). It generates enough random bytes and truncates to the
// requested length.
func randBase64Safe(length int) (string, error) {
	// Each base64 char encodes 6 bits; we need enough bytes.
	nBytes := (length*6 + 7) / 8
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}
	encoded := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(buf)
	return encoded[:length], nil
}

// generateRSAKeyPair generates a 2048-bit RSA keypair and returns the
// PEM-encoded public and private keys as base64-encoded strings.
func generateRSAKeyPair() (pub, priv string, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("generating RSA key: %w", err)
	}

	// Private key -> PKCS#8 PEM -> base64
	privDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", "", fmt.Errorf("marshaling private key: %w", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privDER,
	})
	priv = base64.StdEncoding.EncodeToString(privPEM)

	// Public key -> PKIX PEM -> base64
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("marshaling public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubDER,
	})
	pub = base64.StdEncoding.EncodeToString(pubPEM)

	return pub, priv, nil
}
