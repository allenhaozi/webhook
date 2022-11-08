package utils

import (
	"crypto"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
)

var (
	rsaKeySize           = 2048
	CertificateBlockType = "CERTIFICATE"
)

type CertContext struct {
	// server.crt
	Cert []byte
	// server.key
	Key        []byte
	SigningKey []byte
	// ca.crt
	SigningCert []byte
}

func GenerateCertAndCreate(namespaceName, serviceName, certDir string) {
	certContext := generateCert(namespaceName, serviceName)

	// ca.crt
	caCertFile := filepath.Join(certDir, "ca.crt")
	if err := os.WriteFile(caCertFile, certContext.SigningCert, 0o644); err != nil {
		log.Fatalf("Failed to write CA cert %v", err)
	}
	// server.key
	keyFile := filepath.Join(certDir, "tls.key")
	if err := os.WriteFile(keyFile, certContext.Key, 0o644); err != nil {
		log.Fatalf("Failed to write key file %v", err)
	}
	// server.csr
	certFile := filepath.Join(certDir, "tls.crt")
	if err := os.WriteFile(certFile, certContext.Cert, 0o600); err != nil {
		log.Fatalf("Failed to write cert file %v", err)
	}
}

// Reference: https://github.com/kubernetes/kubernetes/blob/v1.21.1/test/e2e/apimachinery/certs.go.
func generateCert(namespaceName, serviceName string) CertContext {
	signingKey, err := NewPrivateKey()
	if err != nil {
		log.Fatalf("Failed to create CA private key %v", err)
	}

	signingCert, err := cert.NewSelfSignedCACert(cert.Config{CommonName: "self-signed-k8s-cert"}, signingKey)
	if err != nil {
		log.Fatalf("Failed to create CA cert for apiserver %v", err)
	}

	key, err := NewPrivateKey()
	if err != nil {
		log.Fatalf("Failed to create private key for %v", err)
	}

	signedCert, err := NewSignedCert(
		&cert.Config{
			CommonName: serviceName + "." + namespaceName + ".svc",
			AltNames:   cert.AltNames{DNSNames: []string{serviceName + "." + namespaceName + ".svc"}},
			Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		},
		key,
		signingCert,
		signingKey,
	)
	if err != nil {
		log.Fatalf("Failed to create cert%v", err)
	}

	keyPEM, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		log.Fatalf("Failed to marshal key %v", err)
	}

	signingKeyPEM, err := keyutil.MarshalPrivateKeyToPEM(signingKey)
	if err != nil {
		log.Fatalf("Failed to marshal key %v", err)
	}

	c := CertContext{
		Cert:        EncodeCertPEM(signedCert),
		Key:         keyPEM,
		SigningCert: EncodeCertPEM(signingCert),
		SigningKey:  signingKeyPEM,
	}

	return c
}

// NewSignedCert creates a signed certificate using the given CA certificate and key
func NewSignedCert(cfg *cert.Config, key crypto.Signer, caCert *x509.Certificate, caKey crypto.Signer) (*x509.Certificate, error) {
	serial, err := cryptorand.Int(cryptorand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	if len(cfg.CommonName) == 0 {
		return nil, errors.New("must specify a CommonName")
	}
	if len(cfg.Usages) == 0 {
		return nil, errors.New("must specify at least one ExtKeyUsage")
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:     cfg.AltNames.DNSNames,
		IPAddresses:  cfg.AltNames.IPs,
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(time.Hour * 24 * 365 * 10).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  cfg.Usages,
	}
	certDERBytes, err := x509.CreateCertificate(cryptorand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

// NewPrivateKey creates an RSA private key
func NewPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(cryptorand.Reader, rsaKeySize)
}

// EncodeCertPEM returns PEM-endcoded certificate data
func EncodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  CertificateBlockType,
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}
