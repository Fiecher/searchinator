//go:build onnx

package main

import (
	"flag"

	"github.com/Fiecher/searchinator/pkg/semantic"
	"github.com/Fiecher/searchinator/pkg/semantic/onnx"
)

var (
	modelPath = flag.String("model", "", "path to the MiniLM model.onnx (required with -tags onnx)")
	tokPath   = flag.String("tokenizer", "", "path to tokenizer.json (required with -tags onnx)")
	libPath   = flag.String("ort-lib", "", "path to the ONNX Runtime shared library")
)

func newEmbedder() (semantic.Embedder, error) {
	return onnx.NewEmbedder(onnx.Config{
		ModelPath:         *modelPath,
		TokenizerPath:     *tokPath,
		SharedLibraryPath: *libPath,
	})
}

func semanticName() string { return "ONNX-MiniLM" }
