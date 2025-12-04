package adapters

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"time"

	"git-gemini-reviewer-go/internal/config"

	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-notifier/pkg/factory"
	"github.com/shouni/go-remote-io/pkg/remoteio"
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
	publicURL := targetURI
	// GCSクライアントの直接初期化を削除し、Factory経由でURLSignerを取得
	if remoteio.IsGCSURI(targetURI) {
		signedURL, err := a.urlSigner.GenerateSignedURL(
			ctx,
			targetURI,
			"GET",
			signedURLExpiration,
		)
		if err != nil {
			slog.Error("署名付きURLの生成に失敗", "error", err)
			// エラーが発生した場合、publicURL は targetURI のままとなる。
		} else {
			publicURL = signedURL
			slog.Info("署名付きURLの生成に成功", "url", publicURL)
		}
	} else if remoteio.IsS3URI(targetURI) {
		awsRegion := os.Getenv("AWS_REGION")
		if awsRegion == "" {
			awsRegion = "ap-northeast-1" // フォールバック
		}
		// S3の公開URL形式に変換
		publicURL = convertS3URIToPublicURL(targetURI, awsRegion)
	}

	// リポジトリ名を抽出
	repoPath := getRepositoryPath(cfg.RepoURL)

	// 3. Slack に投稿するメッセージを作成
	title := "✅ AIコードレビュー結果がアップロードされました。"
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
	content = strings.TrimSpace(content)

	// 5. HTTP Clientの取得とSlackクライアントの初期化
	slackClient, err := factory.GetSlackClient(a.httpClient)
	if err != nil {
		return fmt.Errorf("Slackクライアントの初期化に失敗しました: %w", err)
	}

	// 6. Slack投稿処理を実行
	if err := slackClient.SendTextWithHeader(ctx, title, content); err != nil {
		return fmt.Errorf("Slackへの結果URL投稿に失敗しました: %w", err)
	}

	slog.Info("レビュー結果のURLを Slack に投稿しました。", "uri", targetURI)
	return nil
}

// --------------------------------------------------------------------------
// ヘルパー関数
// --------------------------------------------------------------------------

// getRepositoryPath はリポジトリURLから 'owner/repo-name' の形式のパスを抽出します。
func getRepositoryPath(repoURL string) string {
	// SSH形式 (git@host:owner/repo.git) を net/url でパース可能な形式に変換
	if strings.HasPrefix(repoURL, "git@") {
		if idx := strings.Index(repoURL, ":"); idx != -1 {
			repoURL = "ssh://" + repoURL[:idx] + "/" + repoURL[idx+1:] // ':' を '/' に置換
		}
	}

	u, err := url.Parse(repoURL)
	if err != nil {
		slog.Warn("リポジトリURLのパースに失敗しました。元のURLをそのまま使用します。", "url", repoURL, "error", err)
		return repoURL // パース失敗時は元のURLを返す
	}

	// パス部分から先頭の '/' と末尾の '.git' を除去
	path := strings.TrimPrefix(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")

	return path
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
