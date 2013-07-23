package http_test

import (
	"testing"

	"github.com/gorilla/http"
)

func TestClientGet(t *testing.T) {
	s := newServer(t)
	defer s.Shutdown()
	err := http.Default.Get(s.Root())
	if err != nil {
		t.Fatal(err)
	}
}
