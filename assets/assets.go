package assets

import _ "embed"

var (
	// DetailPrompt は詳細レビュー用のプロンプトテンプレートです。
	//go:embed prompts/prompt_detail.md
	DetailPrompt string

	// ReleasePrompt はリリース判定用のプロンプトテンプレートです。
	//go:embed prompts/prompt_release.md
	ReleasePrompt string
)
