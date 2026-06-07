//go:build onnx

package onnx

import (
	"fmt"
	"math"
	"sync"

	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/pretrained"
	ort "github.com/yalue/onnxruntime_go"
)

type Config struct {
	ModelPath string

	TokenizerPath string

	SharedLibraryPath string

	Dim int

	MaxSeqLen int

	InputNames []string

	OutputName string
}

var (
	initOnce sync.Once
	initErr  error
)

func ensureEnv(libPath string) error {
	initOnce.Do(func() {
		if libPath != "" {
			ort.SetSharedLibraryPath(libPath)
		}
		initErr = ort.InitializeEnvironment()
	})
	return initErr
}

type Embedder struct {
	tk         *tokenizer.Tokenizer
	modelPath  string
	dim        int
	maxSeqLen  int
	inputNames []string
	outputName string

	mu sync.Mutex
}

func NewEmbedder(cfg Config) (*Embedder, error) {
	if cfg.ModelPath == "" {
		return nil, fmt.Errorf("onnx: ModelPath must not be empty")
	}
	if cfg.TokenizerPath == "" {
		return nil, fmt.Errorf("onnx: TokenizerPath must not be empty")
	}
	if err := ensureEnv(cfg.SharedLibraryPath); err != nil {
		return nil, fmt.Errorf("onnx: init runtime: %w", err)
	}

	tk, err := pretrained.FromFile(cfg.TokenizerPath)
	if err != nil {
		return nil, fmt.Errorf("onnx: load tokenizer %q: %w", cfg.TokenizerPath, err)
	}

	dim := cfg.Dim
	if dim <= 0 {
		dim = 384
	}
	maxLen := cfg.MaxSeqLen
	if maxLen <= 0 {
		maxLen = 256
	}
	inNames := cfg.InputNames
	if len(inNames) == 0 {
		inNames = []string{"input_ids", "attention_mask", "token_type_ids"}
	}
	outName := cfg.OutputName
	if outName == "" {
		outName = "last_hidden_state"
	}

	return &Embedder{
		tk:         tk,
		modelPath:  cfg.ModelPath,
		dim:        dim,
		maxSeqLen:  maxLen,
		inputNames: inNames,
		outputName: outName,
	}, nil
}

func (e *Embedder) Dim() int { return e.dim }

func (e *Embedder) Embed(text string) ([]float32, error) {
	ids, mask, typeIDs, err := e.encode(text)
	if err != nil {
		return nil, err
	}
	seqLen := len(ids)
	if seqLen == 0 {
		return make([]float32, e.dim), nil
	}

	shape := ort.NewShape(1, int64(seqLen))
	idsTensor, err := ort.NewTensor(shape, ids)
	if err != nil {
		return nil, fmt.Errorf("onnx: input_ids tensor: %w", err)
	}
	defer idsTensor.Destroy()
	maskTensor, err := ort.NewTensor(shape, mask)
	if err != nil {
		return nil, fmt.Errorf("onnx: attention_mask tensor: %w", err)
	}
	defer maskTensor.Destroy()
	typeTensor, err := ort.NewTensor(shape, typeIDs)
	if err != nil {
		return nil, fmt.Errorf("onnx: token_type_ids tensor: %w", err)
	}
	defer typeTensor.Destroy()

	outShape := ort.NewShape(1, int64(seqLen), int64(e.dim))
	outTensor, err := ort.NewEmptyTensor[float32](outShape)
	if err != nil {
		return nil, fmt.Errorf("onnx: output tensor: %w", err)
	}
	defer outTensor.Destroy()

	session, err := ort.NewAdvancedSession(
		e.modelPath,
		e.inputNames,
		[]string{e.outputName},
		[]ort.ArbitraryTensor{idsTensor, maskTensor, typeTensor},
		[]ort.ArbitraryTensor{outTensor},
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("onnx: create session: %w", err)
	}
	defer session.Destroy()

	e.mu.Lock()
	err = session.Run()
	e.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("onnx: run: %w", err)
	}

	return meanPoolNormalize(outTensor.GetData(), mask, seqLen, e.dim), nil
}

func (e *Embedder) encode(text string) (ids, mask, typeIDs []int64, err error) {
	en, err := e.tk.EncodeSingle(text, true)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("onnx: tokenize: %w", err)
	}
	n := len(en.Ids)
	if n > e.maxSeqLen {
		n = e.maxSeqLen
	}
	ids = make([]int64, n)
	mask = make([]int64, n)
	typeIDs = make([]int64, n)
	for i := 0; i < n; i++ {
		ids[i] = int64(en.Ids[i])
		mask[i] = int64(en.AttentionMask[i])
		typeIDs[i] = int64(en.TypeIds[i])
	}
	return ids, mask, typeIDs, nil
}

func meanPoolNormalize(data []float32, mask []int64, seqLen, dim int) []float32 {
	out := make([]float32, dim)
	var count float64
	for t := 0; t < seqLen; t++ {
		if mask[t] == 0 {
			continue
		}
		count++
		base := t * dim
		for d := 0; d < dim; d++ {
			out[d] += data[base+d]
		}
	}
	if count > 0 {
		inv := float32(1 / count)
		for d := range out {
			out[d] *= inv
		}
	}

	var sum float64
	for _, v := range out {
		sum += float64(v) * float64(v)
	}
	if sum > 0 {
		inv := float32(1 / math.Sqrt(sum))
		for d := range out {
			out[d] *= inv
		}
	}
	return out
}
