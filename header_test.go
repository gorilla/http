package http

import (
	"reflect"
	"sort"
	"testing"

	"github.com/gorilla/http/client"
)

var toHeadersTests = []struct {
	headers  map[string][]string
	expected []client.Header
}{
	{nil, nil},
	{
		map[string][]string{"Host": []string{"a"}},
		[]client.Header{{Key: "Host", Value: "a"}},
	},
	{
		map[string][]string{"Host": []string{"a", "B"}},
		[]client.Header{
			{Key: "Host", Value: "B"},
			{Key: "Host", Value: "a"},
		},
	},
	{
		map[string][]string{
			"Host":       []string{"a"},
			"Connection": []string{"close"},
		},
		[]client.Header{
			{Key: "Connection", Value: "close"},
			{Key: "Host", Value: "a"},
		},
	},
	{
		map[string][]string{
			"Host":       []string{"a", "B"},
			"Connection": []string{"close"},
		},
		[]client.Header{
			{Key: "Connection", Value: "close"},
			{Key: "Host", Value: "B"},
			{Key: "Host", Value: "a"},
		},
	},
}

func TestToHeaders(t *testing.T) {
	for _, tt := range toHeadersTests {
		actual := toHeaders(tt.headers)
		sort.Sort(client.Headers(actual))
		if !reflect.DeepEqual(tt.expected, actual) {
			t.Errorf("toHeaders(%v): expected %v, got %v", tt.headers, tt.expected, actual)
		}
	}
}

var fromHeadersTests = []struct {
	headers  []client.Header
	expected map[string][]string
}{
	{nil, nil},
	{
		[]client.Header{{Key: "Host", Value: "a"}},
		map[string][]string{"Host": []string{"a"}},
	},
	{
		[]client.Header{
			{Key: "Host", Value: "B"},
			{Key: "Host", Value: "a"},
		},
		map[string][]string{"Host": []string{"B", "a"}},
	},
	{
		[]client.Header{
			{Key: "Connection", Value: "close"},
			{Key: "Host", Value: "a"},
		},
		map[string][]string{
			"Host":       []string{"a"},
			"Connection": []string{"close"},
		},
	},
	{
		[]client.Header{
			{Key: "Connection", Value: "close"},
			{Key: "Host", Value: "B"},
			{Key: "Host", Value: "a"},
		},
		map[string][]string{
			"Host":       []string{"B", "a"},
			"Connection": []string{"close"},
		},
	},
}

func TestFromHeaders(t *testing.T) {
	for _, tt := range fromHeadersTests {
		actual := fromHeaders(tt.headers)
		if !reflect.DeepEqual(tt.expected, actual) {
			t.Errorf("fromHeaders(%v): expected %v, got %v", tt.headers, tt.expected, actual)
		}
	}
}
