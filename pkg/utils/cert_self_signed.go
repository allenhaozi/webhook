package utils

import (
	"crypto"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math"
	"math/big"
	"time"

	"github.com/pkg/errors"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"

	"github.com/allenhaozi/webhook/api/common"
)

var (
	rsaKeySize           = 2048
	CertificateBlockType = "CERTIFICATE"
)

// reference: https://github.com/kubernetes/kubernetes/blob/v1.21.1/test/e2e/apimachinery/certs.go.
func GenerateCert(namespaceName, serviceName string) (*common.CertContext, error) {
	signingKey, err := NewPrivateKey()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create CA private key")
	}

	signingCert, err := cert.NewSelfSignedCACert(cert.Config{CommonName: "self-signed-k8s-cert"}, signingKey)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create CA cert for Apiserver")
	}

	key, err := NewPrivateKey()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create private key")
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
		return nil, errors.Wrap(err, "Failed to create signed certificate")
	}

	keyPEM, err := keyutil.MarshalPrivateKeyToPEM(key)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to marshal private key")
	}

	signingKeyPEM, err := keyutil.MarshalPrivateKeyToPEM(signingKey)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to marshal signed key")
	}

	c := &common.CertContext{
		Cert:        EncodeCertPEM(signedCert),
		Key:         keyPEM,
		SigningCert: EncodeCertPEM(signingCert),
		SigningKey:  signingKeyPEM,
	}

	return c, nil
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
