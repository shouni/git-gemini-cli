package adapters

import (
	"github.com/shouni/gemini-reviewer-core/pkg/domain"
	"github.com/shouni/gemini-reviewer-core/pkg/prompts"

	"git-gemini-cli/assets"
)

// NewPromptAdapter は動的に読み込んだテンプレートを使用して Builder を構築します。
func NewPromptAdapter() (domain.PromptBuilder, error) {
	templates, err := assets.LoadPrompts()
	if err != nil {
		return nil, err
	}
	return prompts.NewBuilder(templates)
}
