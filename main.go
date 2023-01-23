package main

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
)

func logRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestDump, _ := httputil.DumpRequest(r, false)
		log.Printf("url=%s\n%s", r.URL, requestDump)
		next.ServeHTTP(w, r)
	}
}

func tunnel(w http.ResponseWriter, r *http.Request) {
	dialer := net.Dialer{}
	serverConn, err := dialer.DialContext(r.Context(), "tcp", r.Host)
	if err != nil {
		log.Printf("failed to connect to upstream %s", r.Host)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	defer serverConn.Close()

	hj, ok := w.(http.Hijacker)
	if !ok {
		log.Print("hijack of connection failed")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	clientConn, bufClientConn, err := hj.Hijack()
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	go io.Copy(serverConn, bufClientConn)
	io.Copy(bufClientConn, serverConn)
}

func forward(w http.ResponseWriter, r *http.Request) {
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	for header, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(header, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func main() {
	var (
		doCreateCA bool
		caCertFile = "proxy-ca.crt"
		caKeyFile  = "proxy-ca.key"
	)

	flag.BoolVar(&doCreateCA, "create-ca", false, "create a CA for the proxy")
	flag.Parse()

	if doCreateCA {
		err := createCA(caCertFile, caKeyFile)
		if err != nil {
			log.Print(err)
			os.Exit(1)
		}
		return
	}

	handler := logRequest(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "CONNECT" {
			tunnel(w, r)
		} else {
			forward(w, r)
		}
	})

	err := http.ListenAndServe(":8080", http.HandlerFunc(handler))
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
