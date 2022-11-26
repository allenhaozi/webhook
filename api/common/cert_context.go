package common

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

var (
	cacrt  = "ca.crt"
	tlskey = "tls.key"
	tlscrt = "tls.crt"
)

type CertContext struct {
	// tls.crt,server.crt
	Cert []byte
	// tls.key,server.key
	Key []byte

	SigningKey []byte
	// ca.crt
	SigningCert []byte
}

func (c *CertContext) WriteCertFileToLocal(certDir string) error {
	err := os.MkdirAll(certDir, 0o700)
	if err != nil {
		return errors.Wrapf(err, "failed to mkdir certificate dir:%s", certDir)
	}

	// ca.crt
	caCertFile := filepath.Join(certDir, "ca.crt")
	if err := os.WriteFile(caCertFile, c.SigningCert, 0o644); err != nil {
		return errors.Wrapf(err, "Failed to write file:%s", caCertFile)
	}
	// server.key
	keyFile := filepath.Join(certDir, "tls.key")
	if err := os.WriteFile(keyFile, c.Key, 0o644); err != nil {
		return errors.Wrapf(err, "Failed to write file:%s", keyFile)
	}
	// server.csr
	certFile := filepath.Join(certDir, "tls.crt")
	if err := os.WriteFile(certFile, c.Cert, 0o600); err != nil {
		return errors.Wrapf(err, "Failed to write file:%s", certFile)
	}

	return nil
}

func (c *CertContext) ComposeSecrets(namespace, name string) *corev1.Secret {
	s := &corev1.Secret{}
	s.Name = name
	s.Namespace = namespace
	s.Type = corev1.SecretTypeOpaque
	s.Data = map[string][]byte{
		cacrt:  c.SigningCert,
		tlskey: c.Key,
		tlscrt: c.Cert,
	}

	return s
}

func GenerateCertBySecret(s *corev1.Secret) (*CertContext, error) {
	c := &CertContext{}
	if v, ok := s.Data[cacrt]; ok {
		c.SigningCert = v
	} else {
		return nil, errors.Errorf("%s not found", cacrt)
	}

	if v, ok := s.Data[tlskey]; ok {
		c.Key = v
	} else {
		return nil, errors.Errorf("%s not found", tlskey)
	}

	if v, ok := s.Data[tlscrt]; ok {
		c.Cert = v
	} else {
		return nil, errors.Errorf("%s not found", tlscrt)
	}
	return c, nil
}
