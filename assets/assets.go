package assets

import "embed"

const PromptFilesDir = "prompts"

//go:embed prompts/prompt_*.md
var PromptFiles embed.FS
