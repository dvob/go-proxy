package main

import (
	"io"
	"log"
	"net/http"
	"os"
)

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
	err := http.ListenAndServe(":8080", http.HandlerFunc(forward))
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
