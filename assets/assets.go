package assets

import "embed"

const (
	promptDir    = "prompts"
	promptPrefix = "prompt_"
)

//go:embed prompts/prompt_*.md
var PromptFiles embed.FS

// LoadPrompts は埋め込まれたプロンプトファイルを読み込みます。
func LoadPrompts() (map[string]string, error) {
	return load(PromptFiles, promptDir, promptPrefix)
}
