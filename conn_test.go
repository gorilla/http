package http

import (
	"testing"
)

var _ Conn = new(conn)
var _ Dialer = DefaultCachingDialer
var _ Dialer = DefaultDialer
var _ Dialer = InsecureDialer
var _ Dialer = NewCachingDialer(DefaultDialer)

type countingDialer struct {
	Dialer
	count int
}

func (c *countingDialer) Dial(scheme, addr string) (Conn, error) {
	c.count++
	return c.Dialer.Dial(scheme, addr)
}

var dialCountTests = []struct {
	f        func(*testing.T, *server, Dialer)
	expected int // expected dial counts
}{
	{func(t *testing.T, s *server, d Dialer) {
		conn, err := d.Dial("http", s.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		conn.Close()
	}, 1},
	{func(t *testing.T, s *server, d Dialer) {
		conn, err := d.Dial("http", s.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		conn.Close()
		conn, err = d.Dial("http", s.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		conn.Close()
	}, 2},
	{func(t *testing.T, s *server, d Dialer) {
		conn, err := d.Dial("http", s.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		c1 := conn
		conn.Release()
		conn, err = d.Dial("http", s.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		c2 := conn
		if c1 != c2 {
			t.Errorf("expected %v == %v", c1, c2)
		}
		conn.Release()
	}, 2}, // should be 1
}

func TestDialCounts(t *testing.T) {
	s := newServer(t, stdmux())
	defer s.Shutdown()

	for i, tt := range dialCountTests {
		d := countingDialer{Dialer: DefaultCachingDialer}
		tt.f(t, s, &d)
		if actual := d.count; actual != tt.expected {
			t.Errorf("TestDialCounts %d: expected %d, got %d", i, tt.expected, actual)
		}
	}
}
