package client

import (
	"bytes"
	"testing"
)

func TestnewConn(t *testing.T) {
	var b bytes.Buffer
	newConn(&b)
}
