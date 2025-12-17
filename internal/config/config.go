package config

import (
	"strings"

	"github.com/shouni/go-http-kit/pkg/httpkit"
)

// ReviewConfig はAIコードレビューに必要なすべての設定を含みます。
// この構造体は、コマンドライン引数からサービスロジックへ設定を渡すための共通のデータモデルです。
type ReviewConfig struct {
	ReviewMode            string
	GeminiModel           string
	RepoURL               string
	BaseBranch            string
	FeatureBranch         string
	SSHKeyPath            string
	LocalPath             string
	SkipHostKeyCheck      bool
	UseExternalGitCommand bool
}

type PublishConfig struct {
	HttpClient      httpkit.ClientInterface
	ReviewConfig    ReviewConfig
	StorageURI      string
	SlackWebhookURL string
}

// Normalize は設定値の文字列フィールドから前後の空白を一括で削除します。
func (c *ReviewConfig) Normalize() {
	if c == nil {
		return
	}
	c.RepoURL = strings.TrimSpace(c.RepoURL)
	c.BaseBranch = strings.TrimSpace(c.BaseBranch)
	c.FeatureBranch = strings.TrimSpace(c.FeatureBranch)
	c.LocalPath = strings.TrimSpace(c.LocalPath)
	c.ReviewMode = strings.TrimSpace(c.ReviewMode)
	c.GeminiModel = strings.TrimSpace(c.GeminiModel)
	c.SSHKeyPath = strings.TrimSpace(c.SSHKeyPath)
}
