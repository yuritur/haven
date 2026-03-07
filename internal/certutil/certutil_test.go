package certutil_test

import (
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/havenapp/haven/internal/certutil"
)

func TestGenerateSelfSigned(t *testing.T) {
	certPEM, keyPEM, fingerprint, err := certutil.GenerateSelfSigned()
	if err != nil {
		t.Fatalf("GenerateSelfSigned returned error: %v", err)
	}

	if certPEM == "" {
		t.Error("certPEM is empty")
	}
	if keyPEM == "" {
		t.Error("keyPEM is empty")
	}
	if fingerprint == "" {
		t.Error("fingerprint is empty")
	}

	if _, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM)); err != nil {
		t.Fatalf("key pair not valid: %v", err)
	}

	if len(fingerprint) != 64 {
		t.Errorf("fingerprint length = %d, want 64", len(fingerprint))
	}

	_, _, fingerprint2, err := certutil.GenerateSelfSigned()
	if err != nil {
		t.Fatalf("second GenerateSelfSigned returned error: %v", err)
	}
	if fingerprint == fingerprint2 {
		t.Error("two calls produced identical fingerprints")
	}
}

func TestNewPinnedTransport(t *testing.T) {
	certPEM, keyPEM, fp, err := certutil.GenerateSelfSigned()
	if err != nil {
		t.Fatalf("GenerateSelfSigned: %v", err)
	}

	tlsCert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		t.Fatalf("X509KeyPair: %v", err)
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{tlsCert}}
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.StartTLS()
	defer server.Close()

	// Correct fingerprint — expect 200.
	client := &http.Client{Transport: certutil.NewPinnedTransport(fp)}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("request with correct fingerprint failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	// Wrong fingerprint — expect mismatch error.
	wrongFP := strings.Repeat("a", 64)
	badClient := &http.Client{Transport: certutil.NewPinnedTransport(wrongFP)}
	_, err = badClient.Get(server.URL)
	if err == nil {
		t.Fatal("request with wrong fingerprint succeeded, expected error")
	}
	if !strings.Contains(err.Error(), "cert fingerprint mismatch") {
		t.Errorf("error = %q, want it to contain \"cert fingerprint mismatch\"", err.Error())
	}
}
