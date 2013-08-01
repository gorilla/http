package client

type headers []Header

func (h headers) Len() int { return len(h) }

func (h headers) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h headers) Less(i, j int) bool { return h[i].Key < h[j].Key || h[i].Value < h[j].Value }
