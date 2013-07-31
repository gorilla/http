package http

import (
	"os"
	"log"
)

func ExampleGet() {
	// curl in 3 lines of code.
	if _, err := Get(os.Stdout, "http://www.gorillatoolkit.org/") {
		log.Fatalf("could not fetch: %v", err)
	}
}
