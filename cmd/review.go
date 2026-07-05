package cmd

import (
	"fmt"
	"log/slog"

	"github.com/shouni/gemini-reviewer-core/ports"
	"github.com/spf13/cobra"

	"github.com/shouni/git-gemini-cli/internal/builder"
)

// reviewCmd は 'review' サブコマンドを定義します。
var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "コードレビューを実行し、その結果を標準出力に出力します。",
	Long:  `このコマンドは、指定されたGitリポジトリのブランチ間の差分をAIでレビューし、その結果を標準出力に直接表示します。外部サービスとの連携は行いません。`,
	Args:  cobra.NoArgs,
	RunE:  reviewCommand,
}

// --------------------------------------------------------------------------
// コマンドの実行ロジック
// --------------------------------------------------------------------------

// reviewCommand は、リモートリポジトリのブランチ比較を Gemini AI に依頼し、
// 結果を標準出力に出力する generic コマンドの実行ロジックです。
func reviewCommand(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	appCtx, err := builder.BuildContainer(ctx, &opts)
	if err != nil {
		return fmt.Errorf("アプリケーションコンテキストの構築に失敗しました: %w", err)
	}
	defer func() {
		slog.Info("♻️ アプリケーションコンテキストをクローズ中...")
		appCtx.Close()
	}()

	// 1. domain.ReviewRequest 定義に合わせてフィールドを埋める
	req := ports.ReviewRequest{
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
