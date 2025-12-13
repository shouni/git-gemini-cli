package builder

import (
	"context"
	"fmt"
	"log/slog"

	"git-gemini-cli/internal/config"
	"git-gemini-cli/internal/runner"

	internalAdapters "git-gemini-cli/internal/adapters"

	"github.com/shouni/gemini-reviewer-core/pkg/adapters"
	"github.com/shouni/gemini-reviewer-core/pkg/prompts"
	"github.com/shouni/gemini-reviewer-core/pkg/publisher"
)

// buildGitService は adapters.GitService のインスタンスを構築する Factory 関数です。
// 設定 (cfg.UseExternalGitCommand) に基づいて、内部アダプタ (os/exec) またはコアライブラリのアダプタ (go-git) を選択します。
func buildGitService(cfg config.ReviewConfig) adapters.GitService {
	// フラグが true の場合、CLI固有の内部アダプタ (os/execベース) を使用
	if cfg.UseExternalGitCommand {
		slog.Debug("GitService: 外部Gitコマンド利用アダプタ (LocalGitAdapter/os/exec) を使用します。")
		return internalAdapters.NewLocalGitAdapter(
			cfg.LocalPath,
			cfg.SSHKeyPath,
			internalAdapters.WithInsecureSkipHostKeyCheck(cfg.SkipHostKeyCheck),
			internalAdapters.WithBaseBranch(cfg.BaseBranch),
		)
	}

	// フラグが false または未設定の場合、コアライブラリのアダプタ (go-gitベース) を使用
	slog.Debug("GitService: コアライブラリのアダプタ (go-git) を使用します。")
	return adapters.NewGitAdapter(
		cfg.LocalPath,
		cfg.SSHKeyPath,
		adapters.WithInsecureSkipHostKeyCheck(cfg.SkipHostKeyCheck),
		adapters.WithBaseBranch(cfg.BaseBranch),
	)
}

// buildGeminiService は adapters.CodeReviewAI のインスタンスを構築します。
// この関数は BuildReviewRunner の内部ヘルパーとして使用されます。
func buildGeminiService(ctx context.Context, cfg config.ReviewConfig) (adapters.CodeReviewAI, error) {
	geminiService, err := adapters.NewGeminiAdapter(ctx, cfg.GeminiModel)
	if err != nil {
		return nil, fmt.Errorf("Gemini Service の構築に失敗しました: %w", err)
	}

	return geminiService, nil
}

// BuildReviewRunner は、必要な依存関係をすべて構築し、
// 実行可能な ReviewRunner のインスタンスを返します。
func BuildReviewRunner(ctx context.Context, cfg config.ReviewConfig) (runner.ReviewRunner, error) {
	// 1. GitService の構築
	gitService := buildGitService(cfg)
	slog.Debug("GitService (Adapter) を構築しました。",
		slog.String("local_path", cfg.LocalPath),
		slog.String("base_branch", cfg.BaseBranch),
	)

	// 2. GeminiService の構築
	geminiService, err := buildGeminiService(ctx, cfg)
	if err != nil {
		return nil, err
	}
	slog.Debug("GeminiService (Adapter) を構築しました。", slog.String("model", cfg.GeminiModel))

	// 3. Prompt Builder の構築
	promptBuilder, err := prompts.NewPromptBuilder()
	if err != nil {
		return nil, fmt.Errorf("Prompt Builder の構築に失敗しました: %w", err)
	}
	slog.Debug("PromptBuilderを構築しました。", slog.String("component", "PromptBuilder"))

	// 4. 依存関係を注入して Runner を組み立てる
	reviewRunner := runner.NewCoreReviewRunner(
		gitService,
		geminiService,
		promptBuilder,
	)

	slog.Debug("ReviewRunner の構築が完了しました。")
	return reviewRunner, nil
}

// BuildPublishRunner は、必要な依存関係をすべて構築し、
// runner.PublisherRunner (インターフェース) を返します。
func BuildPublishRunner(ctx context.Context, cfg config.PublishConfig) (runner.PublisherRunner, error) {

	// 1. PublisherとSignerの初期化 (マルチクラウド対応)
	writer, urlSigner, err := publisher.NewPublisherAndSigner(ctx, cfg.StorageURI)
	if err != nil {
		return nil, fmt.Errorf("Publisherの初期化に失敗しました (URI: %s): %w", cfg.StorageURI, err)
	}

	// 2. Slackアダプターの構築
	slackNotifier := internalAdapters.NewSlackAdapter(
		cfg.HttpClient,
		cfg.SlackWebhookURL,
	)

	// 3. 依存関係を注入して Runner を組み立てる
	publicRunner := runner.NewCorePublisherRunner(
		writer,
		urlSigner,
		slackNotifier,
	)
	slog.Debug("PublishRunner の構築が完了しました。")

	return publicRunner, nil
}
