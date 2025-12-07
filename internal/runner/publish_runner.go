package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"git-gemini-cli/internal/adapters"
	"git-gemini-cli/internal/config"

	"github.com/shouni/gemini-reviewer-core/pkg/publisher"
	"github.com/shouni/go-remote-io/pkg/remoteio"
)

const (
	// signedURLExpiration は署名付きURLの有効期限を定義します。
	signedURLExpiration = 30 * time.Minute
)

// PublisherRunner は、レビュー結果の公開処理を実行する責務を持つインターフェースです。
type PublisherRunner interface {
	Run(ctx context.Context, cfg config.PublishConfig, reviewResult string) error
}

// CorePublisherRunner は、レビュー結果の公開処理を実行する具象構造体です。
// 依存関係（writer, slackNotifier）をDIコンテナ/builderから注入することに専念します。
type CorePublisherRunner struct {
	writer        publisher.Publisher
	urlSigner     remoteio.URLSigner
	slackNotifier adapters.SlackNotifier
}

// NewCorePublisherRunner は CorePublisherRunner の新しいインスタンスを作成します。
// DIコンテナ/builderはこの関数を利用して依存関係を構築します。
func NewCorePublisherRunner(writer publisher.Publisher, urlSigner remoteio.URLSigner, slackNotifier adapters.SlackNotifier) *CorePublisherRunner {
	return &CorePublisherRunner{
		writer:        writer,
		urlSigner:     urlSigner,
		slackNotifier: slackNotifier,
	}
}

// Run は公開処理のパイプライン全体を実行します。
// このメソッドは、処理のオーケストレーションに専念します。
func (p *CorePublisherRunner) Run(ctx context.Context, cfg config.PublishConfig, reviewResult string) error {
	// 1. ストレージへのアップロード処理
	if err := p.publishToStorage(ctx, cfg, reviewResult); err != nil {
		return err
	}

	// 2. 公開URLの生成 (Slack通知の前に行う)
	publicURL, err := p.getPublicURL(ctx, cfg.StorageURI)
	if err != nil {
		// URL署名/変換が失敗しても処理は続行可能だが、エラーを記録
		slog.Warn("公開URLの生成に失敗しました。署名なし/静的URIで通知を試みます。", "error", err, "uri", cfg.StorageURI)
		// 失敗した場合、そのままの StorageURI を publicURL としてフォールバック
		publicURL = cfg.StorageURI
	}

	// 3. Slack通知処理 (アップロード成功後、publicURLを使って実行)
	p.notifyToSlack(ctx, publicURL, cfg)

	return nil
}

// --- プライベートメソッドへの分割 ---

// publishToStorage はレビュー結果をクラウドストレージにアップロードします。
func (p *CorePublisherRunner) publishToStorage(ctx context.Context, cfg config.PublishConfig, reviewResult string) error {
	meta := createReviewData(cfg.ReviewConfig, reviewResult)
	if err := p.writer.Publish(ctx, cfg.StorageURI, meta); err != nil {
		return fmt.Errorf("ストレージへの書き込みに失敗しました (URI: %s): %w", cfg.StorageURI, err)
	}

	slog.Info("クラウドストレージへのアップロードが完了しました。", "uri", cfg.StorageURI)
	return nil
}

// notifyToSlack はSlackに通知を送信します。
func (p *CorePublisherRunner) notifyToSlack(ctx context.Context, publicURL string, cfg config.PublishConfig) {
	if err := p.slackNotifier.Notify(ctx, publicURL, cfg.StorageURI, cfg.ReviewConfig); err != nil {
		// 🚨 ポリシー: Slack通知は二次的な機能であるため、アップロード成功後はエラーを返さない。
		slog.Error("Slack通知の実行中にエラーが発生しましたが、アップロードは成功しているため処理を続行します。", "error", err)
	}
}

// getPublicURL は URI に応じて署名付きURLを生成するか、公開URLに変換します。
func (p *CorePublisherRunner) getPublicURL(ctx context.Context, storageURI string) (string, error) {
	if p.urlSigner == nil {
		// urlSignerがnilの場合、URIは署名が必要ないか、サポートされていないスキームです。
		slog.Debug("URL Signerがnilです。静的なURI変換のみを試みます。", "uri", storageURI)
	}

	// GCSの場合: 署名付きURLを生成
	if remoteio.IsGCSURI(storageURI) {
		if p.urlSigner == nil {
			return "", fmt.Errorf("GCS URIが指定されましたが、URL Signerがnilです。")
		}

		signedURL, err := p.urlSigner.GenerateSignedURL(ctx, storageURI, "GET", signedURLExpiration)
		if err != nil {
			return "", fmt.Errorf("GCS 署名付きURLの生成に失敗しました: %w", err)
		}
		slog.Info("GCS 署名付きURLの生成に成功", "url", signedURL)
		return signedURL, nil
	}

	// S3の場合: 静的な公開URL形式に変換
	if remoteio.IsS3URI(storageURI) {
		awsRegion := os.Getenv("AWS_REGION")
		if awsRegion == "" {
			awsRegion = "ap-northeast-1" // フォールバック
		}
		publicURL := convertS3URIToPublicURL(storageURI, awsRegion)
		slog.Info("S3 公開URLへの変換に成功", "url", publicURL)
		return publicURL, nil
	}

	// その他: 署名や変換が不要なURI (例: ローカルファイル、未サポートのプロバイダ)
	slog.Debug("静的な公開URL変換や署名が不要なURIです。", "uri", storageURI)
	return storageURI, nil
}

// convertS3URIToPublicURL は S3 URI を AWS の公開 Virtual-Hosted Style アクセス URL に変換します。
// 形式: https://{bucketName}.s3.{region}.amazonaws.com/{objectKey}
func convertS3URIToPublicURL(s3URI, region string) string {
	processedURI := strings.TrimPrefix(s3URI, "s3://")

	// 最初の "/" でバケット名とオブジェクトキーに分割
	parts := strings.SplitN(processedURI, "/", 2)
	bucketName := parts[0]
	objectKey := ""

	if len(parts) > 1 {
		objectKey = parts[1]
	}

	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, region, objectKey)
}

// createReviewData は設定とレビュー結果から publisher.ReviewData を生成します。
func createReviewData(reviewConfig config.ReviewConfig, reviewResult string) publisher.ReviewData {
	return publisher.ReviewData{
		RepoURL:        reviewConfig.RepoURL,
		BaseBranch:     reviewConfig.BaseBranch,
		FeatureBranch:  reviewConfig.FeatureBranch,
		ReviewMarkdown: reviewResult,
	}
}
