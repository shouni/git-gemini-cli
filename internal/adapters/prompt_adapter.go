package adapters

import (
	"github.com/shouni/go-prompt-kit/prompts"

	"git-gemini-cli/assets"
	"git-gemini-cli/internal/domain"
)

// NewPromptAdapter は動的に読み込んだテンプレートを使用して Builder を構築します。
func NewPromptAdapter() (domain.PromptBuilder, error) {
	templates, err := assets.LoadPrompts()
	if err != nil {
		return nil, err
	}
	return prompts.NewBuilder(templates)
}
