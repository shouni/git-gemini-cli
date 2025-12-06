package adapters

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"git-gemini-cli/internal/config"

	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-notifier/pkg/factory"
	"github.com/shouni/go-remote-io/pkg/remoteio"
	"github.com/shouni/go-utils/urlpath"
)

// --- 定数と内部構造体 ---

const (
	// signedURLExpiration は署名付きURLの有効期限を定義します。
	signedURLExpiration = 30 * time.Minute
)

// SlackNotifier は Slack への通知機能を提供する契約を定義します。
type SlackNotifier interface {
	Notify(ctx context.Context, targetURI string, cfg config.ReviewConfig) error
}

// --- 具象アダプター ---

// SlackAdapter は SlackNotifier インターフェースを満たす具象型です。
// 署名付きURL生成のために FactoryRegistry に依存します。
type SlackAdapter struct {
	httpClient httpkit.ClientInterface
	urlSigner  remoteio.URLSigner
	webhookURL string
}

// NewSlackAdapter は新しいアダプターインスタンスを作成します。
func NewSlackAdapter(httpClient httpkit.ClientInterface, urlSigner remoteio.URLSigner, webhookURL string) *SlackAdapter {
	return &SlackAdapter{
		httpClient: httpClient,
		urlSigner:  urlSigner,
		webhookURL: webhookURL,
	}
}

// Notify は SlackNotifier インターフェースの実装です。
// レビュー結果のURLに署名し、Slackに投稿します。
func (a *SlackAdapter) Notify(ctx context.Context, targetURI string, cfg config.ReviewConfig) error {
	// 1. Slack 認証情報の取得とスキップチェック
	if a.webhookURL == "" {
		slog.Info("SLACK_WEBHOOK_URL が設定されていません。Slack通知をスキップします。", "target_uri", targetURI)
		return nil
	}

	// 2. 署名付きURLの生成
	publicURL, err := a.getPublicURL(ctx, targetURI)
	if err != nil {
		// URL署名は必須ではないため、警告ログを出力し、署名なしのURIで続行する
		slog.Warn("公開URLの生成に失敗しました。署名なしURIで通知を試みます。", "error", err, "uri", targetURI)
		publicURL = targetURI
	}

	// 3. HTTP Clientの取得とSlackクライアントの初期化
	slackClient, err := factory.GetSlackClient(a.httpClient)
	if err != nil {
		return fmt.Errorf("Slackクライアントの初期化に失敗しました: %w", err)
	}

	// 4. Slack に投稿するメッセージを作成
	title := "✅ AIコードレビュー結果がアップロードされました。"
	content := a.buildSlackContent(publicURL, targetURI, cfg)

	// 5. Slack投稿処理を実行
	if err := slackClient.SendTextWithHeader(ctx, title, content); err != nil {
		return fmt.Errorf("Slackへの結果URL投稿に失敗しました: %w", err)
	}

	slog.Info("レビュー結果のURLを Slack に投稿しました。", "uri", targetURI)
	return nil
}

// getPublicURL は URI に応じて署名付きURLを生成するか、公開URLに変換します。
func (a *SlackAdapter) getPublicURL(ctx context.Context, targetURI string) (string, error) {
	if a.urlSigner == nil {
		// urlSignerがnilの場合、URIは署名が必要ないか、サポートされていないスキームです。
		slog.Debug("URL Signerがnilです。静的なURI変換のみを試みます。", "uri", targetURI)
	}

	// GCSの場合: 署名付きURLを生成
	if remoteio.IsGCSURI(targetURI) {
		if a.urlSigner == nil {
			return "", fmt.Errorf("GCS URIが指定されましたが、URL Signerがnilです。")
		}

		signedURL, err := a.urlSigner.GenerateSignedURL(ctx, targetURI, "GET", signedURLExpiration)
		if err != nil {
			return "", fmt.Errorf("GCS 署名付きURLの生成に失敗しました: %w", err)
		}
		slog.Info("GCS 署名付きURLの生成に成功", "url", signedURL)
		return signedURL, nil
	}

	// S3の場合: 静的な公開URL形式に変換
	if remoteio.IsS3URI(targetURI) {
		awsRegion := os.Getenv("AWS_REGION")
		if awsRegion == "" {
			awsRegion = "ap-northeast-1" // フォールバック
		}
		publicURL := convertS3URIToPublicURL(targetURI, awsRegion)
		slog.Info("S3 公開URLへの変換に成功", "url", publicURL)
		return publicURL, nil
	}

	// その他: 署名や変換が不要なURI (例: ローカルファイル、未サポートのプロバイダ)
	slog.Debug("静的な公開URL変換や署名が不要なURIです。", "uri", targetURI)
	return targetURI, nil
}

// buildSlackContent は投稿メッセージの本文を組み立てます。
// publicURL はSlackメッセージ内のリンク先URLとして、targetURI はそのリンクの表示テキストとして使用されます。
func (a *SlackAdapter) buildSlackContent(publicURL, targetURI string, cfg config.ReviewConfig) string {
	repoPath := urlpath.GetRepositoryPath(cfg.RepoURL)
	content := fmt.Sprintf(
		"**詳細URL:** <%s|%s>\n"+
			"**リポジトリ:** `%s`\n"+
			"**ブランチ:** `%s` ← `%s`\n"+
			"**モード:** `%s`\n"+
			"**モデル:** `%s`",
		publicURL,
		targetURI,
		repoPath,
		cfg.BaseBranch,
		cfg.FeatureBranch,
		cfg.ReviewMode,
		cfg.GeminiModel,
	)
	return strings.TrimSpace(content)
}

// --------------------------------------------------------------------------
// ヘルパー関数
// --------------------------------------------------------------------------

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
