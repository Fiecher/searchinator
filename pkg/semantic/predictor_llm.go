package semantic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"
)

type LLMConfig struct {
	Endpoint string

	APIKey string

	Model string

	Client *http.Client
}

type LLMPredictor struct {
	cfg    LLMConfig
	client *http.Client
}

func NewLLMPredictor(cfg LLMConfig) (*LLMPredictor, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, fmt.Errorf("semantic: LLM endpoint must not be empty")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, fmt.Errorf("semantic: LLM model must not be empty")
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &LLMPredictor{cfg: cfg, client: client}, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func (l *LLMPredictor) Predict(description string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 6
	}
	prompt := fmt.Sprintf(
		"Описание: %q. Перечисли через запятую %d отдельных слов (без пояснений), "+
			"которые с наибольшей вероятностью встречаются в текстах, подходящих под это описание.",
		description, limit)

	body, err := json.Marshal(chatRequest{
		Model: l.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: "Ты помощник поиска. Отвечай только списком слов через запятую."},
			{Role: "user", Content: prompt},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("semantic: marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.cfg.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("semantic: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if l.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+l.cfg.APIKey)
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("semantic: LLM request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("semantic: LLM status %d", resp.StatusCode)
	}

	var parsed chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("semantic: decode response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("semantic: LLM returned no choices")
	}
	return parseWords(parsed.Choices[0].Message.Content, limit), nil
}

func parseWords(reply string, limit int) []string {
	fields := strings.FieldsFunc(strings.ToLower(reply), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	seen := make(map[string]struct{}, len(fields))
	out := make([]string, 0, limit)
	for _, w := range fields {
		if _, ok := seen[w]; ok {
			continue
		}
		seen[w] = struct{}{}
		out = append(out, w)
		if limit > 0 && len(out) == limit {
			break
		}
	}
	return out
}
