package runner

import (
	"context"
	"fmt"
	"log/slog"

	"git-gemini-cli/internal/adapters"
	"git-gemini-cli/internal/config"

	"github.com/shouni/gemini-reviewer-core/pkg/publisher"
	"github.com/shouni/go-http-kit/pkg/httpkit"
)

// PublishParams は Run メソッドのパラメータをカプセル化します。
type PublishParams struct {
	Config          config.ReviewConfig
	TargetURI       string
	ReviewResult    string
	SlackWebhookURL string
}

// PublisherRunner は、レビュー結果の公開処理を実行する責務を持つインターフェースです。
type PublisherRunner interface {
	Run(ctx context.Context, params PublishParams) error
}

// CorePublisherRunner は PublisherRunner インターフェースの具体的な実装です。
type CorePublisherRunner struct {
	httpClient httpkit.ClientInterface
}

// NewCorePublisherRunner は CorePublisherRunner の新しいインスタンスを生成します。
func NewCorePublisherRunner(client httpkit.ClientInterface) *CorePublisherRunner {
	return &CorePublisherRunner{
		httpClient: client,
	}
}

// Run は公開処理のパイプライン全体を実行します。
func (p *CorePublisherRunner) Run(ctx context.Context, params PublishParams) error {

	// マルチクラウド対応ファクトリの利用
	writer, urlSigner, err := publisher.NewPublisherAndSigner(ctx, params.TargetURI)
	if err != nil {
		return err // 初期化に失敗したら即座にエラーを返す
	}

	// 結果のPublish
	meta := newReviewData(params.Config, params.ReviewResult)
	err = writer.Publish(ctx, params.TargetURI, meta)
	if err != nil {
		return fmt.Errorf("ストレージへの書き込みに失敗しました (URI: %s): %w", params.TargetURI, err)
	}
	slog.Info("クラウドストレージへのアップロードが完了しました。", "uri", params.TargetURI)

	// Slack通知 (Webhook URLが設定されている場合のみ実行)
	webhookURL := params.SlackWebhookURL
	if webhookURL != "" {
		slackNotifier := adapters.NewSlackAdapter(p.httpClient, urlSigner, webhookURL)
		slog.Debug("SlackNotifierを構築しました。", "adapter_type", "adapters")
		if err := slackNotifier.Notify(ctx, params.TargetURI, params.Config); err != nil {
			// 🚨 ポリシー: Slack通知は二次的な機能であるため、アップロード成功後はエラーを返さない。
			slog.Error("Slack通知の実行中にエラーが発生しましたが、アップロードは成功しているため処理を続行します。", "error", err)
		}
	} else {
		slog.Info("Slack Webhook URLが設定されていないため、通知をスキップしました。")
	}

	return nil
}

// newReviewData は設定とレビュー結果から publisher.ReviewData を生成します。
func newReviewData(cfg config.ReviewConfig, reviewResult string) publisher.ReviewData {
	return publisher.ReviewData{
		RepoURL:        cfg.RepoURL,
		BaseBranch:     cfg.BaseBranch,
		FeatureBranch:  cfg.FeatureBranch,
		ReviewMarkdown: reviewResult,
	}
}
