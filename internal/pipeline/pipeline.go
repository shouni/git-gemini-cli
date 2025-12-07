package pipeline

import (
	"context"
	"fmt"

	"git-gemini-cli/internal/builder"
	"git-gemini-cli/internal/config"
	"log/slog"

	"github.com/shouni/go-utils/urlpath"
)

// ExecuteReviewPipeline は、すべての依存関係を構築し、レビューパイプラインを実行します。
// 実行結果の文字列とエラーを返します。
func ExecuteReviewPipeline(
	ctx context.Context,
	cfg config.ReviewConfig,
) (string, error) {
	const baseRepoDirName = "reviewerRepos"

	// LocalPathが指定されていない場合、RepoURLから動的に生成しcfgを更新します。
	if cfg.LocalPath == "" {
		cfg.LocalPath = urlpath.SanitizeURLToUniquePath(cfg.RepoURL, baseRepoDirName)
		slog.Debug("LocalPathが未指定のため、URLから動的にパスを生成しました。", "generatedPath", cfg.LocalPath)
	}

	reviewRunner, err := builder.BuildReviewRunner(ctx, cfg)
	if err != nil {
		// BuildReviewRunner が内部でアダプタやビルダーの構築エラーをラップして返す
		return "", fmt.Errorf("レビュー実行器の構築に失敗しました: %w", err)
	}

	slog.Info("レビューパイプラインを開始します。")

	reviewResult, err := reviewRunner.Run(ctx, cfg)
	if err != nil {
		return "", err
	}

	if reviewResult == "" {
		slog.Info("Diff がないためレビューをスキップしました。")
		return "", nil
	}

	return reviewResult, nil
}

// ExecutePublishPipeline は、すべての依存関係を構築し、パブリッシュパイプラインを実行します。
func ExecutePublishPipeline(
	ctx context.Context,
	cfg config.PublishConfig,
	reviewResult string,
) error {

	// 1. クラウドストレージに保存し、そのURLを通知
	publishRunner, err := builder.BuildPublishRunner(ctx, cfg)
	if err != nil {
		return fmt.Errorf("PublishRunnerの構築に失敗しました: %w", err)
	}
	err = publishRunner.Run(ctx, cfg, reviewResult)
	if err != nil {
		return fmt.Errorf("公開処理の実行に失敗しました: %w", err)
	}

	return nil
}
