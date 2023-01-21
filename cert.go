package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"
)

type certGenerator struct {
	caCert *x509.Certificate
	caKey  crypto.PrivateKey
}

func newCertGenerator(publicKeyFile, privateKeyFile string) (*certGenerator, error) {
	tlsCert, err := tls.LoadX509KeyPair(publicKeyFile, privateKeyFile)
	if err != nil {
		return nil, err
	}
	caCert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return nil, err
	}
	return &certGenerator{
		caCert: caCert,
		caKey:  tlsCert.PrivateKey,
	}, nil
}

func (cg *certGenerator) Get(hostname string) (*tls.Config, error) {
	serial, err := getRandomSerialNumber()
	if err != nil {
		return nil, err
	}

	hostCert := &x509.Certificate{
		SerialNumber: serial,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		DNSNames:              []string{hostname},
		NotBefore:             time.Now().Add(-time.Second * 300),
		NotAfter:              time.Now().Add(time.Hour * 24 * 30),
		Subject: pkix.Name{
			CommonName: hostname,
		},
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	certDER, err := x509.CreateCertificate(rand.Reader, hostCert, cg.caCert, key.Public(), cg.caKey)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{
					certDER,
				},
				PrivateKey: key,
			},
		},
	}
	return tlsConfig, nil
}

func getRandomSerialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, serialNumberLimit)
}
