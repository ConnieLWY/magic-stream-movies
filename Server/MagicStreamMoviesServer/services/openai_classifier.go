package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// OpenAIClassifier implements ReviewClassifier with OpenAI. The client is created once at startup and reused.
type OpenAIClassifier struct {
	llm            llms.Model
	promptTemplate string
}

func NewOpenAIClassifier(apiKey, model, promptTemplate string) (*OpenAIClassifier, error) {
	llm, err := openai.New(openai.WithToken(apiKey), openai.WithModel(model))
	if err != nil {
		return nil, fmt.Errorf("create openai client: %w", err)
	}
	return &OpenAIClassifier{llm: llm, promptTemplate: promptTemplate}, nil
}

func (c *OpenAIClassifier) Classify(ctx context.Context, review string, labels []string) (string, error) {
	prompt := strings.Replace(c.promptTemplate, "{rankings}", strings.Join(labels, ","), 1) + review
	return llms.GenerateFromSinglePrompt(ctx, c.llm, prompt)
}
