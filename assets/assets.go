package assets

import (
	"embed"
)

//go:embed prompts/prompt_*.md
var PromptFiles embed.FS
