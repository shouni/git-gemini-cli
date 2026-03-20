package runner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/shouni/gemini-reviewer-core/pkg/ports"
	"github.com/shouni/go-remote-io/pkg/remoteio"

	"git-gemini-cli/internal/domain"
)

const (
	// signedURLExpiration は署名付きURLの有効期限を定義します。
	signedURLExpiration = 30 * time.Minute
)

// PublisherRunner は、レビュー結果の公開処理を実行する具象構造体です。
type PublisherRunner struct {
	publisher ports.Publisher
	urlSigner remoteio.URLSigner
	notifier  domain.Notifier
}

// NewPublisherRunner は PublisherRunner の新しいインスタンスを作成します。
func NewPublisherRunner(publisher ports.Publisher, urlSigner remoteio.URLSigner, notifier domain.Notifier) *PublisherRunner {
	return &PublisherRunner{
		publisher: publisher,
		urlSigner: urlSigner,
		notifier:  notifier,
	}
}

// Run は公開処理のパイプライン全体を実行します。
func (p *PublisherRunner) Run(ctx context.Context, req domain.ReviewRequest) error {
	// 1. ストレージへのアップロード処理
	if err := p.publishToStorage(ctx, req); err != nil {
		return err
	}

	// 2. 公開URLの生成
	publicURL, err := p.getPublicURL(ctx, req.StorageURI)
	if err != nil {
		slog.Warn("公開URLの生成に失敗しました。署名なし/静的URIで通知を試みます。", "error", err, "uri", req.StorageURI)
		publicURL = req.StorageURI
	}

	// 3. 通知処理
	p.notify(ctx, publicURL, req)

	return nil
}

// --- プライベートメソッド ---

// publishToStorage はレビュー結果をクラウドストレージにアップロードします。
func (p *PublisherRunner) publishToStorage(ctx context.Context, req domain.ReviewRequest) error {
	meta := createReviewData(req)
	if err := p.publisher.Publish(ctx, req.StorageURI, meta); err != nil {
		return fmt.Errorf("ストレージへの書き込みに失敗しました (URI: %s): %w", req.StorageURI, err)
	}
	slog.Info("クラウドストレージへのアップロードが完了しました。", "uri", req.StorageURI)
	return nil
}

// notify はSlackに通知を送信します。
func (p *PublisherRunner) notify(ctx context.Context, publicURL string, req domain.ReviewRequest) {
	if err := p.notifier.Notify(ctx, publicURL, req.StorageURI, req); err != nil {
		slog.Error("Slack通知の実行中にエラーが発生しましたが、処理を続行します。", "error", err)
	}
}

// getPublicURL は URI に応じて署名付きURLを生成するか、公開URLに変換します。
func (p *PublisherRunner) getPublicURL(ctx context.Context, storageURI string) (string, error) {
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
func createReviewData(req domain.ReviewRequest) ports.ReviewData {
	return ports.ReviewData{
		RepoURL:        req.Config.RepoURL,
		BaseBranch:     req.Config.BaseBranch,
		FeatureBranch:  req.Config.FeatureBranch,
		ReviewMarkdown: req.ReviewMarkdown,
	}
}
