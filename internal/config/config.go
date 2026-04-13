package config

import (
	"strings"
	"time"

	"github.com/shouni/go-utils/envutil"
)

const (
	DefaultHTTPTimeout  = 30 * time.Second
	SignedURLExpiration = 30 * time.Minute
)

// Config はAIコードレビューに必要なすべての設定を含みます。
// この構造体は、コマンドライン引数からサービスロジックへ設定を渡すための共通のデータモデルです。
type Config struct {
	ReviewMode       string
	GeminiModel      string
	RepoURL          string
	BaseBranch       string
	FeatureBranch    string
	SSHKeyPath       string
	LocalPath        string
	SkipHostKeyCheck bool
	UseExternalGit   bool
	ProjectID        string
	GeminiAPIKey     string
	GCSBucket        string
	SlackWebhookURL  string
}

// Normalize は設定値の文字列フィールドから前後の空白を一括で削除します。
func (c *Config) Normalize() {
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
	c.SlackWebhookURL = strings.TrimSpace(c.SlackWebhookURL)
	c.GCSBucket = strings.TrimSpace(c.GCSBucket)
}

// FillDefaults は、現在の設定で空のフィールドを envCfg の値で補完します。
func (c *Config) FillDefaults(envCfg *Config) {
	if c.ProjectID == "" {
		c.ProjectID = envCfg.ProjectID
	}
	if c.GeminiAPIKey == "" {
		c.GeminiAPIKey = envCfg.GeminiAPIKey
	}
	if c.SlackWebhookURL == "" {
		c.SlackWebhookURL = envCfg.SlackWebhookURL
	}
}

// LoadConfig は環境変数から設定を読み込みます。
func LoadConfig() *Config {
	return &Config{
		ProjectID:       envutil.GetEnv("GCP_PROJECT_ID", ""),
		GeminiAPIKey:    envutil.GetEnv("GEMINI_API_KEY", ""),
		SlackWebhookURL: envutil.GetEnv("SLACK_WEBHOOK_URL", ""),
	}
}
