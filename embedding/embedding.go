package embedding

import "context"

type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Dim() int
	Close() error
}

type Pooling int

const (
	PoolingCLS Pooling = iota
	PoolingMean
)

type LocalConfig struct {
	ModelPath     string
	TokenizerPath string
	ORTLibPath    string
	Dim           int
	MaxTokens     int
	InputNames    []string
	OutputName    string
	Pooling       Pooling
}
