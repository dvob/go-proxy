package main

import (
	"fmt"
	"io"
	"log"
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

func main() {
	err := http.ListenAndServe(":8080", http.HandlerFunc(forward))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
