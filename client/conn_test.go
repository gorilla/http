package client

import (
	"testing"
)

var phaseStringTests = []struct {
	phase
	expected string
}{
	{0, "UNKNOWN"},
	{1, "headers"},
	{2, "body"},
	{3, "UNKNOWN"},
}

func TestPhaseString(t *testing.T) {
	for _, tt := range phaseStringTests {
		actual := tt.phase.String()
		if actual != tt.expected {
			t.Errorf("phase(%d).String(): expected %q, got %q", tt.phase, tt.expected, actual)
		}
	}
}

func TestPhaseError(t *testing.T) {
	var c Conn
	err := c.WriteHeader("Host", "localhost")
	if _, ok := err.(*phaseError); !ok {
		t.Fatalf("expected %T, got %v", new(phaseError), err)
	}
	expected := `phase error: expected headers, got UNKNOWN`
	if actual := err.Error(); actual != expected {
		t.Fatalf("phaseError.Error(): expected %q, got %q", expected, actual)
	}
}
