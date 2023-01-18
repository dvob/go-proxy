package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
)

func logRequest(r *http.Request) {
	requestDump, err := httputil.DumpRequest(r, false)
	if err != nil {
		requestDump = []byte("failed to dump request")
	}
	log.Printf("scheme=%s host=%s path=%s\n%s", r.URL.Scheme, r.URL.Host, r.URL.Path, requestDump)
}

func tunnel(w http.ResponseWriter, r *http.Request) {
	dialer := net.Dialer{}
	upstreamConn, err := dialer.DialContext(r.Context(), "tcp", r.Host)
	if err != nil {
		log.Printf("failed to connect to upstream %s", r.Host)
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	defer upstreamConn.Close()
	w.WriteHeader(http.StatusOK)

	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}

	downstreamConn, bufferedDownstreamConn, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer downstreamConn.Close()
	go io.Copy(upstreamConn, bufferedDownstreamConn)
	io.Copy(bufferedDownstreamConn, upstreamConn)
}

func forward(w http.ResponseWriter, r *http.Request) {
	logRequest(r)
	resp, err := http.DefaultTransport.RoundTrip(r)
	if err != nil {
		log.Print(err)
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

func proxy(w http.ResponseWriter, r *http.Request) {
	if r.Method == "CONNECT" {
		tunnel(w, r)
	} else {
		forward(w, r)
	}
}

func main() {
	// initialize interceptor
	interceptor, err := newIntercepter()
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		err = interceptor.serve()
		if err != nil {
			log.Fatal(err)
		}
	}()

	err = http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "CONNECT" {
			interceptor.intercept(w, r)
		} else {
			forward(w, r)
		}
	}))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
