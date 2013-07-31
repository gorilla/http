package http

import (
	"bytes"
	"log"
)

func ExampleGet() {
	var b bytes.Buffer
	var url = "http://www.gorillatoolkit.org/"
	_, err := Get(&b, url)
	if err != nil {
		log.Fatalf("could not fetch: %v", err)
	}
	log.Printf("Fetched %v: %q", err, b.String())
}
