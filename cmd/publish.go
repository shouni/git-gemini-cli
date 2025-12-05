package cmd

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"git-gemini-cli/internal/pipeline"
	"git-gemini-cli/internal/runner"
)

// PublishFlags は GCS/S3 への公開フラグを保持します。
type PublishFlags struct {
	URI string // 宛先URI (例: gs://bucket/..., s3://bucket/...)
}

var publishFlags PublishFlags

func init() {
	// フラグ名を汎用的なものに変更
	publishCmd.Flags().StringVarP(&publishFlags.URI, "uri", "s", "", "保存先のURI (例: gs://bucket/result.html, s3://bucket/result.html)")
	// URIフラグは必須にする
	publishCmd.MarkFlagRequired("uri")

	publishCmd.MarkPersistentFlagRequired("repo-url")
	publishCmd.MarkPersistentFlagRequired("feature-branch")
}

// publishCmd は 'publish' サブコマンドを定義します。
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "AIレビュー結果をHTMLに変換し、指定されたGCS/S3 URIに保存します。",
	Long:  `このコマンドは、AIレビュー結果をスタイル付きHTMLに変換した後、go-remote-io を利用してURIスキームに応じたクラウドストレージ（gs:// または s3://）にアップロードします。`,
	Args:  cobra.NoArgs,
	RunE:  publishCommand,
}

// --------------------------------------------------------------------------
// コマンドの実行ロジック
// --------------------------------------------------------------------------

// publishCommand は publish コマンドの実行ロジックです。
func publishCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// 1. レビューパイプラインを実行 (ReviewConfigを渡す)
	reviewResult, err := pipeline.ExecuteReviewPipeline(ctx, ReviewConfig)
	if err != nil {
		return err
	}

	if reviewResult == "" {
		slog.Warn("レビュー結果の内容が空のため、ストレージへの保存をスキップします。", "uri", publishFlags.URI)
		return nil
	}

	// 2. クラウドストレージに保存し、そのURLを通知
	httpClient, err := GetHTTPClient(ctx)
	if err != nil {
		return fmt.Errorf("HTTPクライアントの取得に失敗しました: %w", err)
	}
	publisherRunner := runner.NewCorePublisherRunner(httpClient)
	publishParams := runner.PublishParams{
		Config:          ReviewConfig,
		TargetURI:       publishFlags.URI,
		ReviewResult:    reviewResult,
		SlackWebhookURL: ReviewConfig.SlackWebhookURL,
	}
	err = publisherRunner.Run(ctx, publishParams)
	if err != nil {
		return fmt.Errorf("公開処理の実行に失敗しました: %w", err)
	}

	return nil
}
