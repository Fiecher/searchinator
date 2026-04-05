package analysis

type Tokenizer interface {
	Tokenize(text string) []string
}

type TokenFilter interface {
	Filter(tokens []string) []string
}

type Analyzer interface {
	Analyze(text string) []string
}

type PipelineAnalyzer struct {
	tokenizer Tokenizer
	filters   []TokenFilter
}

func NewPipelineAnalyzer(tokenizer Tokenizer, filters ...TokenFilter) *PipelineAnalyzer {
	return &PipelineAnalyzer{
		tokenizer: tokenizer,
		filters:   filters,
	}
}

func (p *PipelineAnalyzer) Analyze(text string) []string {
	tokens := p.tokenizer.Tokenize(text)
	for _, f := range p.filters {
		tokens = f.Filter(tokens)
	}
	return tokens
}
