package cmd

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"git-gemini-cli/internal/builder"
	"git-gemini-cli/internal/domain"
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
}

// --------------------------------------------------------------------------
// コマンドの実行ロジック
// --------------------------------------------------------------------------

// publishCommand は、AIによるレビュー結果を生成し、指定されたURIのクラウドストレージに
// 公開（アップロード）と通知を行う publish コマンドの実行ロジックです。
func publishCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	appCtx, err := builder.BuildContainer(ctx, &ReviewConfig)
	if err != nil {
		// コンテナの構築エラーをラップして返す
		return fmt.Errorf("コンテナの構築に失敗しました: %w", err)
	}
	defer func() {
		if closeErr := appCtx.Close(); closeErr != nil {
			slog.ErrorContext(ctx, "コンテナのクローズに失敗しました", "error", closeErr)
		}
	}()

	req := domain.ReviewRequest{
		Config:     ReviewConfig,
		StorageURI: publishFlags.URI,
	}

	if err := appCtx.Pipeline.Execute(ctx, req); err != nil {
		return fmt.Errorf("レビューおよび公開パイプラインの実行に失敗しました: %w", err)
	}

	slog.Info("処理完了", "uri", req.StorageURI)

	return nil
}
