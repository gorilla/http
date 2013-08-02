package client

type Headers []Header

func (h Headers) Len() int { return len(h) }

func (h Headers) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h Headers) Less(i, j int) bool { return h[i].Key < h[j].Key || h[i].Value < h[j].Value }
