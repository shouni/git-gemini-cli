package cmd

import (
	"context"
	"fmt"
	"git-gemini-reviewer-go/internal/config"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/shouni/gemini-reviewer-core/pkg/publisher"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-notifier/pkg/factory"
	"github.com/shouni/go-remote-io/pkg/gcsfactory"
	"github.com/shouni/go-remote-io/pkg/remoteio"
	"github.com/shouni/go-remote-io/pkg/s3factory"

	"github.com/spf13/cobra"
)

const defaultHTTPTimeout = 30 * time.Second

// PublishFlags は GCS/S3 への公開フラグを保持します。
type PublishFlags struct {
	URI         string // 宛先URI (例: gs://bucket/..., s3://bucket/...)
	ContentType string // 保存する際のMIMEタイプ
}

// slackAuthInfo は、Slack投稿に必要な認証情報と投稿情報をカプセル化します。
type slackAuthInfo struct {
	WebhookURL string
	Username   string
	IconEmoji  string
	Channel    string
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

	// --- 4. Slack通知 ---
	publicURL := targetURI
	// GCSクライアントの直接初期化を削除し、Factory経由でURLSignerを取得
	if remoteio.IsGCSURI(targetURI) {
		urlSigner, err := registry.GCSFactory.NewGCSURLSigner()
		if err != nil {
			slog.Error("URLSigner の取得に失敗", "error", err)
		}

		// 抽象化されたインターフェースを経由して署名付きURLを生成
		const signedURLExpiration = 15 * time.Minute
		publicURL, err := urlSigner.GenerateSignedURL(
			ctx,
			targetURI,
			"GET",
			signedURLExpiration,
		)
		if err != nil {
			slog.Error("署名付きURLの生成に失敗", "error", err)
		}
		slog.Info("署名付きURLの生成に成功", "url", publicURL)

	} else if remoteio.IsS3URI(targetURI) {
		const defaultAWSRegion = "ap-northeast-1"
		// S3の公開URL形式に変換
		publicURL = convertS3URIToPublicURL(targetURI, defaultAWSRegion)
	}

	// ロジックを分離した関数へ委譲
	if err := sendSlackNotification(ctx, publicURL, ReviewConfig); err != nil {
		// 🚨 ポリシー: Slack通知は二次的な機能であるため、アップロード成功後はエラーを返さない。
		slog.Error("Slack通知の実行中にエラーが発生しましたが、アップロードは成功しているため処理を続行します。", "error", err)
	}

	return nil
}

// --------------------------------------------------------------------------
// プライベート関数 (ロジック分離)
// --------------------------------------------------------------------------

// sendSlackNotification は Slack 通知を送信します。
func sendSlackNotification(ctx context.Context, publicURL string, cfg config.ReviewConfig) error {
	// 1. Slack 認証情報の取得
	slackAuthInfo := getSlackAuthInfo()

	// Webhook URLが設定されていない場合はSlack通知をスキップ
	if slackAuthInfo.WebhookURL == "" {
		slog.Info("SLACK_WEBHOOK_URL が設定されていません。Slack通知をスキップします。")
		return nil
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
		publicURL,
		repoPath,
		cfg.BaseBranch,
		cfg.FeatureBranch,
		cfg.ReviewMode,
		cfg.GeminiModel,
	)

	// 4. HTTP Clientの取得
	httpClient := httpkit.New(defaultHTTPTimeout)

	// 5. Slackクライアントの初期化
	slackClient, err := factory.GetSlackClient(httpClient)
	if err != nil {
		return fmt.Errorf("Slackクライアントの初期化に失敗しました: %w", err)
	}

	// 6. Slack投稿処理を実行
	if err := slackClient.SendTextWithHeader(ctx, title, content); err != nil {
		return fmt.Errorf("Slackへの結果URL投稿に失敗しました: %w", err)
	}

	slog.Info("レビュー結果のURLを Slack に投稿しました。", "uri", publicURL)
	return nil
}

// --------------------------------------------------------------------------
// ヘルパー関数
// --------------------------------------------------------------------------

// getSlackAuthInfo は、環境変数から Slack 認証情報を取得します。
func getSlackAuthInfo() slackAuthInfo {
	return slackAuthInfo{
		WebhookURL: os.Getenv("SLACK_WEBHOOK_URL"),
		Username:   os.Getenv("SLACK_USERNAME"),
		IconEmoji:  os.Getenv("SLACK_ICON_EMOJI"),
		Channel:    os.Getenv("SLACK_CHANNEL"),
	}
}

// getRepositoryPath はリポジトリURLから 'owner/repo-name' の形式のパスを抽出します。
func getRepositoryPath(repoURL string) string {
	s := repoURL

	// 1. プロトコルとプレフィックスの除去 (git@, https://, http://, ssh://)
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "git@")
	s = strings.TrimPrefix(s, "ssh://")

	// 2. Backlog/SSH形式: ホスト名とパスの区切り (:) を利用してパスを抽出
	// 例: bbmf@bbmf.git.backlog.jp:/MK/TEST -> /MK/TEST
	if idx := strings.Index(s, ":"); idx != -1 {
		// ':' 以降の部分をパスとする
		s = s[idx+1:]
	} else {
		// 3. HTTPS形式など、ホスト名とパスが '/' で区切られている形式に対応
		// 例: github.com/owner/repo-name -> owner/repo-name
		if idx := strings.Index(s, "/"); idx != -1 {
			s = s[idx+1:]
		}
	}

	// 4. パスの先頭にある可能性のある '/' を完全に除去
	// 上記 2. で抽出されたパスが "/MK/TEST" の場合に対応
	s = strings.TrimPrefix(s, "/")

	// 5. .git 拡張子を除去
	s = strings.TrimSuffix(s, ".git")

	return s
}

// convertS3URIToPublicURL は S3 URI を AWS の公開 Path-Style アクセス URL に変換します。
func convertS3URIToPublicURL(s3URI, region string) string {
	processedURI := strings.TrimPrefix(s3URI, "s3://")

	// 最初の "/" でバケット名とオブジェクトキーに分割
	parts := strings.SplitN(processedURI, "/", 2)
	bucketName := parts[0]
	objectKey := "/"

	if len(parts) > 1 {
		objectKey = "/" + parts[1]
	}

	// 公開URL形式に再構成 (Path-Style Access)
	// 形式: https://s3.{region}.amazonaws.com/{bucketName}{objectKey}
	publicURL := fmt.Sprintf("https://s3.%s.amazonaws.com/%s%s", region, bucketName, objectKey)
	return publicURL
}
