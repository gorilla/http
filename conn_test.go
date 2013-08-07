package http

import (
	"testing"
)

var _ Conn = new(conn)
var _ Dialer = new(dialer)

type countingDialer struct {
	Dialer
	count int
}

func (c *countingDialer) Dial(nw, addr string) (Conn, error) {
	c.count++
	return c.Dialer.Dial(nw, addr)
}

func TestDialSameHost(t *testing.T) {
	s := newServer(t, stdmux())
	defer s.Shutdown()

	d := countingDialer{Dialer: new(dialer)}
	c := &Client{dialer: &d}
	_, _, b, err := c.Get(s.Root(), nil)
	if err != nil {
		t.Fatal(err)
	}
	b.Close()
	if d.count != 1 {
		t.Fatalf("dialer: expected 1, got %d", d.count)
	}
	_, _, b, err = c.Get(s.Root(), nil)
	if err != nil {
		t.Fatal(err)
	}
	b.Close()
	// should be 1 if connection is reused
	if d.count != 2 {
		t.Fatalf("dialer: expected 2, got %d", d.count)
	}
}
