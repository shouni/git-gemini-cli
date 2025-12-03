package pipeline

import (
	"context"
	"fmt"

	"git-gemini-reviewer-go/internal/builder"
	"git-gemini-reviewer-go/internal/config"
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

	// cfg.UseInternalGitAdapter = true は、Git操作に内部で実装されたアダプタを使用することを示します。
	// これは、os/exec を利用した外部コマンド実行を抽象化したものであり、
	// README.mdで説明されている「外部コマンド実行」の原則に沿っています。
	// この設定により、コアライブラリ側でGitコマンドの実行方法を抽象化し、
	// 将来的な拡張性やテスト容易性を向上させています。
	cfg.UseInternalGitAdapter = true
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
