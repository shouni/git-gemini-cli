package builder

import (
	"context"
	"fmt"
	"log/slog"

	"git-gemini-cli/internal/adapters"
	"git-gemini-cli/internal/config"
	"git-gemini-cli/internal/runner"

	core "github.com/shouni/gemini-reviewer-core/pkg/adapters"
	"github.com/shouni/gemini-reviewer-core/pkg/prompts"
	"github.com/shouni/gemini-reviewer-core/pkg/publisher"
	"github.com/shouni/go-remote-io/pkg/gcsfactory"
	"github.com/shouni/go-remote-io/pkg/remoteio"
	"github.com/shouni/go-remote-io/pkg/s3factory"
)

// buildGitService は adapters.GitService のインスタンスを構築する Factory 関数です。
func buildGitService(cfg config.ReviewConfig) core.GitService {
	if cfg.UseExternalGitCommand {
		slog.Debug("GitService: 外部Gitコマンド利用アダプタ (LocalGitAdapter/os/exec) を使用します。")
		return adapters.NewLocalGitAdapter(
			cfg.LocalPath,
			cfg.SSHKeyPath,
			adapters.WithInsecureSkipHostKeyCheck(cfg.SkipHostKeyCheck),
			adapters.WithBaseBranch(cfg.BaseBranch),
		)
	}

	slog.Debug("GitService: コアライブラリのアダプタ (go-git) を使用します。")
	return core.NewGitAdapter(
		cfg.LocalPath,
		cfg.SSHKeyPath,
		core.WithInsecureSkipHostKeyCheck(cfg.SkipHostKeyCheck),
		core.WithBaseBranch(cfg.BaseBranch),
	)
}

// buildCodeReviewAI は adapters.CodeReviewAI のインスタンスを構築します。
func buildCodeReviewAI(ctx context.Context, cfg config.ReviewConfig) (core.CodeReviewAI, error) {
	codeReviewAI, err := core.NewGeminiAdapter(ctx, cfg.GeminiModel)
	if err != nil {
		return nil, fmt.Errorf("CodeReviewAIアダプターの構築に失敗しました: %w", err)
	}
	return codeReviewAI, nil
}

// BuildReviewRunner は、必要な依存関係をすべて構築し、ReviewRunner のインスタンスを返します。
func BuildReviewRunner(ctx context.Context, cfg config.ReviewConfig) (runner.ReviewRunner, error) {
	gitService := buildGitService(cfg)

	codeReviewAI, err := buildCodeReviewAI(ctx, cfg)
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
	htmlRunner, err := publisher.NewMarkdownToHtmlRunner(ctx)
	if err != nil {
		return nil, fmt.Errorf("MarkdownToHtmlRunnerの初期化に失敗しました: %w", err)
	}

	reviewPublisher, err := publisher.NewPublisher(ctx, ioFactory, htmlRunner)
	if err != nil {
		return nil, fmt.Errorf("Publisherの初期化に失敗しました (URI: %s): %w", cfg.StorageURI, err)
	}

	// --- 構築失敗時のリソース解放用ロジック ---
	success := false
	defer func() {
		if !success {
			slog.Warn("PublishRunnerの構築中にエラーが発生したため、リソースをクリーンアップします。")
			_ = reviewPublisher.Close()
		}
	}()

	urlSigner, err := ioFactory.URLSigner()
	if err != nil {
		return nil, fmt.Errorf("URLSignerの初期化に失敗しました: %w", err)
	}

	// 3. 依存関係を注入して Runner を組み立てる
	slackNotifier := adapters.NewSlackAdapter(
		cfg.HttpClient,
		cfg.SlackWebhookURL,
	)

	publisherRunner := runner.NewDefaultPublisherRunner(
		reviewPublisher,
		urlSigner,
		slackNotifier,
	)

	// すべて成功したため、defer での Close をスキップ
	success = true
	slog.Debug("PublishRunner の構築が完了しました。")

	return publisherRunner, nil
}
