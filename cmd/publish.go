package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"git-gemini-cli/internal/config"
	"git-gemini-cli/internal/pipeline"
)

// PublishFlags は GCS/S3 への公開フラグを保持します。
type PublishFlags struct {
	URI string // 宛先URI (例: gs://bucket/..., s3://bucket/...)
}

var publishFlags PublishFlags

// publishCmd は 'publish' サブコマンドを定義します。
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "AIレビュー結果をHTMLに変換し、指定されたGCS/S3 URIに保存します。",
	Long:  `このコマンドは、AIレビュー結果をスタイル付きHTMLに変換した後、go-remote-io を利用してURIスキームに応じたクラウドストレージ（gs:// または s3://）にアップロードします。`,
	Args:  cobra.NoArgs,
	RunE:  publishCommand,
}

func init() {
	// フラグ名を汎用的なものに変更
	publishCmd.Flags().StringVarP(&publishFlags.URI, "uri", "s", "", "保存先のURI (例: gs://bucket/result.html, s3://bucket/result.html)")
	// URIフラグは必須にする
	publishCmd.MarkFlagRequired("uri")

	publishCmd.MarkPersistentFlagRequired("repo-url")
	publishCmd.MarkPersistentFlagRequired("feature-branch")
}

// --------------------------------------------------------------------------
// コマンドの実行ロジック
// --------------------------------------------------------------------------

// publishCommand は、AIによるレビュー結果を生成し、指定されたURIのクラウドストレージに
// 公開（アップロード）と通知を行う publish コマンドの実行ロジックです。
func publishCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// 1. パイプラインを実行し、結果を受け取る
	reviewResult, err := pipeline.ExecuteReviewPipeline(ctx, ReviewConfig)
	if err != nil {
		return err
	}
	if reviewResult == "" {
		slog.Warn("レビュー結果の内容が空のため、ストレージへの保存をスキップします。", "uri", publishFlags.URI)
		return nil
	}

	httpClient, err := GetHTTPClient(ctx)
	if err != nil {
		return fmt.Errorf("HTTPクライアントの取得に失敗しました: %w", err)
	}

	// 2. 公開パイプラインを実行し、結果を受け取る
	err = pipeline.ExecutePublishPipeline(
		ctx,
		config.PublishConfig{
			HttpClient:      httpClient,
			ReviewConfig:    ReviewConfig,
			StorageURI:      publishFlags.URI,
			SlackWebhookURL: os.Getenv("SLACK_WEBHOOK_URL"),
		},
		reviewResult,
	)
	// 2. パイプライン実行中にエラーが発生した場合、それを呼び出し元に返す
	if err != nil {
		return fmt.Errorf("公開パイプラインの実行に失敗しました: %w", err)
	}
	slog.Info("処理完了", "uri", publishFlags.URI)

	return nil
}
