package connector

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"os"
	"time"
)

// Identities are exact URI SANs, never Common Names or caller-supplied headers.
func Identity(tenant, role string) string {
	return "spiffe://tsunami.lab/tenants/" + tenant + "/" + role
}

func ConnectorIdentity(tenant, audience string) string {
	return Identity(tenant, "connectors/"+audience)
}

func peerIdentity(cert *x509.Certificate, expected ...string) bool {
	if len(cert.URIs) != 1 {
		return false
	}
	for _, value := range expected {
		if cert.URIs[0].String() == value {
			return true
		}
	}
	return false
}

func loadTLSMaterial(caFile, certFile, keyFile string) (*x509.CertPool, tls.Certificate, error) {
	data, err := os.ReadFile(caFile)
	if err != nil {
		return nil, tls.Certificate{}, errors.New("connector CA file unavailable")
	}
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(data) {
		return nil, tls.Certificate{}, errors.New("invalid connector CA")
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, tls.Certificate{}, errors.New("invalid connector TLS certificate/key")
	}
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, tls.Certificate{}, errors.New("invalid connector leaf certificate")
	}
	return roots, cert, nil
}

func LoadClientTLS(caFile, certFile, keyFile, serverName, tenant, audience, role string) (*tls.Config, error) {
	roots, cert, err := loadTLSMaterial(caFile, certFile, keyFile)
	if err != nil {
		return nil, err
	}
	if serverName == "" || (role != "broker" && role != "emergency-operator") || !peerIdentity(cert.Leaf, Identity(tenant, role)) {
		return nil, errors.New("incorrect connector client identity")
	}
	if _, err := cert.Leaf.Verify(x509.VerifyOptions{Roots: roots, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); err != nil {
		return nil, errors.New("connector client certificate is not currently valid")
	}
	return &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: roots, Certificates: []tls.Certificate{cert}, ServerName: serverName, VerifyConnection: func(cs tls.ConnectionState) error {
		// Standard CA, hostname, validity, and EKU checks run before this hook.
		if len(cs.VerifiedChains) == 0 || !peerIdentity(cs.PeerCertificates[0], ConnectorIdentity(tenant, audience)) {
			return errors.New("unexpected connector server identity")
		}
		return nil
	}}, nil
}

func LoadServerTLS(caFile, certFile, keyFile, serverName, tenant, audience string) (*tls.Config, time.Time, error) {
	roots, cert, err := loadTLSMaterial(caFile, certFile, keyFile)
	if err != nil {
		return nil, time.Time{}, err
	}
	if !peerIdentity(cert.Leaf, ConnectorIdentity(tenant, audience)) {
		return nil, time.Time{}, errors.New("incorrect connector server identity")
	}
	chains, err := cert.Leaf.Verify(x509.VerifyOptions{Roots: roots, DNSName: serverName, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}})
	if err != nil {
		return nil, time.Time{}, errors.New("connector server certificate is not currently valid")
	}
	expires := cert.Leaf.NotAfter
	for _, c := range chains[0] {
		if c.NotAfter.Before(expires) {
			expires = c.NotAfter
		}
	}
	return &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cert}, ClientCAs: roots, ClientAuth: tls.RequireAndVerifyClientCert, SessionTicketsDisabled: true, VerifyConnection: func(cs tls.ConnectionState) error {
		if len(cs.VerifiedChains) == 0 || !peerIdentity(cs.PeerCertificates[0], Identity(tenant, "broker"), Identity(tenant, "emergency-operator")) {
			return errors.New("unexpected connector client identity")
		}
		return nil
	}}, expires, nil
}

// TLS does not automatically recheck expiry for every HTTP request or cancel
// an open response body. Check again and clamp execution to both peer and own
// certificate validity. Signed grant expiry remains an independent boundary.
func (s *Server) transportPermission(r *http.Request) (time.Time, error) {
	if r.TLS == nil || r.TLS.Version < tls.VersionTLS13 || len(r.TLS.VerifiedChains) == 0 || s.TLSNotAfter.IsZero() {
		return time.Time{}, errors.New("authenticated connector TLS required")
	}
	role := "broker"
	if r.URL.Path == "/v1/cutoff" || r.URL.Path == "/v1/status" {
		role = "emergency-operator"
	}
	if !peerIdentity(r.TLS.PeerCertificates[0], Identity(s.Tenant, role)) {
		return time.Time{}, errors.New("connector client role denied")
	}
	now := time.Now()
	expires := s.TLSNotAfter
	for _, cert := range r.TLS.VerifiedChains[0] {
		if now.Before(cert.NotBefore) || !now.Before(cert.NotAfter) {
			return time.Time{}, errors.New("connector client certificate expired")
		}
		if cert.NotAfter.Before(expires) {
			expires = cert.NotAfter
		}
	}
	if !now.Before(expires) {
		return time.Time{}, errors.New("connector server certificate expired")
	}
	return expires, nil
}
