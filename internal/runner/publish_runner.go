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

// PublisherRunner は、公開/通知を行うコアな実行主体を定義します。
type PublisherRunner interface {
	Run(
		ctx context.Context,
		cfg config.ReviewConfig,
		targetURI string,
		reviewResult string,
		slackWebhookUrl string,
	) error
}

// CorePublisherRunner は CorePublisherRunner インターフェースの具体的な実装です。
type CorePublisherRunner struct {
	httpClient httpkit.ClientInterface
}

// NewCorePublisherRunner は NewCorePublisherRunner のインスタンスを構築します。
func NewCorePublisherRunner(httpkit httpkit.ClientInterface) *CorePublisherRunner {
	return &CorePublisherRunner{
		httpClient: httpkit,
	}
}

// Run は公開処理のパイプライン全体を実行します。
func (p *CorePublisherRunner) Run(
	ctx context.Context,
	cfg config.ReviewConfig,
	targetURI string,
	reviewResult string,
	slackWebhookUrl string,
) error {

	// マルチクラウド対応ファクトリの利用
	writer, urlSigner, err := publisher.NewPublisherAndSigner(ctx, targetURI)
	if err != nil {
		return err // 初期化に失敗したら即座にエラーを返す
	}

	// 結果のPublish
	meta := publisher.ReviewData{
		RepoURL:        cfg.RepoURL,
		BaseBranch:     cfg.BaseBranch,
		FeatureBranch:  cfg.FeatureBranch,
		ReviewMarkdown: reviewResult,
	}
	err = writer.Publish(ctx, targetURI, meta)
	if err != nil {
		return fmt.Errorf("ストレージへの書き込みに失敗しました (URI: %s): %w", targetURI, err)
	}
	slog.Info("クラウドストレージへのアップロードが完了しました。", "uri", targetURI)

	// Slack通知
	slackNotifier := adapters.NewSlackAdapter(p.httpClient, urlSigner, slackWebhookUrl)
	slog.Debug("SlackNotifierを構築しました。", "adapter_type", "adapters")
	if err := slackNotifier.Notify(ctx, targetURI, cfg); err != nil {
		// 🚨 ポリシー: Slack通知は二次的な機能であるため、アップロード成功後はエラーを返さない。
		slog.Error("Slack通知の実行中にエラーが発生しましたが、アップロードは成功しているため処理を続行します。", "error", err)
	}

	return nil
}
