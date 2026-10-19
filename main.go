package main

import (
	"log"
	"net/http"
)

func main() {
	store := NewPaymentStore()

	mux := newRouter(store)

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}
