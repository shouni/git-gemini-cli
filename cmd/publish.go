package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"git-gemini-reviewer-go/internal/adapters"
	"git-gemini-reviewer-go/internal/pipeline"

	"github.com/shouni/gemini-reviewer-core/pkg/publisher"
	"github.com/spf13/cobra"
)

// PublishFlags は GCS/S3 への公開フラグを保持します。
type PublishFlags struct {
	URI         string // 宛先URI (例: gs://bucket/..., s3://bucket/...)
	ContentType string // 保存する際のMIMEタイプ
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
	publishCmd.Flags().StringVarP(&publishFlags.ContentType, "content-type", "t", "text/html; charset=utf-8", "クラウドストレージに保存する際のMIMEタイプ")
	publishCmd.Flags().StringVarP(&publishFlags.URI, "uri", "s", "", "保存先のURI (例: gs://bucket/result.html, s3://bucket/result.html)")
	// URIフラグは必須にする
	publishCmd.MarkFlagRequired("uri")
}

// --------------------------------------------------------------------------
// コマンドの実行ロジック
// --------------------------------------------------------------------------

// publishCommand は publish コマンドの実行ロジックです。
func publishCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	targetURI := publishFlags.URI

	// 1. レビューパイプラインを実行 (ReviewConfigを渡す)
	reviewResult, err := pipeline.ExecuteReviewPipeline(ctx, ReviewConfig)
	if err != nil {
		return err
	}

	if reviewResult == "" {
		slog.Warn("レビュー結果の内容が空のため、ストレージへの保存をスキップします。", "uri", targetURI)
		return nil
	}

	// --- 2. マルチクラウド対応ファクトリの利用 ---
	writer, urlSigner, err := publisher.NewPublisherAndSigner(ctx, targetURI)
	if err != nil {
		return err // 初期化に失敗したら即座にエラーを返す
	}

	// 3. 結果のPublish
	meta := publisher.ReviewData{
		RepoURL:        ReviewConfig.RepoURL,
		BaseBranch:     ReviewConfig.BaseBranch,
		FeatureBranch:  ReviewConfig.FeatureBranch,
		ReviewMarkdown: reviewResult,
	}
	err = writer.Publish(ctx, publishFlags.URI, meta)
	if err != nil {
		return fmt.Errorf("ストレージへの書き込みに失敗しました (URI: %s): %w", publishFlags.URI, err)
	}
	slog.Info("クラウドストレージへのアップロードが完了しました。", "uri", publishFlags.URI)

	// --- 4. Slack通知 ---
	httpClient, err := GetHTTPClient(ctx)
	if err != nil {
		return fmt.Errorf("HTTPクライアントの取得に失敗しました: %w", err)
	}
	slackNotifier := adapters.NewSlackAdapter(httpClient, urlSigner, os.Getenv("SLACK_WEBHOOK_URL"))
	slog.Debug("SlackNotifierを構築しました。", "adapter_type", "adapters")
	if err := slackNotifier.Notify(ctx, targetURI, ReviewConfig); err != nil {
		// 🚨 ポリシー: Slack通知は二次的な機能であるため、アップロード成功後はエラーを返さない。
		slog.Error("Slack通知の実行中にエラーが発生しましたが、アップロードは成功しているため処理を続行します。", "error", err)
	}

	return nil
}
