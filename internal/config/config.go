package config

import "github.com/shouni/go-http-kit/pkg/httpkit"

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
	TargetURI       string
	SlackWebhookURL string
	ReviewResult    string
}
