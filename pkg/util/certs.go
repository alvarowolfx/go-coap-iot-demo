package util

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pion/dtls/v2/examples/util"
)

func GetRootCert() (*tls.Certificate, error) {
	return util.LoadCertificate("./certs/server.pem")
}

func GetCert() *tls.Certificate {
	cert, err := util.LoadKeyAndCertificate("./certs/server-key.pem", "./certs/server.pem")
	if err == nil {
		return cert
	}
	log.Printf("failed reading certs files: %v", err)

	var rootTemplate = x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Country:      []string{"US"},
			Organization: []string{"Company Co."},
			CommonName:   "Root CA",
		},
		NotBefore:             time.Now().Add(-10 * time.Second),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            2,
		IPAddresses:           []net.IP{net.ParseIP("0.0.0.0")},
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}

	rootCert := genCert(&rootTemplate, &rootTemplate, &priv.PublicKey, priv)

	return rootCert
}

func genCert(template, parent *x509.Certificate, publicKey *ecdsa.PublicKey, privateKey *ecdsa.PrivateKey) *tls.Certificate {
	certBytes, err := x509.CreateCertificate(rand.Reader, template, parent, publicKey, privateKey)
	if err != nil {
		panic("Failed to create certificate:" + err.Error())
	}

	certOut, err := os.Create("./certs/server.pem")
	if err != nil {
		log.Fatalf("err failed to create root_ca: %v", err)
	}

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		log.Fatalf("Failed to write data to server.pem: %v", err)
	}

	keyOut, err := os.OpenFile("./server-key.pem", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Failed to open key.pem for writing: %v", err)
	}

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		log.Fatalf("Unable to marshal private key: %v", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		log.Fatalf("Failed to write data to key.pem: %v", err)
	}

	if err := certOut.Close(); err != nil {
		log.Fatalf("Error closing server.pem: %v", err)
	}

	if err := keyOut.Close(); err != nil {
		log.Fatalf("Error closing server-key.pem: %v", err)
	}

	privKey, err := LoadKey("./server-key.pem")
	if err != nil {
		log.Fatalf("err parsing private key: %v", err)
	}

	return &tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey:  privKey,
	}
}

// LoadKeyAndCertificate reads certificates or key from file
func LoadKeyAndCertificate(keyPath string, certificatePath string) (*tls.Certificate, error) {
	privateKey, err := LoadKey(keyPath)
	if err != nil {
		return nil, err
	}

	certificate, err := LoadCertificate(certificatePath)
	if err != nil {
		return nil, err
	}

	certificate.PrivateKey = privateKey

	return certificate, nil
}

// LoadKey Load/read key from file
func LoadKey(path string) (crypto.PrivateKey, error) {
	rawData, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(rawData)
	if block == nil || !strings.HasSuffix(block.Type, "PRIVATE KEY") {
		return nil, errors.New("block is not a private key, unable to load key")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey, *ecdsa.PrivateKey:
			return key, nil
		default:
			return nil, errors.New("unknown key time in PKCS#8 wrapping, unable to load key")
		}
	}

	if key, err := x509.ParseECPrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	return nil, errors.New("no private key found, unable to load key")
}

// LoadCertificate Load/read certificate(s) from file
func LoadCertificate(path string) (*tls.Certificate, error) {
	rawData, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	var certificate tls.Certificate

	for {
		block, rest := pem.Decode(rawData)
		if block == nil {
			break
		}

		if block.Type != "CERTIFICATE" {
			return nil, errors.New("block is not a certificate, unable to load certificates")
		}

		certificate.Certificate = append(certificate.Certificate, block.Bytes)
		rawData = rest
	}

	if len(certificate.Certificate) == 0 {
		return nil, errors.New("no certificate found, unable to load certificates")
	}

	return &certificate, nil
}
