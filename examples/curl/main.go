// curl is a simple cURL replacement.
package main

import (
	"log"
	"os"

	"github.com/gorilla/http"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: %v $URL", os.Args[0])
	}
	if _, err := http.Get(os.Stdout, os.Args[1]); err != nil {
		log.Fatalf("unable to fetch %q: %v", os.Args[1], err)
	}
}
