package adapters

import (
	"embed"
	"fmt"
	"path"
	"strings"

	"github.com/shouni/gemini-reviewer-core/pkg/domain"
	"github.com/shouni/gemini-reviewer-core/pkg/prompts"

	"git-gemini-cli/assets"
)

// NewPromptAdapter は動的に読み込んだテンプレートを使用して Builder を構築します。
func NewPromptAdapter() (domain.PromptBuilder, error) {
	templates, err := loadTemplates(assets.PromptFiles, assets.PromptFilesDir)
	if err != nil {
		return nil, err
	}

	return prompts.NewBuilder(templates)
}

// loadTemplates は指定された FS からプロンプトファイルを読み込み、 ファイル名をキー（mode名）としたマップを返します。
func loadTemplates(fs embed.FS, rootDir string) (map[string]string, error) {
	templates := make(map[string]string)

	entries, err := fs.ReadDir(rootDir)
	if err != nil {
		return nil, fmt.Errorf("ディレクトリ %s の読み込みに失敗: %w", rootDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		modeName := strings.TrimPrefix(
			strings.TrimSuffix(fileName, path.Ext(fileName)),
			"prompt_",
		)

		filePath := path.Join(rootDir, fileName)
		content, err := fs.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("ファイル %s の読み込みに失敗: %w", filePath, err)
		}

		templates[modeName] = string(content)
	}

	return templates, nil
}
