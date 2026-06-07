package engine

import (
	"github.com/Fiecher/searchinator/pkg/analysis"
	"github.com/Fiecher/searchinator/pkg/ranking"
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

func EnglishConfig() Config {
	return Config{
		Analyzer: analysis.NewPipelineAnalyzer(
			analysis.NewWhitespaceTokenizer(),
			analysis.NewLowercaseFilter(),
			analysis.NewPunctuationFilter(),
			analysis.NewStopWordsFilter(analysis.DefaultEnglishStopWords()),
			analysis.NewPorterStemmer(),
		),
		Ranker: ranking.NewBM25(ranking.DefaultBM25Params()),
	}
}

func BilingualConfig() Config {
	return Config{
		Analyzer: analysis.NewPipelineAnalyzer(
			analysis.NewWhitespaceTokenizer(),
			analysis.NewLowercaseFilter(),
			analysis.NewPunctuationFilter(),
			analysis.NewStopWordsFilter(analysis.DefaultEnglishStopWords()),
			analysis.NewStopWordsFilter(analysis.DefaultRussianStopWords()),
			analysis.NewPorterStemmer(),
			analysis.NewRussianStemmer(),
		),
		Ranker: ranking.NewBM25(ranking.DefaultBM25Params()),
	}
}

func FuzzyConfig(vocab []string, maxDistance int) Config {
	return Config{
		Analyzer: analysis.NewPipelineAnalyzer(
			analysis.NewWhitespaceTokenizer(),
			analysis.NewLowercaseFilter(),
			analysis.NewPunctuationFilter(),
			analysis.NewFuzzyFilter(vocab, maxDistance),
		),
		Ranker: ranking.NewBM25(ranking.DefaultBM25Params()),
	}
}
