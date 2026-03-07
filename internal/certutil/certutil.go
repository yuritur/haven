package certutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"time"
)

// GenerateSelfSigned creates an ECDSA P-256 self-signed certificate valid for 10 years.
// Returns PEM-encoded cert, PEM-encoded private key, and SHA-256 fingerprint as hex string.
func GenerateSelfSigned() (certPEM, keyPEM, fingerprint string, err error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", "", fmt.Errorf("generate key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", "", fmt.Errorf("generate serial: %w", err)
	}

	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "haven"},
		NotBefore:    now,
		NotAfter:     now.AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyAgreement,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return "", "", "", fmt.Errorf("create certificate: %w", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", "", "", fmt.Errorf("marshal key: %w", err)
	}

	certPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
	keyPEM = string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))

	sum := sha256.Sum256(certDER)
	fingerprint = hex.EncodeToString(sum[:])

	return certPEM, keyPEM, fingerprint, nil
}

// NewPinnedTransport returns an *http.Transport that accepts self-signed certs but
// enforces that the server presents a cert matching the expected SHA-256 fingerprint.
// This provides TOFU-style MITM protection without a CA chain.
func NewPinnedTransport(fingerprint string) *http.Transport {
	return &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // fingerprint pinning replaces CA verification
			MinVersion:         tls.VersionTLS12,
			VerifyConnection: func(cs tls.ConnectionState) error {
				for _, cert := range cs.PeerCertificates {
					sum := sha256.Sum256(cert.Raw)
					if hex.EncodeToString(sum[:]) == fingerprint {
						return nil
					}
				}
				return fmt.Errorf("certutil: no presented certificate matches pinned fingerprint %s", fingerprint)
			},
		},
	}
}
