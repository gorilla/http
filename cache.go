package http

import (
	"sync"
)

type connKey struct {
	scheme string
	host   string
}

type cachingDialer struct {
	sync.Mutex                        // protects following fields
	conns          map[connKey][]Conn // maps call to a, possibly empty, slice of existing Conns
	straightDialer Dialer
}

// NewCachingDialer takes an existing dialer and essentially memoizes the Dial
// method with inactive connections.
func NewCachingDialer(d Dialer) Dialer {
	return &cachingDialer{
		conns:          make(map[connKey][]Conn),
		straightDialer: d}
}

func (d *cachingDialer) Dial(scheme, host string) (Conn, error) {
	key := connKey{scheme: scheme, host: host}
	d.Lock()
	if c, ok := d.conns[key]; ok {
		if len(c) > 0 {
			conn := c[0]
			c[0], c = c[len(c)-1], c[:len(c)-1]
			d.Unlock()
			return conn, nil
		}
	}
	d.Unlock()

	c, err := d.straightDialer.Dial(scheme, host)
	return &cachedConn{
		Conn:          c,
		cachingDialer: d,
		key:           key}, err
}

type cachedConn struct {
	Conn

	cachingDialer *cachingDialer
	key           connKey
}

func (c *cachedConn) Release() {
	c.Conn.Release()
	c.cachingDialer.Lock()
	defer c.cachingDialer.Unlock()
	c.cachingDialer.conns[c.key] = append(c.cachingDialer.conns[c.key], c)
}

// DefaultCachingDialer is simply the DefaultDialer wrapped with a cache.
var DefaultCachingDialer = NewCachingDialer(DefaultDialer)
