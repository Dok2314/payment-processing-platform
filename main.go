package main

import (
	"log"
	"net/http"
)

func main() {
	store := NewPaymentStore()

	service := NewPaymentService(store)

	mux := newRouter(service)

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}
