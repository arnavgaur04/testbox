package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"testbox/internal/api"
)

func main() {
	addr := flag.String("addr", ":8081", "Listen address for API server")
	flag.Parse()

	srv := api.NewServer()
	fmt.Printf("testbox API listening on %s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, srv.Handler()))
}
