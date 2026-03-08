package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/shouni/gemini-reviewer-core/pkg/domain"
	"github.com/shouni/go-remote-io/pkg/remoteio"

	"git-gemini-cli/internal/adapters"
	"git-gemini-cli/internal/config"
)

const (
	// signedURLExpiration は署名付きURLの有効期限を定義します。
	signedURLExpiration = 30 * time.Minute
)

// PublisherRunner は、レビュー結果の公開処理を実行する責務を持つインターフェースです。
type PublisherRunner interface {
	Run(ctx context.Context, cfg config.PublishConfig, reviewResult string) error
}

// DefaultPublisherRunner は、レビュー結果の公開処理を実行する具象構造体です。
type DefaultPublisherRunner struct {
	publisher     domain.Publisher
	urlSigner     remoteio.URLSigner
	slackNotifier adapters.SlackNotifier
}

// NewDefaultPublisherRunner は DefaultPublisherRunner の新しいインスタンスを作成します。
func NewDefaultPublisherRunner(publisher domain.Publisher, urlSigner remoteio.URLSigner, slackNotifier adapters.SlackNotifier) *DefaultPublisherRunner {
	return &DefaultPublisherRunner{
		publisher:     publisher,
		urlSigner:     urlSigner,
		slackNotifier: slackNotifier,
	}
}

// Run は公開処理のパイプライン全体を実行します。
func (p *DefaultPublisherRunner) Run(ctx context.Context, cfg config.PublishConfig, reviewResult string) error {
	// 1. ストレージへのアップロード処理
	if err := p.publishToStorage(ctx, cfg, reviewResult); err != nil {
		return err
	}

	// 2. 公開URLの生成 (Slack通知の前に行う)
	publicURL, err := p.getPublicURL(ctx, cfg.StorageURI)
	if err != nil {
		slog.Warn("公開URLの生成に失敗しました。署名なし/静的URIで通知を試みます。", "error", err, "uri", cfg.StorageURI)
		publicURL = cfg.StorageURI
	}

	// 3. Slack通知処理
	p.notifyToSlack(ctx, publicURL, cfg)

	return nil
}

// --- プライベートメソッド ---

// publishToStorage はレビュー結果をクラウドストレージにアップロードします。
func (p *DefaultPublisherRunner) publishToStorage(ctx context.Context, cfg config.PublishConfig, reviewResult string) error {
	meta := createReviewData(cfg.ReviewConfig, reviewResult)
	if err := p.publisher.Publish(ctx, cfg.StorageURI, meta); err != nil {
		return fmt.Errorf("ストレージへの書き込みに失敗しました (URI: %s): %w", cfg.StorageURI, err)
	}
	slog.Info("クラウドストレージへのアップロードが完了しました。", "uri", cfg.StorageURI)
	return nil
}

// notifyToSlack はSlackに通知を送信します。
func (p *DefaultPublisherRunner) notifyToSlack(ctx context.Context, publicURL string, cfg config.PublishConfig) {
	if err := p.slackNotifier.Notify(ctx, publicURL, cfg.StorageURI, cfg.ReviewConfig); err != nil {
		slog.Error("Slack通知の実行中にエラーが発生しましたが、処理を続行します。", "error", err)
	}
}

// getPublicURL は URI に応じて署名付きURLを生成するか、公開URLに変換します。
func (p *DefaultPublisherRunner) getPublicURL(ctx context.Context, storageURI string) (string, error) {
	if remoteio.IsGCSURI(storageURI) {
		if p.urlSigner == nil {
			return "", fmt.Errorf("GCS URIが指定されましたが、URL Signerが利用不可です。")
		}
		signedURL, err := p.urlSigner.GenerateSignedURL(ctx, storageURI, "GET", signedURLExpiration)
		if err != nil {
			return "", fmt.Errorf("GCS 署名付きURLの生成に失敗しました: %w", err)
		}
		return signedURL, nil
	}

	if remoteio.IsS3URI(storageURI) {
		awsRegion := os.Getenv("AWS_REGION")
		if awsRegion == "" {
			awsRegion = "ap-northeast-1"
		}
		return convertS3URIToPublicURL(storageURI, awsRegion), nil
	}

	return storageURI, nil
}

// convertS3URIToPublicURL は S3 URI を AWS の公開 Virtual-Hosted Style アクセス URL に変換します。
// 形式: https://{bucketName}.s3.{region}.amazonaws.com/{objectKey}
func convertS3URIToPublicURL(s3URI, region string) string {
	processedURI := strings.TrimPrefix(s3URI, "s3://")
	parts := strings.SplitN(processedURI, "/", 2)
	bucketName := parts[0]
	objectKey := "/"
	if len(parts) > 1 {
		objectKey = "/" + parts[1]
	}

	// 公開URL形式に再構成 (Path-Style Access)
	// 形式: https://s3.{region}.amazonaws.com/{bucketName}{objectKey}
	return fmt.Sprintf("https://s3.%s.amazonaws.com/%s%s", region, bucketName, objectKey)
}

// createReviewData は設定とレビュー結果から publisher.ReviewData を生成します。
func createReviewData(reviewConfig config.ReviewConfig, reviewResult string) domain.ReviewData {
	return domain.ReviewData{
		RepoURL:        reviewConfig.RepoURL,
		BaseBranch:     reviewConfig.BaseBranch,
		FeatureBranch:  reviewConfig.FeatureBranch,
		ReviewMarkdown: reviewResult,
	}
}
