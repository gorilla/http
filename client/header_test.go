package client

import (
	"reflect"
	"sort"
	"testing"
)

var headersSortTests = []struct {
	headers, expected []Header
}{
	{nil, nil},
	{[]Header{{"Host", "localhost"}}, []Header{{"Host", "localhost"}}},
	{[]Header{{"Connection", "close"}, {"Host", "localhost"}}, []Header{{"Connection", "close"}, {"Host", "localhost"}}},
	{[]Header{{"Host", "localhost"}, {"Connection", "close"}}, []Header{{"Connection", "close"}, {"Host", "localhost"}}},
	{[]Header{{"Host", "A"}, {"Host", "b"}}, []Header{{"Host", "A"}, {"Host", "b"}}},
	{[]Header{{"Host", "b"}, {"Host", "A"}}, []Header{{"Host", "A"}, {"Host", "b"}}},
}

func TestHeadersSort(t *testing.T) {
	for _, tt := range headersSortTests {
		sort.Sort(Headers(tt.headers)) // mutates test fixture
		if !reflect.DeepEqual(tt.headers, tt.expected) {
			t.Errorf("expected %v, got %v", tt.expected, tt.headers)
		}
	}
}
