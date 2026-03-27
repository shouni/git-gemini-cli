package cmd

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"git-gemini-cli/internal/builder"
	"git-gemini-cli/internal/domain"
)

// publishCmd は 'publish' サブコマンドを定義します。
var publishCmd = &cobra.Command{
	Use:   "publish",
	Short: "AIレビュー結果をHTMLに変換し、指定されたGCS/S3 URIに保存します。",
	Long:  `このコマンドは、AIレビュー結果をスタイル付きHTMLに変換した後、go-remote-io を利用してURIスキームに応じたクラウドストレージ（gs:// または s3://）にアップロードします。`,
	Args:  cobra.NoArgs,
	RunE:  publishCommand,
}

func init() {
	publishCmd.Flags().StringVar(&opts.GCSBucket, "bucket", "", "保存先のGCSバケット名")
	publishCmd.Flags().StringVar(&opts.GCSPath, "path", "", "バケット内の保存パス (例: reports/rev_01.md)")

	publishCmd.MarkFlagRequired("bucket")
	publishCmd.MarkFlagRequired("path")
}

// --------------------------------------------------------------------------
// コマンドの実行ロジック
// --------------------------------------------------------------------------

// publishCommand は、AIによるレビュー結果を生成し、指定されたURIのクラウドストレージに
// 公開（アップロード）と通知を行う publish コマンドの実行ロジックです。
func publishCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	appCtx, err := builder.BuildContainer(ctx, &opts)
	if err != nil {
		return fmt.Errorf("アプリケーションコンテキストの構築に失敗しました: %w", err)
	}
	defer func() {
		slog.Info("♻️ アプリケーションコンテキストをクローズ中...")
		appCtx.Close()
	}()

	// 1. 最新の domain.ReviewRequest 定義に合わせてフィールドを埋める
	req := domain.ReviewRequest{
		RepoURL:       opts.RepoURL,
		BaseBranch:    opts.BaseBranch,
		FeatureBranch: opts.FeatureBranch,
		Mode:          opts.ReviewMode,
		ModelName:     opts.GeminiModel,
		GCSBucket:     opts.GCSBucket,
		GCSPath:       opts.GCSPath,
	}

	// 2. パイプラインの実行（Execute は error を返します）
	if err := appCtx.Pipeline.Execute(ctx, req); err != nil {
		return fmt.Errorf("レビューおよび公開パイプラインの実行に失敗しました: %w", err)
	}

	// 3. 完了ログを出力（req.GCSURI() メソッドを使用してフルパスを表示）
	slog.Info("処理完了", "uri", req.GCSURI())

	return nil
}
