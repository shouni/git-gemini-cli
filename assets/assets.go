// Package assets は、プロンプトテンプレートを埋め込みリソースとして提供します。
package assets

import (
	"embed"
	"fmt"

	"github.com/shouni/go-prompt-kit/resource"
)

const promptDir = "prompts"

var (
	// promptFiles はプロンプトテンプレートです。ディレクトリ内は現在プロンプトのみのため、
	// ファイル名のprefixは不要です（ファイル名がそのままモード名になります）。
	//go:embed prompts/*.md
	promptFiles embed.FS

	// partialFiles は、複数のプロンプトモードで共有するテキスト断片です。
	// prompts/ とは別ディレクトリに置き、レビューモードの一覧には含めません。
	//go:embed partials/*.md
	partialFiles embed.FS
)

// LoadPrompts は埋め込まれたプロンプトファイルを読み込みます。
func LoadPrompts() (map[string]string, error) {
	return resource.Load(promptFiles, promptDir, "")
}

// LoadFindingsFormat は、レビュー指摘のJSONフォーマットを説明する共通テキストを読み込みます。
// 全レビューモードのプロンプトで共有され、AIの構造化出力(findings配列)のスキーマに
// 対応する項目を説明します。
func LoadFindingsFormat() (string, error) {
	return loadPartial("findings_format.md")
}

// LoadVerdictFormat は、判定結果のJSONフォーマット(verdictオブジェクト)を説明する
// 共通テキストを読み込みます。
func LoadVerdictFormat() (string, error) {
	return loadPartial("verdict_format.md")
}

func loadPartial(name string) (string, error) {
	b, err := partialFiles.ReadFile("partials/" + name)
	if err != nil {
		return "", fmt.Errorf("共有テンプレート '%s' の読み込みに失敗: %w", name, err)
	}
	return string(b), nil
}
