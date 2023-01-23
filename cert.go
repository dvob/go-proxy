package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"time"
)

func createCA(caCertFile, caKeyFile string) error {
	serial, err := getRandomSerialNumber()
	if err != nil {
		return err
	}

	caCert := &x509.Certificate{
		SerialNumber:          serial,
		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature |
			x509.KeyUsageCertSign |
			x509.KeyUsageCRLSign,
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 365 * 10),
		Subject: pkix.Name{
			CommonName: "Proxy CA",
		},
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caCert, caCert, key.Public(), key)
	if err != nil {
		return err
	}
	caCertPEM := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCertDER,
	}
	err = os.WriteFile(caCertFile, pem.EncodeToMemory(caCertPEM), 0640)
	if err != nil {
		return err
	}

	caKeyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}
	caKeyPEM := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: caKeyDER,
	}
	err = os.WriteFile(caKeyFile, pem.EncodeToMemory(caKeyPEM), 0600)
	if err != nil {
		return err
	}
	return nil
}

func getRandomSerialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, serialNumberLimit)
}
