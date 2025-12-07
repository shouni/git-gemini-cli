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
	Run(ctx context.Context, cfg config.PublishConfig, reviewResult string) error
}

// CorePublisherRunner は、レビュー結果の公開処理を実行する具象構造体です。
// 依存関係（writer, slackNotifier）をDIコンテナ/builderから注入することに専念します。
type CorePublisherRunner struct {
	writer        publisher.Publisher
	slackNotifier adapters.SlackNotifier
}

// NewCorePublisherRunner は CorePublisherRunner の新しいインスタンスを作成します。
// DIコンテナ/builderはこの関数を利用して依存関係を構築します。
func NewCorePublisherRunner(writer publisher.Publisher, slackNotifier adapters.SlackNotifier) *CorePublisherRunner {
	return &CorePublisherRunner{
		writer:        writer,
		slackNotifier: slackNotifier,
	}
}

// Run は公開処理のパイプライン全体を実行します。
// このメソッドは、処理のオーケストレーションに専念します。
func (p *CorePublisherRunner) Run(ctx context.Context, cfg config.PublishConfig, reviewResult string) error {
	// 1. ストレージへのアップロード処理
	if err := p.publishToStorage(ctx, cfg, reviewResult); err != nil {
		return err
	}

	// 2. Slack通知処理 (アップロード成功後のみ実行)
	p.notifyToSlack(ctx, cfg)

	return nil
}

// --- プライベートメソッドへの分割 ---

// publishToStorage はレビュー結果をクラウドストレージにアップロードします。
func (p *CorePublisherRunner) publishToStorage(ctx context.Context, cfg config.PublishConfig, reviewResult string) error {
	meta := createReviewData(cfg.ReviewConfig, reviewResult)
	if err := p.writer.Publish(ctx, cfg.StorageURI, meta); err != nil {
		return fmt.Errorf("ストレージへの書き込みに失敗しました (URI: %s): %w", cfg.StorageURI, err)
	}

	slog.Info("クラウドストレージへのアップロードが完了しました。", "uri", cfg.StorageURI)
	return nil
}

// notifyToSlack はSlackに通知を送信します。
func (p *CorePublisherRunner) notifyToSlack(ctx context.Context, cfg config.PublishConfig) {
	if err := p.slackNotifier.Notify(ctx, cfg.StorageURI, cfg.ReviewConfig); err != nil {
		// 🚨 ポリシー: Slack通知は二次的な機能であるため、アップロード成功後はエラーを返さない。
		slog.Error("Slack通知の実行中にエラーが発生しましたが、アップロードは成功しているため処理を続行します。", "error", err)
	}
}

// createReviewData は設定とレビュー結果から publisher.ReviewData を生成します。
func createReviewData(reviewCfg config.ReviewConfig, reviewResult string) publisher.ReviewData {
	return publisher.ReviewData{
		RepoURL:        reviewCfg.RepoURL,
		BaseBranch:     reviewCfg.BaseBranch,
		FeatureBranch:  reviewCfg.FeatureBranch,
		ReviewMarkdown: reviewResult,
	}
}
