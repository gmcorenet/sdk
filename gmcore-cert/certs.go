package gmcorecert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

type Config struct {
	HTTPSAddr          string
	PublicHost         string
	TLSMode            string
	LetsEncryptEmail   string
	LetsEncryptStaging bool
	DataDir            string
}

type Strategy interface {
	HTTPHandler(http.Handler) http.Handler
	TLSConfig() *tls.Config
	ListenAndServeTLS(*http.Server) error
}

type staticStrategy struct {
	certFile    string
	keyFile     string
	httpsSuffix string
}

type autoStrategy struct {
	manager     *autocert.Manager
	httpsSuffix string
}

func NewStrategy(cfg Config) (Strategy, error) {
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, err
	}

	liveCertDir := filepath.Join(cfg.DataDir, "certs", "live")
	liveCertFile := filepath.Join(liveCertDir, "tls.crt")
	liveKeyFile := filepath.Join(liveCertDir, "tls.key")

	tlsMode := strings.ToLower(cfg.TLSMode)
	isSelfSignedMode := tlsMode == "selfsigned" || (tlsMode == "auto" && !isACMEHost(strings.TrimSpace(cfg.PublicHost)))

	if valid, err := isReusableCertificate(liveCertFile, liveKeyFile, isSelfSignedMode); err == nil && valid {
		return staticStrategy{
			certFile:    liveCertFile,
			keyFile:     liveKeyFile,
			httpsSuffix: advertisedHTTPSSuffix(cfg.HTTPSAddr),
		}, nil
	}

	switch tlsMode {
	case "existing":
		return nil, fmt.Errorf("tls mode existing requires a valid certificate at %s", liveCertFile)
	case "selfsigned":
		return ensureSelfSignedStrategy(cfg, liveCertFile, liveKeyFile)
	case "auto":
		if host := strings.TrimSpace(cfg.PublicHost); isACMEHost(host) {
			return ensureACMEStrategy(cfg)
		}
		return ensureSelfSignedStrategy(cfg, liveCertFile, liveKeyFile)
	default:
		return nil, fmt.Errorf("unknown tls mode %q", cfg.TLSMode)
	}
}

func ensureACMEStrategy(cfg Config) (Strategy, error) {
	cacheName := "production"
	if cfg.LetsEncryptStaging {
		cacheName = "staging"
	}
	cacheDir := filepath.Join(cfg.DataDir, "certs", "autocert", cacheName)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, err
	}

	manager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache(cacheDir),
		HostPolicy: autocert.HostWhitelist(cfg.PublicHost),
		Email:      cfg.LetsEncryptEmail,
	}

	if cfg.LetsEncryptStaging {
		manager.Client = &acme.Client{DirectoryURL: "https://acme-staging-v02.api.letsencrypt.org/directory"}
	}

	return autoStrategy{
		manager:     manager,
		httpsSuffix: advertisedHTTPSSuffix(cfg.HTTPSAddr),
	}, nil
}

func ensureSelfSignedStrategy(cfg Config, liveCertFile, liveKeyFile string) (Strategy, error) {
	if err := os.MkdirAll(filepath.Dir(liveCertFile), 0o755); err != nil {
		return nil, err
	}
	if valid, err := isReusableCertificate(liveCertFile, liveKeyFile, true); err == nil && valid {
		return staticStrategy{
			certFile:    liveCertFile,
			keyFile:     liveKeyFile,
			httpsSuffix: advertisedHTTPSSuffix(cfg.HTTPSAddr),
		}, nil
	}
	if err := generateSelfSignedCertificate(cfg.PublicHost, liveCertFile, liveKeyFile); err != nil {
		return nil, err
	}
	return staticStrategy{
		certFile:    liveCertFile,
		keyFile:     liveKeyFile,
		httpsSuffix: advertisedHTTPSSuffix(cfg.HTTPSAddr),
	}, nil
}

func (s staticStrategy) HTTPHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil {
			target := "https://" + redirectHost(r.Host, s.httpsSuffix) + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusPermanentRedirect)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s staticStrategy) TLSConfig() *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12}
}

func (s staticStrategy) ListenAndServeTLS(server *http.Server) error {
	return server.ListenAndServeTLS(s.certFile, s.keyFile)
}

func (s autoStrategy) HTTPHandler(next http.Handler) http.Handler {
	return s.manager.HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := "https://" + redirectHost(r.Host, s.httpsSuffix) + r.URL.RequestURI()
		http.Redirect(w, r, target, http.StatusPermanentRedirect)
	}))
}

func (s autoStrategy) TLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion:     tls.VersionTLS12,
		NextProtos:     []string{"h2", "http/1.1", acme.ALPNProto},
		GetCertificate: s.manager.GetCertificate,
	}
}

func (s autoStrategy) ListenAndServeTLS(server *http.Server) error {
	return server.ListenAndServeTLS("", "")
}

func redirectHost(host, httpsSuffix string) string {
	if strings.Contains(host, ":") {
		name, _, err := net.SplitHostPort(host)
		if err == nil {
			return name + httpsSuffix
		}
	}
	return host + httpsSuffix
}

func advertisedHTTPSSuffix(httpsAddr string) string {
	if httpsAddr == "" {
		return ""
	}
	if strings.HasPrefix(httpsAddr, ":443") || httpsAddr == ":443" {
		return ""
	}
	if strings.HasPrefix(httpsAddr, ":") {
		return httpsAddr
	}
	_, port, err := net.SplitHostPort(httpsAddr)
	if err != nil || port == "443" {
		return ""
	}
	return ":" + port
}

func isIPAddress(host string) bool {
	return net.ParseIP(host) != nil
}

func isACMEHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" || host == "localhost" || isIPAddress(host) {
		return false
	}
	return strings.Contains(host, ".")
}

func isReusableCertificate(certFile, keyFile string, allowSelfSigned bool) (bool, error) {
	if _, err := os.Stat(certFile); err != nil {
		return false, err
	}
	if _, err := os.Stat(keyFile); err != nil {
		return false, err
	}

	pair, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return false, err
	}
	if len(pair.Certificate) == 0 {
		return false, errors.New("empty certificate chain")
	}

	cert, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return false, err
	}

	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return false, nil
	}
	if isSelfSigned(cert) && !allowSelfSigned {
		return false, nil
	}
	if isStagingCertificate(cert) {
		return false, nil
	}
	return true, nil
}

func isSelfSigned(cert *x509.Certificate) bool {
	return cert.Subject.String() == cert.Issuer.String()
}

func isStagingCertificate(cert *x509.Certificate) bool {
	value := strings.ToLower(cert.Issuer.String() + " " + cert.Subject.String())
	return strings.Contains(value, "staging")
}

func generateSelfSignedCertificate(host, certFile, keyFile string) error {
	if host == "" {
		host = "localhost"
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	serialNumber, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(90 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if ip := net.ParseIP(host); ip != nil {
		template.IPAddresses = []net.IP{ip}
	} else {
		template.DNSNames = []string{host}
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return err
	}

	certPEM := pemEncode("CERTIFICATE", der)
	keyPEM := pemEncodePKCS1(privateKey)

	if err := os.WriteFile(certFile, certPEM, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(keyFile, keyPEM, 0o600); err != nil {
		return err
	}
	return nil
}

func pemEncode(blockType string, der []byte) []byte {
	return []byte(fmt.Sprintf("-----BEGIN %s-----\n%s-----END %s-----\n", blockType, chunkBase64(der), blockType))
}

func pemEncodePKCS1(key *rsa.PrivateKey) []byte {
	return pemEncode("RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(key))
}

func chunkBase64(raw []byte) string {
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(raw)))
	base64.StdEncoding.Encode(encoded, raw)
	var builder strings.Builder
	for i := 0; i < len(encoded); i += 64 {
		end := i + 64
		if end > len(encoded) {
			end = len(encoded)
		}
		builder.Write(encoded[i:end])
		builder.WriteByte('\n')
	}
	return builder.String()
}
