package cmd

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"git-gemini-cli/internal/builder"
	"git-gemini-cli/internal/domain"
)

// genericCmd は 'generic' サブコマンドを定義します。
var genericCmd = &cobra.Command{
	Use:   "generic",
	Short: "コードレビューを実行し、その結果を標準出力に出力します。",
	Long:  `このコマンドは、指定されたGitリポジトリのブランチ間の差分をAIでレビューし、その結果を標準出力に直接表示します。外部サービスとの連携は行いません。`,
	Args:  cobra.NoArgs,
	RunE:  genericCommand,
}

// --------------------------------------------------------------------------
// コマンドの実行ロジック
// --------------------------------------------------------------------------

// genericCommand は、リモートリポジトリのブランチ比較を Gemini AI に依頼し、
// 結果を標準出力に出力する generic コマンドの実行ロジックです。
func genericCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	appCtx, err := builder.BuildContainer(ctx, &opts)
	if err != nil {
		return fmt.Errorf("アプリケーションコンテキストの構築に失敗しました: %w", err)
	}
	defer func() {
		slog.Info("♻️ アプリケーションコンテキストをクローズ中...")
		appCtx.Close()
	}()

	// 1. パイプラインを実行し、結果を受け取る
	req := domain.ReviewRequest{
		RepoURL:       opts.RepoURL,
		BaseBranch:    opts.BaseBranch,
		FeatureBranch: opts.FeatureBranch,
		Mode:          opts.ReviewMode,
		ModelName:     opts.GeminiModel,
	}

	// ReviewPipeline.Review は中間結果の Outcome を返します
	outcome := appCtx.Pipeline.Review(ctx, req)
	if outcome.Error != nil {
		return fmt.Errorf("review process failed at step %s: %w", outcome.StepName, outcome.Error)
	}

	// 2. レビュー結果の出力
	// ReviewMarkdown が空でない場合にのみ標準出力に出力する
	if outcome.ReviewMarkdown != "" {
		printReviewResult(outcome.ReviewMarkdown)
		slog.Info("レビュー結果を標準出力に出力しました。")
	} else if outcome.IsSkipped {
		slog.Info("差分がないため、レビュー出力はスキップされました。")
	}

	return nil
}

// printReviewResult は noPost 時に結果を標準出力します。
func printReviewResult(result string) {
	// 標準出力 (fmt.Println) は維持
	fmt.Println("\n--- Gemini AI レビュー結果 ---")
	fmt.Println(result)
	fmt.Println("-----------------------------------------------------")
}
