package adapters

import (
	"context"
	"fmt"

	"git-gemini-cli/internal/config"

	"github.com/shouni/gemini-reviewer-core/ai"
)

// NewCodeReviewAI は adapters.CodeReviewAI のインスタンスを構築します。
func NewCodeReviewAI(ctx context.Context, cfg *config.Config) (*ai.GeminiAdapter, error) {
	opt := ai.GeminiOptions{
		ProjectID: cfg.ProjectID,
		APIKey:    cfg.GeminiAPIKey,
	}
	codeReviewAI, err := ai.NewGeminiAdapter(ctx, opt)
	if err != nil {
		return nil, fmt.Errorf("CodeReviewAIアダプターの構築に失敗しました: %w", err)
	}
	return codeReviewAI, nil
}
