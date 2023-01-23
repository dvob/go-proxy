package main

import (
	"io"
	"log"
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
	handler := logRequest(forward)

	err := http.ListenAndServe(":8080", http.HandlerFunc(handler))
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
