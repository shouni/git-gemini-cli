package domain

import (
	"context"

	core "github.com/shouni/gemini-reviewer-core/pkg/domain"
)

// Pipeline は、処理を行うインターフェースです。
type Pipeline interface {
	Execute(ctx context.Context, req ReviewRequest) error
	Review(ctx context.Context, req ReviewRequest) (string, error)
	Publish(ctx context.Context, req ReviewRequest) error
}

// ReviewRunner は、レビュー要求に対して実際のレビュー処理（分析等）をAIで実行するインターフェースです。
type ReviewRunner interface {
	Run(ctx context.Context, req ReviewRequest) (string, error)
}

// PublishRunner は、生成されたスクリプトの公開処理を実行する責務を持つインターフェースです。
type PublishRunner interface {
	Run(ctx context.Context, req ReviewRequest) error
}

// PromptBuilder は、プロンプト文字列を生成する責務を定義します。
type PromptBuilder interface {
	Build(mode string, data core.TemplateData) (string, error)
}

// Notifier は、生成されたコンテンツまたはエラーに関する通知を指定されたターゲットまたはチャネルに送信するためのインターフェイスです。
type Notifier interface {
	// Notify は、パブリック URL やストレージ URL などのメタデータを含む通知をターゲットに送信します。
	Notify(ctx context.Context, publicURL, storageURI string, req ReviewRequest) error
}
