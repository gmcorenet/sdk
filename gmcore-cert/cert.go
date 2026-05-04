package gmcore_cert

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"time"
)

type Certificate struct {
	*x509.Certificate
	PrivateKey interface{}
}

type CertificateManager struct {
	certs map[string]*Certificate
}

func NewManager() *CertificateManager {
	return &CertificateManager{certs: make(map[string]*Certificate)}
}

func (m *CertificateManager) Generate(host string, validDays int) (*Certificate, error) {
	priv, err := generatePrivateKey()
	if err != nil {
		return nil, err
	}

	serialNumber, err := generateSerialNumber()
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   host,
			Organization: []string{"GMCore"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, validDays),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, priv.Public(), priv)
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, err
	}

	return &Certificate{Certificate: cert, PrivateKey: priv}, nil
}

func (m *CertificateManager) Add(name string, cert *Certificate) {
	m.certs[name] = cert
}

func (m *CertificateManager) Get(name string) *Certificate {
	if cert, ok := m.certs[name]; ok {
		return cert
	}
	return nil
}

func (m *CertificateManager) Remove(name string) {
	delete(m.certs, name)
}

func (c *Certificate) EncodeCertificate() (string, error) {
	return encodePEM("CERTIFICATE", c.Raw)
}

func (c *Certificate) EncodePrivateKey() (string, error) {
	bytes, err := encodePrivateKey(c.PrivateKey)
	if err != nil {
		return "", err
	}
	return encodePEM("RSA PRIVATE KEY", bytes)
}

func (c *Certificate) SaveToFile(certFile, keyFile string) error {
	certPEM, err := c.EncodeCertificate()
	if err != nil {
		return err
	}
	keyPEM, err := c.EncodePrivateKey()
	if err != nil {
		return err
	}

	if err := os.WriteFile(certFile, []byte(certPEM), 0600); err != nil {
		return err
	}
	return os.WriteFile(keyFile, []byte(keyPEM), 0600)
}

func LoadCertificateFromFile(certFile, keyFile string) (*Certificate, error) {
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	return &Certificate{Certificate: cert}, nil
}

func encodePEM(label string, data []byte) (string, error) {
	var buf bytes.Buffer
	pem.Encode(&buf, &pem.Block{Type: label, Bytes: data})
	return buf.String(), nil
}

func generatePrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

func generateSerialNumber() (*big.Int, error) {
	serial := make([]byte, 16)
	if _, err := rand.Read(serial); err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(serial), nil
}

func encodePrivateKey(key interface{}) ([]byte, error) {
	if pk, ok := key.(*rsa.PrivateKey); ok {
		return pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(pk),
		}), nil
	}
	return nil, errors.New("unsupported key type")
}
