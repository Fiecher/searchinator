package searchinator

type Document struct {
	ID   string
	Text string
	Meta map[string]any
}

type Result struct {
	Document Document
	Score    float64
}
