package builder

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/shouni/gemini-reviewer-core/pkg/prompts"
	"github.com/shouni/gemini-reviewer-core/pkg/publisher"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-remote-io/pkg/gcsfactory"
	"github.com/shouni/go-remote-io/pkg/remoteio"
	"github.com/shouni/go-remote-io/pkg/s3factory"

	"git-gemini-cli/internal/adapters"
	"git-gemini-cli/internal/config"
	"git-gemini-cli/internal/runner"
)

// BuildReviewRunner は、必要な依存関係をすべて構築し、ReviewRunner のインスタンスを返します。
func BuildReviewRunner(ctx context.Context, cfg config.ReviewConfig) (runner.ReviewRunner, error) {
	gitService := adapters.NewGitService(cfg)
	codeReviewAI, err := adapters.NewCodeReviewAI(ctx, cfg)
	if err != nil {
		return nil, err
	}

	promptBuilder, err := prompts.NewBuilder()
	if err != nil {
		return nil, fmt.Errorf("Prompt Builder の構築に失敗しました: %w", err)
	}

	reviewRunner := runner.NewDefaultReviewRunner(
		gitService,
		codeReviewAI,
		promptBuilder,
	)

	slog.Debug("ReviewRunner の構築が完了しました。")
	return reviewRunner, nil
}

// BuildPublishRunner は、必要な依存関係をすべて構築し、
// runner.PublisherRunner (インターフェース) を返します。
func BuildPublishRunner(ctx context.Context, cfg config.PublishConfig) (runner.PublisherRunner, error) {
	var ioFactory remoteio.IOFactory
	var err error

	// --- 構築失敗時のリソース解放用ロジック ---
	success := false
	defer func() {
		if !success {
			slog.Warn("PublishRunnerの構築中にエラーが発生したため、リソースをクリーンアップします。")
			_ = ioFactory.Close()
		}
	}()

	// 1. IOFactory の初期化
	switch {
	case remoteio.IsGCSURI(cfg.StorageURI):
		ioFactory, err = gcsfactory.New(ctx)
	case remoteio.IsS3URI(cfg.StorageURI):
		ioFactory, err = s3factory.New(ctx)
	default:
		return nil, fmt.Errorf("サポートされていないストレージURIスキームです: %s", cfg.StorageURI)
	}
	if err != nil {
		return nil, fmt.Errorf("IOFactoryの初期化に失敗しました: %w", err)
	}

	// 2. コンポーネントの構築
	writer, err := ioFactory.OutputWriter()
	if err != nil {
		return nil, fmt.Errorf("OutputWriter初期化に失敗しました: %w", err)
	}
	urlSigner, err := ioFactory.URLSigner()
	if err != nil {
		return nil, fmt.Errorf("URLSignerの初期化に失敗しました: %w", err)
	}
	htmlRunner, err := publisher.NewMarkdownToHtmlRunner(ctx)
	if err != nil {
		return nil, fmt.Errorf("MarkdownToHtmlRunnerの初期化に失敗しました: %w", err)
	}

	// HTTPクライアントの初期化
	httpClient := httpkit.New(config.DefaultHTTPTimeout)
	slackNotifier := adapters.NewSlackAdapter(
		httpClient,
		cfg.SlackWebhookURL,
	)

	reviewPublisher, err := publisher.NewPublisher(ctx, writer, htmlRunner)
	if err != nil {
		return nil, fmt.Errorf("Publisherの初期化に失敗しました (URI: %s): %w", cfg.StorageURI, err)
	}

	// 3. 依存関係を注入して Runner を組み立てる
	publisherRunner := runner.NewDefaultPublisherRunner(
		reviewPublisher,
		urlSigner,
		slackNotifier,
	)

	success = true
	slog.Debug("PublishRunner の構築が完了しました。")

	return publisherRunner, nil
}
