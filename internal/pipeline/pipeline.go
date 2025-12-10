package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"git-gemini-cli/internal/builder"
	"git-gemini-cli/internal/config"
)

// ErrSkipReview は、レビュー対象の差分が存在しないためにパイプラインがスキップされたことを示すエラーです。
var ErrSkipReview = errors.New("差分が見つからなかったためレビューをスキップしました")

// ExecuteReviewPipeline は、すべての依存関係を構築し、レビューパイプラインを実行します。
// 実行結果の文字列とエラーを返します。
func ExecuteReviewPipeline(
	ctx context.Context,
	cfg config.ReviewConfig,
) (string, error) {

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
		slog.Info(ErrSkipReview.Error())
		return "", ErrSkipReview
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
