package embedding

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/daulet/tokenizers"
	onnxruntime_go "github.com/yalue/onnxruntime_go"
)

var (
	ortInitOnce sync.Once
	ortInitErr  error
)

func initRuntime(libPath string) error {
	ortInitOnce.Do(func() {
		if libPath != "" {
			onnxruntime_go.SetSharedLibraryPath(libPath)
		}
		ortInitErr = onnxruntime_go.InitializeEnvironment()
	})
	return ortInitErr
}

type Local struct {
	tokenizer  *tokenizers.Tokenizer
	session    *onnxruntime_go.DynamicAdvancedSession
	inputNames []string
	dim        int
	maxTokens  int
	pooling    Pooling
	mu         sync.Mutex
}

func NewLocal(cfg LocalConfig) (*Local, error) {
	if cfg.ModelPath == "" || cfg.TokenizerPath == "" {
		return nil, fmt.Errorf("embedding: ModelPath and TokenizerPath are required")
	}
	if cfg.Dim <= 0 {
		return nil, fmt.Errorf("embedding: Dim must be positive, got %d", cfg.Dim)
	}

	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 8192
	}
	inputNames := cfg.InputNames
	if len(inputNames) == 0 {
		inputNames = []string{"input_ids", "attention_mask"}
	}
	outputName := cfg.OutputName
	if outputName == "" {
		outputName = "last_hidden_state"
	}

	if err := initRuntime(cfg.ORTLibPath); err != nil {
		return nil, fmt.Errorf("embedding: init runtime: %w", err)
	}

	tok, err := tokenizers.FromFile(cfg.TokenizerPath)
	if err != nil {
		return nil, fmt.Errorf("embedding: load tokenizer: %w", err)
	}

	session, err := onnxruntime_go.NewDynamicAdvancedSession(
		cfg.ModelPath, inputNames, []string{outputName}, nil)
	if err != nil {
		tok.Close()
		return nil, fmt.Errorf("embedding: create session: %w", err)
	}

	return &Local{
		tokenizer:  tok,
		session:    session,
		inputNames: inputNames,
		dim:        cfg.Dim,
		maxTokens:  maxTokens,
		pooling:    cfg.Pooling,
	}, nil
}

var _ Embedder = (*Local)(nil)

func (l *Local) Dim() int { return l.dim }

func (l *Local) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.session != nil {
		l.session.Destroy()
		l.session = nil
	}
	if l.tokenizer != nil {
		err := l.tokenizer.Close()
		l.tokenizer = nil
		return err
	}
	return nil
}

func (l *Local) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.session == nil || l.tokenizer == nil {
		return nil, fmt.Errorf("embedding: embedder is closed")
	}

	batch := len(texts)
	idsList := make([][]int64, batch)
	maskList := make([][]int64, batch)
	maxLen := 1
	for i, text := range texts {
		enc := l.tokenizer.EncodeWithOptions(text, true, tokenizers.WithReturnAttentionMask())
		ids, mask := enc.IDs, enc.AttentionMask
		if len(ids) > l.maxTokens {
			ids = ids[:l.maxTokens]
			mask = mask[:l.maxTokens]
		}
		ii := make([]int64, max(len(ids), 1))
		mm := make([]int64, len(ii))
		for j := range ids {
			ii[j] = int64(ids[j])
			if j < len(mask) {
				mm[j] = int64(mask[j])
			}
		}
		idsList[i], maskList[i] = ii, mm
		maxLen = max(maxLen, len(ii))
	}

	flatIDs := make([]int64, batch*maxLen)
	flatMask := make([]int64, batch*maxLen)
	for i := range batch {
		copy(flatIDs[i*maxLen:], idsList[i])
		copy(flatMask[i*maxLen:], maskList[i])
	}

	shape := onnxruntime_go.NewShape(int64(batch), int64(maxLen))
	inputs := make([]onnxruntime_go.Value, len(l.inputNames))
	defer func() {
		for _, v := range inputs {
			if v != nil {
				v.Destroy()
			}
		}
	}()

	for i, name := range l.inputNames {
		var data []int64
		switch name {
		case "input_ids":
			data = flatIDs
		case "attention_mask":
			data = flatMask
		case "token_type_ids":
			data = make([]int64, batch*maxLen)
		default:
			return nil, fmt.Errorf("embedding: unsupported input name %q", name)
		}
		tensor, err := onnxruntime_go.NewTensor(shape, data)
		if err != nil {
			return nil, fmt.Errorf("embedding: create input %q: %w", name, err)
		}
		inputs[i] = tensor
	}

	outputs := []onnxruntime_go.Value{nil}
	if err := l.session.Run(inputs, outputs); err != nil {
		return nil, fmt.Errorf("embedding: run: %w", err)
	}

	out, ok := outputs[0].(*onnxruntime_go.Tensor[float32])
	if !ok {
		if outputs[0] != nil {
			outputs[0].Destroy()
		}
		return nil, fmt.Errorf("embedding: unexpected output type %T", outputs[0])
	}
	defer out.Destroy()

	return pool(out.GetShape(), out.GetData(), flatMask, maxLen, batch, l.dim, l.pooling)
}

func pool(shape onnxruntime_go.Shape, data []float32, mask []int64, maxLen, batch, dim int, mode Pooling) ([][]float32, error) {
	result := make([][]float32, batch)
	switch len(shape) {
	case 3:
		seqLen, hidden := int(shape[1]), int(shape[2])
		if hidden != dim {
			return nil, fmt.Errorf("embedding: model hidden %d != configured dim %d", hidden, dim)
		}
		if mode == PoolingMean && seqLen != maxLen {
			return nil, fmt.Errorf("embedding: mean pooling needs mask stride %d == seq len %d", maxLen, seqLen)
		}
		for b := range batch {
			base := b * seqLen * hidden
			if mode == PoolingMean {
				result[b] = meanPool(data[base:base+seqLen*hidden], mask[b*maxLen:b*maxLen+seqLen], seqLen, hidden)
				continue
			}
			vec := make([]float32, hidden)
			copy(vec, data[base:base+hidden])
			normalize(vec)
			result[b] = vec
		}
	case 2:
		hidden := int(shape[1])
		if hidden != dim {
			return nil, fmt.Errorf("embedding: model hidden %d != configured dim %d", hidden, dim)
		}
		for b := range batch {
			vec := make([]float32, hidden)
			copy(vec, data[b*hidden:(b+1)*hidden])
			normalize(vec)
			result[b] = vec
		}
	default:
		return nil, fmt.Errorf("embedding: unexpected output rank %d", len(shape))
	}
	return result, nil
}

func meanPool(rows []float32, mask []int64, seqLen, hidden int) []float32 {
	vec := make([]float32, hidden)
	for t := range seqLen {
		if mask[t] == 0 {
			continue
		}
		off := t * hidden
		for h := range hidden {
			vec[h] += rows[off+h]
		}
	}
	normalize(vec)
	return vec
}

func normalize(v []float32) {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	if sum == 0 {
		return
	}
	inv := float32(1.0 / math.Sqrt(sum))
	for i := range v {
		v[i] *= inv
	}
}
