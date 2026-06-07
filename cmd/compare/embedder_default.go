//go:build !onnx

package main

import "github.com/Fiecher/searchinator/pkg/semantic"

func newEmbedder() (semantic.Embedder, error) {
	return semantic.NewHashEmbedder(256), nil
}

func semanticName() string { return "HashEmbedder" }
