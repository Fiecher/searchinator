package engine

import (
	"github.com/Fiecher/searchinator/analysis"
	"github.com/Fiecher/searchinator/ranking"
)

type Config struct {
	Analyzer analysis.Analyzer
	Ranker   ranking.Ranker
}

func DefaultConfig() Config {
	return Config{
		Analyzer: analysis.NewPipelineAnalyzer(
			analysis.NewWhitespaceTokenizer(),
			analysis.NewLowercaseFilter(),
			analysis.NewPunctuationFilter(),
		),
		Ranker: ranking.NewBM25(ranking.DefaultBM25Params()),
	}
}
