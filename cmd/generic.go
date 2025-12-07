package cmd

import (
	"fmt"
	"log/slog"

	"git-gemini-cli/internal/pipeline"

	"github.com/spf13/cobra"
)

// genericCmd は 'generic' サブコマンドを定義します。
var genericCmd = &cobra.Command{
	Use:   "generic",
	Short: "コードレビューを実行し、その結果を標準出力に出力します。",
	Long:  `このコマンドは、指定されたGitリポジトリのブランチ間の差分をAIでレビューし、その結果を標準出力に直接表示します。外部サービスとの連携は行いません。`,
	Args:  cobra.NoArgs,
	RunE:  genericCommand,
}

func init() {
	genericCmd.MarkPersistentFlagRequired("repo-url")
	genericCmd.MarkPersistentFlagRequired("feature-branch")
}

// --------------------------------------------------------------------------
// コマンドの実行ロジック
// --------------------------------------------------------------------------

// genericCommand は generic コマンドの実行ロジックです。
func genericCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// 1. パイプラインを実行し、結果を受け取る
	reviewResult, err := pipeline.ExecuteReviewPipeline(ctx, ReviewConfig)
	if err != nil {
		return err
	}

	// 2. レビュー結果の出力、レビュー結果の内容が空でない場合にのみ標準出力に出力する
	if reviewResult != "" {
		printReviewResult(reviewResult)
		slog.Info("レビュー結果を標準出力に出力しました。")
	} else {
		slog.Info("レビュー結果の内容が空のため、標準出力への出力はスキップしました。")
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
