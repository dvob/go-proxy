package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"log"
	"math/big"
	"net"
	"net/http"
	"time"
)

type intercepter struct {
	proxyCACert *x509.Certificate
	proxyCAKey  crypto.PrivateKey
	conns       chan net.Conn
	srv         *http.Server
}

func (i *intercepter) Accept() (net.Conn, error) {
	return <-i.conns, nil
}

func (i *intercepter) Addr() net.Addr {
	return nil
}

func (i *intercepter) Close() error {
	close(i.conns)
	return i.srv.Close()
}

func newIntercepter() (*intercepter, error) {
	tlsCert, err := tls.LoadX509KeyPair("proxy-ca.crt", "proxy-ca.key")
	if err != nil {
		return nil, err
	}
	proxyCA, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return nil, err
	}
	return &intercepter{
		proxyCACert: proxyCA,
		proxyCAKey:  tlsCert.PrivateKey,
		conns:       make(chan net.Conn),
		srv: &http.Server{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				r.URL.Scheme = "https"
				r.URL.Host = r.Host
				forward(w, r)
			}),
		},
	}, nil
}

func (i *intercepter) serve() error {
	return i.srv.Serve(i)
}

func (i *intercepter) handleConn(c net.Conn) {
	i.conns <- c
}

func (i *intercepter) getTLSConfigForHost(host string) (*tls.Config, error) {
	host, _, err := net.SplitHostPort(host)
	if err != nil {
		return nil, err
	}

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
		DNSNames:              []string{host},
		NotBefore:             time.Now().Add(-time.Second * 300),
		NotAfter:              time.Now().Add(time.Hour * 24 * 30),
		Subject: pkix.Name{
			CommonName: host,
		},
	}

	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}

	der, err := x509.CreateCertificate(rand.Reader, hostCert, i.proxyCACert, key.Public(), i.proxyCAKey)
	if err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{
					der,
				},
				PrivateKey: key,
			},
		},
	}
	return tlsConfig, nil
}

func (i *intercepter) intercept(w http.ResponseWriter, r *http.Request) {
	logRequest(r)
	tlsConfig, err := i.getTLSConfigForHost(r.Host)
	if err != nil {
		log.Println("failed to obtain tls config:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}
	downstreamConn, _, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tlsConn := tls.Server(downstreamConn, tlsConfig)
	i.handleConn(tlsConn)
}

func getRandomSerialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, serialNumberLimit)
}
