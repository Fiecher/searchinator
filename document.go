package searchinator

type Document struct {
	ID   string
	Text string
	Meta map[string]any

	Fields map[string]string
}

type Result struct {
	Document Document
	Score    float64
}
