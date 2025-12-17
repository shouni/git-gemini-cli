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
func (rc *ReviewConfig) Normalize() {
	if rc == nil {
		return
	}
	rc.RepoURL = strings.TrimSpace(rc.RepoURL)
	rc.BaseBranch = strings.TrimSpace(rc.BaseBranch)
	rc.FeatureBranch = strings.TrimSpace(rc.FeatureBranch)
	rc.LocalPath = strings.TrimSpace(rc.LocalPath)
	rc.ReviewMode = strings.TrimSpace(rc.ReviewMode)
	rc.GeminiModel = strings.TrimSpace(rc.GeminiModel)
	rc.SSHKeyPath = strings.TrimSpace(rc.SSHKeyPath)
}
