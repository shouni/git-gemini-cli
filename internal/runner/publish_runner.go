package runner

import (
	"context"
	"fmt"
	"log/slog"

	"git-gemini-cli/internal/adapters"
	"git-gemini-cli/internal/config"

	"github.com/shouni/gemini-reviewer-core/pkg/publisher"
)

// PublisherRunner は、レビュー結果の公開処理を実行する責務を持つインターフェースです。
type PublisherRunner interface {
	Run(ctx context.Context, cfg config.PublishConfig) error
}

// CorePublisherRunner は、レビュー結果の公開処理を実行する具象構造体です。
type CorePublisherRunner struct {
	writer        publisher.Publisher
	slackNotifier adapters.SlackNotifier
}

// NewCorePublisherRunner は CorePublisherRunner の新しいインスタンスを作成します。
func NewCorePublisherRunner(writer publisher.Publisher, slackNotifier adapters.SlackNotifier) *CorePublisherRunner {
	return &CorePublisherRunner{
		writer:        writer,
		slackNotifier: slackNotifier,
	}
}

// Run は公開処理のパイプライン全体を実行します。
func (p *CorePublisherRunner) Run(ctx context.Context, cfg config.PublishConfig) error {
	meta := newReviewData(cfg)
	err := p.writer.Publish(ctx, cfg.TargetURI, meta)
	if err != nil {
		return fmt.Errorf("ストレージへの書き込みに失敗しました (URI: %s): %w", cfg.TargetURI, err)
	}
	slog.Info("クラウドストレージへのアップロードが完了しました。", "uri", cfg.TargetURI)

	if err := p.slackNotifier.Notify(ctx, cfg.TargetURI, cfg.ReviewConfig); err != nil {
		// 🚨 ポリシー: Slack通知は二次的な機能であるため、アップロード成功後はエラーを返さない。
		slog.Error("Slack通知の実行中にエラーが発生しましたが、アップロードは成功しているため処理を続行します。", "error", err)
	}

	return nil
}

// newReviewData は設定とレビュー結果から publisher.ReviewData を生成します。
func newReviewData(cfg config.PublishConfig) publisher.ReviewData {
	return publisher.ReviewData{
		RepoURL:        cfg.ReviewConfig.RepoURL,
		BaseBranch:     cfg.ReviewConfig.BaseBranch,
		FeatureBranch:  cfg.ReviewConfig.FeatureBranch,
		ReviewMarkdown: cfg.ReviewResult,
	}
}
