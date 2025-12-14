package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

// GenerateSelfSignedCert generates a self-signed TLS certificate for the given IP address
func GenerateSelfSignedCert(ipAddr string) (tls.Certificate, error) {
	// step 1: generate RSA private key (2048-bit)
	// this key is used for both signing and encryption
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("TLS: Failed to generate private key: %v", err)
	}

	// step 2: create certificate template (X.509 standard)
	// defines what the certificate will contain
	template := x509.Certificate{
		// serial number must be unique (using current time as random source)
		SerialNumber: big.NewInt(time.Now().Unix()),

		// subject identifies who this cert belongs to
		Subject: pkix.Name{
			Organization: []string{"Chord DHT"},
			CommonName:   ipAddr, // IP:PORT of this node
		},

		// validity period
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year

		// key usage: what this cert can be used for
		KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,

		// extended key usage: specifically for TLS server authentication
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},

		// basic constraints
		BasicConstraintsValid: true,
	}

	// step 3: add IP address to certificates subject alternative names (SAN)
	// allows the cert to be valid for this specific IP
	host, _, err := net.SplitHostPort(ipAddr)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("invalid IP address format: %v", err)
	}

	ip := net.ParseIP(host)
	if ip != nil {
		template.IPAddresses = append(template.IPAddresses, ip)
	} else {
		// If not an IP, treat as DNS name
		template.DNSNames = append(template.DNSNames, host)
	}

	// step 4: create the certificate (self-signed)
	// x509.CreateCertificate signs the template with the private key
	// parent = template means "self-signed" (we sign our own cert)
	certDER, err := x509.CreateCertificate(
		rand.Reader,           // random source for crypto operations
		&template,             // certificate to create
		&template,             // parent cert (same = self-signed)
		&privateKey.PublicKey, // public key to include in cert
		privateKey,            // private key to sign with
	)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to create certificate: %v", err)
	}

	// step 5: encode certificate to PEM (Privacy Enhanced Mail) format (text-based encoding)
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// step 6: encode private key to PEM format
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// step 7: create tls.Certificate from PEM-encoded data
	// final format needed by Go's TLS library
	tlsCert, err := tls.X509KeyPair(certPEM, privateKeyPEM)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("failed to create TLS certificate: %v", err)
	}

	return tlsCert, nil
}
