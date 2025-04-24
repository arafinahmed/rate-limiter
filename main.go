package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/unlimited", unlimitedHandler)
	http.HandleFunc("/limited", limitedHandler)

	addr := "127.0.0.1:8080"
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func unlimitedHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Unlimited! Let's Go!")
}

func limitedHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Limited, don't over use me!")
}
