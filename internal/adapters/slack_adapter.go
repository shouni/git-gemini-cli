package adapters

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-notifier/pkg/slack"
	"github.com/shouni/go-utils/urlpath"

	"git-gemini-cli/internal/config"
)

// SlackAdapter は SlackNotifier インターフェースを満たす具象型です。
type SlackAdapter struct {
	webhookURL  string // Webhook URLを保持
	slackClient *slack.Client
}

// NewSlackAdapter は新しいアダプターインスタンスを作成します。
func NewSlackAdapter(httpClient httpkit.RequestExecutor, webhookURL string) (*SlackAdapter, error) {
	if webhookURL == "" {
		// オプショナル機能として扱い、空のままインスタンスを返す
		return &SlackAdapter{}, nil
	}

	if httpClient == nil {
		return nil, errors.New("http client cannot be nil")
	}

	client, err := slack.NewClient(httpClient, webhookURL)
	if err != nil {
		return nil, fmt.Errorf("Slackクライアントの初期化に失敗しました: %w", err)
	}

	return &SlackAdapter{
		webhookURL:  webhookURL,
		slackClient: client,
	}, nil
}

// Notify は Slack への通知を実行します。
// publicURL をリンク先として、Slack に投稿します。
func (s *SlackAdapter) Notify(ctx context.Context, publicURL, storageURI string, cfg config.ReviewConfig) error {
	if s.webhookURL == "" || s.slackClient == nil {
		slog.Info("Slack通知が無効化されているか、クライアントが未初期化のためスキップします。", "storage_uri", storageURI)
		return nil
	}

	title := "✅ AIコードレビュー結果がアップロードされました。"
	content := s.buildSlackContent(publicURL, storageURI, cfg)

	if err := s.slackClient.SendTextWithHeader(ctx, title, content); err != nil {
		return fmt.Errorf("Slackへの結果URL投稿に失敗しました: %w", err)
	}

	slog.Info("レビュー結果のURLを Slack に投稿しました。", "public_url", publicURL)
	return nil
}

// buildSlackContent は投稿メッセージの本文を組み立てます。
func (s *SlackAdapter) buildSlackContent(publicURL, storageURI string, cfg config.ReviewConfig) string {
	repoPath := urlpath.GetRepositoryPath(cfg.RepoURL)
	content := fmt.Sprintf(
		"*詳細URL:* <%s|%s>\n"+
			"*リポジトリ:* `%s`\n"+
			"*ブランチ:* `%s` ← `%s`\n"+
			"*モード:* `%s`\n"+
			"*モデル:* `%s`",
		publicURL,
		storageURI,
		repoPath,
		cfg.BaseBranch,
		cfg.FeatureBranch,
		cfg.ReviewMode,
		cfg.GeminiModel,
	)
	return strings.TrimSpace(content)
}
