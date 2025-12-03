package cmd

import (
	"fmt"
	"log/slog"

	"github.com/shouni/gemini-reviewer-core/pkg/publisher"
	"github.com/shouni/go-remote-io/pkg/gcsfactory"
	"github.com/shouni/go-remote-io/pkg/remoteio"
	"github.com/shouni/go-remote-io/pkg/s3factory"

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
	reviewResult, err := executeReviewPipeline(ctx, ReviewConfig)
	if err != nil {
		return err
	}

	if reviewResult == "" {
		slog.Warn("レビュー結果の内容が空のため、ストレージへの保存をスキップします。", "uri", targetURI)
		return nil
	}

	// --- 2. マルチクラウド対応ファクトリの利用 ---

	// a. FactoryRegistryの構築（必要なFactoryのみを初期化）
	registry := publisher.FactoryRegistry{}

	// GCSまたはS3のどちらか必要なファクトリのみを初期化
	if remoteio.IsGCSURI(targetURI) {
		gcsFactory, err := gcsfactory.NewGCSClientFactory(ctx)
		if err != nil {
			return fmt.Errorf("GCSクライアントファクトリの初期化に失敗しました: %w", err)
		}
		registry.GCSFactory = gcsFactory
	} else if remoteio.IsS3URI(targetURI) {
		s3Factory, err := s3factory.NewS3ClientFactory(ctx)
		if err != nil {
			return fmt.Errorf("S3クライアントファクトリの初期化に失敗しました: %w", err)
		}
		registry.S3Factory = s3Factory
	}

	// b. Publisherの動的生成（URIスキーム判定とインスタンス生成を委譲）
	writer, err := publisher.NewPublisher(targetURI, registry)
	if err != nil {
		// publisher.NewPublisherでURIスキームがサポート外の場合もここでエラーになる
		return fmt.Errorf("パブリッシャーの初期化に失敗しました: %w", err)
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

	return nil
}
