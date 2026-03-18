package assets

import "embed"

const (
	PromptDir    = "prompts"
	PromptPrefix = "prompt_"
)

//go:embed prompts/prompt_*.md
var PromptFiles embed.FS
