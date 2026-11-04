package connector

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const LabTenant = "demo-a"
const LabAudience = "customer-a"
const LabServerName = "tsunami-connector.tsunami-customer-a.svc.cluster.local"

type authority struct {
	cert    *x509.Certificate
	key     ed25519.PrivateKey
	certPEM []byte
}

func serial() (*big.Int, error) { return rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128)) }

func newAuthority(now time.Time) (*authority, error) {
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	n, err := serial()
	if err != nil {
		return nil, err
	}
	t := &x509.Certificate{SerialNumber: n, Subject: pkix.Name{CommonName: "Tsunami Customer A lab CA"}, NotBefore: now.Add(-time.Minute), NotAfter: now.Add(30 * 24 * time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign, MaxPathLenZero: true}
	der, err := x509.CreateCertificate(rand.Reader, t, t, pub, key)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	return &authority{cert, key, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})}, nil
}

func privatePEM(key ed25519.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil
}

func (a *authority) issue(identity string, names []string, usage x509.ExtKeyUsage, start, end time.Time) ([]byte, []byte, error) {
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	n, err := serial()
	if err != nil {
		return nil, nil, err
	}
	u, err := url.Parse(identity)
	if err != nil {
		return nil, nil, err
	}
	if end.After(a.cert.NotAfter) {
		return nil, nil, errors.New("leaf validity exceeds CA validity")
	}
	t := &x509.Certificate{SerialNumber: n, NotBefore: start, NotAfter: end, URIs: []*url.URL{u}, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{usage}, BasicConstraintsValid: true}
	for _, name := range names {
		if ip := net.ParseIP(name); ip != nil {
			t.IPAddresses = append(t.IPAddresses, ip)
		} else {
			t.DNSNames = append(t.DNSNames, name)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, t, a.cert, pub, a.key)
	if err != nil {
		return nil, nil, err
	}
	k, err := privatePEM(key)
	if err != nil {
		return nil, nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), k, nil
}

func atomicFile(path string, data []byte, mode os.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".pki-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(mode); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// ProvisionLabPKI is an offline lab provisioner, not an online CA service.
// It never exports the CA key to Kubernetes or returns private material.
func ProvisionLabPKI(dir string, renew bool) (map[string]any, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	certData, certErr := os.ReadFile(filepath.Join(dir, "ca.crt"))
	keyData, keyErr := os.ReadFile(filepath.Join(dir, "ca.key"))
	var a *authority
	if os.IsNotExist(certErr) && os.IsNotExist(keyErr) {
		var err error
		a, err = newAuthority(now)
		if err != nil {
			return nil, err
		}
		key, err := privatePEM(a.key)
		if err != nil {
			return nil, err
		}
		if err = atomicFile(filepath.Join(dir, "ca.key"), key, 0600); err != nil {
			return nil, err
		}
		if err = atomicFile(filepath.Join(dir, "ca.crt"), a.certPEM, 0644); err != nil {
			return nil, err
		}
	} else {
		if certErr != nil || keyErr != nil {
			return nil, errors.New("incomplete lab CA; refusing replacement")
		}
		c, _ := pem.Decode(certData)
		k, _ := pem.Decode(keyData)
		if c == nil || k == nil {
			return nil, errors.New("invalid lab CA files")
		}
		cert, err := x509.ParseCertificate(c.Bytes)
		if err != nil {
			return nil, errors.New("invalid lab CA certificate")
		}
		parsed, err := x509.ParsePKCS8PrivateKey(k.Bytes)
		if err != nil {
			return nil, errors.New("invalid lab CA key")
		}
		key, ok := parsed.(ed25519.PrivateKey)
		if !ok || !cert.IsCA || cert.CheckSignatureFrom(cert) != nil || !key.Public().(ed25519.PublicKey).Equal(cert.PublicKey) || now.Before(cert.NotBefore) || !now.Before(cert.NotAfter) {
			return nil, errors.New("lab CA identity or validity mismatch")
		}
		a = &authority{cert, key, certData}
	}
	record := map[string]any{"checked_at": now, "tenant": LabTenant, "audience": LabAudience, "server_name": LabServerName, "tls_minimum": "1.3", "leaf_lifetime_hours": 24, "ca_not_after": a.cert.NotAfter, "renewal_requested": renew}
	caHash := sha256.Sum256(a.cert.Raw)
	record["ca_sha256"] = hex.EncodeToString(caHash[:])
	leaves := []map[string]any{}
	for _, role := range []string{"connector", "broker", "emergency-operator"} {
		identity := Identity(LabTenant, role)
		names := []string{}
		usage := x509.ExtKeyUsageClientAuth
		if role == "connector" {
			identity = ConnectorIdentity(LabTenant, LabAudience)
			names = []string{LabServerName}
			usage = x509.ExtKeyUsageServerAuth
		}
		certPath, keyPath := filepath.Join(dir, role+".crt"), filepath.Join(dir, role+".key")
		_, cErr := os.Stat(certPath)
		_, kErr := os.Stat(keyPath)
		if !renew && (os.IsNotExist(cErr) != os.IsNotExist(kErr)) {
			return nil, errors.New("incomplete lab leaf files; refusing implicit replacement")
		}
		if renew || (os.IsNotExist(cErr) && os.IsNotExist(kErr)) {
			cert, key, err := a.issue(identity, names, usage, now.Add(-time.Minute), now.Add(24*time.Hour))
			if err != nil {
				return nil, err
			}
			if err = atomicFile(keyPath, key, 0600); err != nil {
				return nil, err
			}
			if err = atomicFile(certPath, cert, 0644); err != nil {
				return nil, err
			}
		}
		pair, err := os.ReadFile(certPath)
		if err != nil {
			return nil, err
		}
		block, _ := pem.Decode(pair)
		if block == nil {
			return nil, errors.New("invalid lab leaf certificate")
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		roots := x509.NewCertPool()
		roots.AddCert(a.cert)
		if !peerIdentity(cert, identity) {
			return nil, errors.New("lab leaf identity mismatch")
		}
		if _, err = cert.Verify(x509.VerifyOptions{Roots: roots, KeyUsages: []x509.ExtKeyUsage{usage}}); err != nil {
			return nil, errors.New("lab leaf invalid or expired; renew explicitly")
		}
		if _, _, err = loadTLSMaterial(filepath.Join(dir, "ca.crt"), certPath, keyPath); err != nil {
			return nil, err
		}
		fingerprint := sha256.Sum256(cert.Raw)
		leaves = append(leaves, map[string]any{"role": role, "identity": identity, "not_after": cert.NotAfter, "sha256": hex.EncodeToString(fingerprint[:])})
	}
	record["certificates"] = leaves
	data, _ := json.MarshalIndent(record, "", "  ")
	if err := atomicFile(filepath.Join(dir, "record.json"), append(data, '\n'), 0644); err != nil {
		return nil, err
	}
	return record, nil
}
