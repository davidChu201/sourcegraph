// Package idkey deals with Sourcegarph identity keys (which identify
// a Sourcegraph instance or cluster).
package idkey

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/jwt"
)

// bits is the size (in bits) of RSA keys generated by Generate.
var bits = getPrivKeyBits()

func getPrivKeyBits() int {
	s := strings.TrimSpace(os.Getenv("ID_KEY_SIZE"))
	if s == "" {
		return 2048 // default
	}
	var err error
	n, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("Invalid ID_KEY_SIZE: %s.", err)
	}
	if n < 1024 {
		log.Fatalf("ID_KEY_SIZE must be at least 1024 (got %d).", n)
	}
	return n
}

// IDKey holds a Sourcegraph identity key (which identifies a
// Sourcegraph instance or cluster).
type IDKey struct {
	key      *rsa.PrivateKey
	pemBytes []byte

	// ID is k's public key fingerprint, which can act as a client's
	// or server's identity.
	ID string
}

// Generate generates a new Sourcegraph identity key.
func Generate() (*IDKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	idkey := &IDKey{key: key}
	if err := idkey.precompute(); err != nil {
		return nil, err
	}
	return idkey, nil
}

// New creates a new Sourcegraph identity key from PEM-encoded bytes
// of the form:
//
//  -----BEGIN RSA PRIVATE KEY-----
//  ...
//  -----END RSA PRIVATE KEY-----
func New(pem []byte) (*IDKey, error) {
	var k IDKey
	if err := k.UnmarshalText(pem); err != nil {
		return nil, err
	}
	return &k, nil
}

// FromString creates a new Sourcegraph identity key from a PEM-encoded
// string. It allows encoding the PEM data in base64, to make it easier to
// pass in env vars (which are often serialized/deserialized via buggy
// bash scripts).
func FromString(idKeyData string) (*IDKey, error) {
	if strings.HasPrefix(idKeyData, "base64:") {
		idKeyData = strings.TrimPrefix(idKeyData, "base64:")
		b, err := base64.StdEncoding.DecodeString(idKeyData)
		if err != nil {
			return nil, err
		}
		idKeyData = string(b)
	}
	return New([]byte(idKeyData))
}

// Private returns k's private key.
func (k *IDKey) Private() *rsa.PrivateKey { return k.key }

// Public returns k's public key.
func (k *IDKey) Public() *rsa.PublicKey { return k.key.Public().(*rsa.PublicKey) }

func (k *IDKey) MarshalText() ([]byte, error) {
	return k.pemBytes, nil
}

func (k *IDKey) UnmarshalText(data []byte) error {
	privBlock, _ := pem.Decode(data)
	if privBlock == nil {
		return errors.New("invalid private key PEM-encoded data")
	}
	if privBlock.Type != "RSA PRIVATE KEY" {
		return fmt.Errorf("invalid private key block type %q", privBlock.Type)
	}
	privKey, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	if err != nil {
		return err
	}
	k.key = privKey
	return k.precompute()
}

func (k *IDKey) precompute() error {
	k.key.Precompute()

	// Precompute bytes.
	k.pemBytes = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k.key)})

	fp, err := Fingerprint(k.Public())
	if err != nil {
		return err
	}
	k.ID = fp
	return nil
}

// Fingerprint returns the fingerprint used as the ID (generated from
// the ID key's public key).
func Fingerprint(pubKey crypto.PublicKey) (string, error) {
	// Precompute ID.
	b, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return base64.StdEncoding.EncodeToString(sum[:]), nil
}

var testEnvOnce sync.Once

// SetTestEnvironment adjusts the configuration for use in non-production
// environments. This is so non-production environments can run faster. This
// should never be called in production
func SetTestEnvironment(size int) {
	testEnvOnce.Do(func() {
		bits = size
	})
}

func (k *IDKey) TokenSource(ctx context.Context, tokenURL string) oauth2.TokenSource {
	c := &jwt.Config{
		Email:      k.ID,
		Subject:    k.ID,
		PrivateKey: k.pemBytes,
		TokenURL:   tokenURL,
	}
	return c.TokenSource(ctx)
}

type contextKey int

const (
	idKeyKey contextKey = iota
)

// FromContext returns the Sourcegraph identity key from the context,
// or nil if none is set.
func FromContext(ctx context.Context) *IDKey {
	idkey, _ := ctx.Value(idKeyKey).(*IDKey)
	return idkey
}

// NewContext returns a child context with the given ID key.
func NewContext(ctx context.Context, idkey *IDKey) context.Context {
	return context.WithValue(ctx, idKeyKey, idkey)
}
